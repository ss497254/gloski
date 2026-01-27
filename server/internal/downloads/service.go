package downloads

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/database"
)

// Config holds configuration for the download service
type Config struct {
	MaxConcurrent int    // Maximum concurrent downloads
	MaxRetries    int    // Maximum retry attempts
	BaseURL       string // Base URL for share links
}

// Service manages downloads
type Service struct {
	config     Config
	store      *Store
	downloader *Downloader
	downloads  map[string]*Download
	mu         sync.RWMutex

	// Active download management
	activeDownloads map[string]context.CancelFunc
	activeMu        sync.Mutex
	queue           chan string
	wg              sync.WaitGroup

	// Speed tracking
	speedTrackers map[string]*speedTracker
	speedMu       sync.Mutex

	// Shutdown flag
	shuttingDown bool
	shutdownMu   sync.RWMutex
}

// speedTracker tracks download speed using a sliding window
type speedTracker struct {
	samples    []speedSample
	maxSamples int
}

type speedSample struct {
	bytes     int64
	timestamp time.Time
}

func newSpeedTracker() *speedTracker {
	return &speedTracker{
		samples:    make([]speedSample, 0, 10),
		maxSamples: 10,
	}
}

func (st *speedTracker) addSample(bytes int64) {
	now := time.Now()
	st.samples = append(st.samples, speedSample{bytes: bytes, timestamp: now})

	// Keep only recent samples (last 5 seconds)
	cutoff := now.Add(-5 * time.Second)
	newSamples := make([]speedSample, 0, len(st.samples))
	for _, s := range st.samples {
		if s.timestamp.After(cutoff) {
			newSamples = append(newSamples, s)
		}
	}
	st.samples = newSamples
}

func (st *speedTracker) getSpeed() int64 {
	if len(st.samples) < 2 {
		return 0
	}

	first := st.samples[0]
	last := st.samples[len(st.samples)-1]
	duration := last.timestamp.Sub(first.timestamp).Seconds()
	if duration <= 0 {
		return 0
	}

	bytesTransferred := last.bytes - first.bytes
	return int64(float64(bytesTransferred) / duration)
}

// NewService creates a new download service
func NewService(db *database.Database, config Config) (*Service, error) {
	// Ensure sensible defaults
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 3
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	store := NewStore(db)

	s := &Service{
		config:          config,
		store:           store,
		downloader:      NewDownloader(),
		downloads:       make(map[string]*Download),
		activeDownloads: make(map[string]context.CancelFunc),
		queue:           make(chan string, 100),
		speedTrackers:   make(map[string]*speedTracker),
	}

	// Load existing downloads from database
	downloads, err := store.Load()
	if err != nil {
		// Log but don't fail - start fresh
		downloads = []*Download{}
	}

	for _, d := range downloads {
		s.downloads[d.ID] = d
		// Reset downloading status to paused (server restarted)
		if d.Status == StatusDownloading {
			d.Status = StatusPaused
			store.UpdateStatus(d.ID, StatusPaused, "")
		}
	}

	// Start worker pool
	for i := 0; i < config.MaxConcurrent; i++ {
		s.wg.Add(1)
		go s.worker()
	}

	// Resume pending downloads (non-blocking, use select)
	go func() {
		for id, d := range s.downloads {
			if d.Status == StatusPending {
				select {
				case s.queue <- id:
				default:
					// Queue full, will be picked up later
				}
			}
		}
	}()

	return s, nil
}

// worker processes downloads from the queue
func (s *Service) worker() {
	defer s.wg.Done()

	for id := range s.queue {
		s.processDownload(id)
	}
}

// isShuttingDown returns true if the service is shutting down
func (s *Service) isShuttingDown() bool {
	s.shutdownMu.RLock()
	defer s.shutdownMu.RUnlock()
	return s.shuttingDown
}

// processDownload handles a single download
func (s *Service) processDownload(id string) {
	// Check if shutting down
	if s.isShuttingDown() {
		return
	}

	s.mu.Lock()
	download, exists := s.downloads[id]
	if !exists || download.Status != StatusPending {
		s.mu.Unlock()
		return
	}

	download.Status = StatusDownloading
	now := time.Now()
	download.StartedAt = &now
	s.mu.Unlock()

	// Persist status change
	s.store.Update(download)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	s.activeMu.Lock()
	s.activeDownloads[id] = cancel
	s.activeMu.Unlock()

	// Initialize speed tracker
	s.speedMu.Lock()
	s.speedTrackers[id] = newSpeedTracker()
	s.speedMu.Unlock()

	defer func() {
		s.activeMu.Lock()
		delete(s.activeDownloads, id)
		s.activeMu.Unlock()

		s.speedMu.Lock()
		delete(s.speedTrackers, id)
		s.speedMu.Unlock()
	}()

	// Get resume position
	resumeFrom := GetPartialFileSize(download.Destination, download.Filename)
	if resumeFrom > 0 {
		s.mu.Lock()
		download.Progress = resumeFrom
		s.mu.Unlock()
	}

	// Progress callback with throttling to reduce lock contention
	var lastProgressUpdate time.Time
	const progressUpdateInterval = 100 * time.Millisecond
	onProgress := func(downloaded int64) {
		now := time.Now()
		if now.Sub(lastProgressUpdate) < progressUpdateInterval {
			return // Throttle updates
		}
		lastProgressUpdate = now

		s.speedMu.Lock()
		var speed int64
		if tracker, ok := s.speedTrackers[id]; ok {
			tracker.addSample(downloaded)
			speed = tracker.getSpeed()
		}
		s.speedMu.Unlock()

		s.mu.Lock()
		download.Progress = downloaded
		download.Speed = speed
		s.mu.Unlock()
	}

	// Execute download
	result, err := s.downloader.Download(ctx, download.URL, download.Destination, download.Filename, resumeFrom, onProgress)

	s.mu.Lock()
	if err != nil {
		if ctx.Err() == context.Canceled {
			// Check if paused or cancelled
			if download.Status == StatusPaused || download.Status == StatusCancelled {
				s.mu.Unlock()
				s.store.Update(download)
				return
			}
		}

		download.Status = StatusFailed
		download.Error = err.Error()
		download.Speed = 0

		// Auto-retry if not cancelled
		if download.Retries < download.MaxRetries {
			download.Retries++
			download.Status = StatusPending
			download.Error = ""
			s.mu.Unlock()
			s.store.Update(download)
			go func() { s.queue <- id }()
			return
		}
	} else {
		download.Status = StatusCompleted
		download.Progress = result.Size
		download.Total = result.Size
		download.FilePath = result.FilePath
		download.Speed = 0
		now := time.Now()
		download.CompletedAt = &now
	}
	s.mu.Unlock()

	s.store.Update(download)
}

// Add adds a new download to the queue
func (s *Service) Add(url, destination, filename string) (*Download, error) {
	// Expand ~ to home directory
	if len(destination) > 0 && destination[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			destination = filepath.Join(home, destination[1:])
		}
	}

	// Fetch metadata if filename is empty
	if filename == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var err error
		filename, _, _, err = s.downloader.FetchMetadata(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch metadata: %w", err)
		}
	}

	download := &Download{
		ID:          uuid.New().String(),
		URL:         url,
		Destination: destination,
		Filename:    filename,
		FilePath:    filepath.Join(destination, filename),
		Status:      StatusPending,
		Progress:    0,
		Total:       -1, // Unknown until download starts
		Speed:       0,
		CreatedAt:   time.Now(),
		MaxRetries:  s.config.MaxRetries,
		ShareLinks:  []ShareLink{},
	}

	// Insert into database
	if err := s.store.Insert(download); err != nil {
		return nil, fmt.Errorf("failed to save download: %w", err)
	}

	// Fetch total size in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, size, _, err := s.downloader.FetchMetadata(ctx, url)
		if err == nil && size > 0 {
			s.mu.Lock()
			if d, ok := s.downloads[download.ID]; ok {
				d.Total = size
			}
			s.mu.Unlock()
			s.store.UpdateProgress(download.ID, download.Progress, size, 0)
		}
	}()

	s.mu.Lock()
	s.downloads[download.ID] = download
	s.mu.Unlock()

	// Add to queue with backpressure handling
	select {
	case s.queue <- download.ID:
		// Successfully queued
	default:
		// Queue is full - return error instead of blocking
		s.mu.Lock()
		delete(s.downloads, download.ID)
		s.mu.Unlock()
		s.store.Delete(download.ID)
		return nil, fmt.Errorf("download queue is full, please try again later")
	}

	return download, nil
}

// List returns all downloads
func (s *Service) List() []*Download {
	s.mu.RLock()
	defer s.mu.RUnlock()

	downloads := make([]*Download, 0, len(s.downloads))
	for _, d := range s.downloads {
		downloads = append(downloads, d)
	}
	return downloads
}

// Get returns a download by ID
func (s *Service) Get(id string) (*Download, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	download, exists := s.downloads[id]
	if !exists {
		return nil, fmt.Errorf("download not found")
	}
	return download, nil
}

// Pause pauses a download
func (s *Service) Pause(id string) error {
	s.mu.Lock()
	download, exists := s.downloads[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("download not found")
	}

	if !download.CanPause() {
		s.mu.Unlock()
		return fmt.Errorf("download cannot be paused")
	}

	download.Status = StatusPaused
	download.Speed = 0
	s.mu.Unlock()

	// Cancel active download
	s.activeMu.Lock()
	if cancel, ok := s.activeDownloads[id]; ok {
		cancel()
	}
	s.activeMu.Unlock()

	s.store.Update(download)
	return nil
}

// Resume resumes a paused download
func (s *Service) Resume(id string) error {
	s.mu.Lock()
	download, exists := s.downloads[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("download not found")
	}

	if !download.CanResume() {
		s.mu.Unlock()
		return fmt.Errorf("download cannot be resumed")
	}

	download.Status = StatusPending
	s.mu.Unlock()

	s.store.Update(download)
	s.queue <- id
	return nil
}

// Cancel cancels a download
func (s *Service) Cancel(id string) error {
	s.mu.Lock()
	download, exists := s.downloads[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("download not found")
	}

	if !download.CanCancel() {
		s.mu.Unlock()
		return fmt.Errorf("download cannot be cancelled")
	}

	download.Status = StatusCancelled
	download.Speed = 0
	s.mu.Unlock()

	// Cancel active download
	s.activeMu.Lock()
	if cancel, ok := s.activeDownloads[id]; ok {
		cancel()
	}
	s.activeMu.Unlock()

	// Cleanup partial file
	CleanupPartialFile(download.Destination, download.Filename)

	s.store.Update(download)
	return nil
}

// Retry retries a failed download
func (s *Service) Retry(id string) error {
	s.mu.Lock()
	download, exists := s.downloads[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("download not found")
	}

	if !download.CanRetry() {
		s.mu.Unlock()
		return fmt.Errorf("download cannot be retried")
	}

	download.Status = StatusPending
	download.Error = ""
	download.Retries = 0
	download.Progress = 0
	s.mu.Unlock()

	// Cleanup partial file for fresh start
	CleanupPartialFile(download.Destination, download.Filename)

	s.store.Update(download)
	s.queue <- id
	return nil
}

// Delete removes a download and optionally its file
func (s *Service) Delete(id string, deleteFile bool) error {
	s.mu.Lock()
	download, exists := s.downloads[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("download not found")
	}

	// Cancel if active
	if download.IsActive() {
		download.Status = StatusCancelled
	}
	s.mu.Unlock()

	// Cancel active download
	s.activeMu.Lock()
	if cancel, ok := s.activeDownloads[id]; ok {
		cancel()
	}
	s.activeMu.Unlock()

	// Delete files
	if deleteFile && download.Status == StatusCompleted {
		os.Remove(download.FilePath)
	}
	CleanupPartialFile(download.Destination, download.Filename)

	// Remove from database
	if err := s.store.Delete(id); err != nil {
		return fmt.Errorf("failed to delete from database: %w", err)
	}

	// Remove from map
	s.mu.Lock()
	delete(s.downloads, id)
	s.mu.Unlock()

	return nil
}

// CreateShareLink creates a share link for a download
func (s *Service) CreateShareLink(id string, expiresIn *int) (*ShareLink, error) {
	s.mu.Lock()
	download, exists := s.downloads[id]
	if !exists {
		s.mu.Unlock()
		return nil, fmt.Errorf("download not found")
	}

	if download.Status != StatusCompleted {
		s.mu.Unlock()
		return nil, fmt.Errorf("can only share completed downloads")
	}
	s.mu.Unlock()

	token := uuid.New().String()
	shareLink := ShareLink{
		Token:     token,
		URL:       fmt.Sprintf("%s/api/share/%s", s.config.BaseURL, token),
		CreatedAt: time.Now(),
	}

	if expiresIn != nil && *expiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(*expiresIn) * time.Second)
		shareLink.ExpiresAt = &expiresAt
	}

	// Save to database first - if this fails, we don't update memory
	if err := s.store.InsertShareLink(id, &shareLink); err != nil {
		return nil, fmt.Errorf("failed to save share link: %w", err)
	}

	// Only update in-memory state after successful DB write
	s.mu.Lock()
	if d, ok := s.downloads[id]; ok {
		d.ShareLinks = append(d.ShareLinks, shareLink)
	}
	s.mu.Unlock()

	return &shareLink, nil
}

// RevokeShareLink removes a share link
func (s *Service) RevokeShareLink(id, token string) error {
	s.mu.Lock()
	download, exists := s.downloads[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("download not found")
	}

	if !download.RemoveShareLink(token) {
		s.mu.Unlock()
		return fmt.Errorf("share link not found")
	}
	s.mu.Unlock()

	// Delete from database
	if err := s.store.DeleteShareLink(id, token); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to delete share link: %w", err)
	}

	return nil
}

// GetByShareToken finds a download by share token
func (s *Service) GetByShareToken(token string) (*Download, error) {
	// Try to find in database (this checks expiration too)
	download, _, err := s.store.GetByShareToken(token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired share link")
		}
		return nil, err
	}

	return download, nil
}

// Shutdown gracefully shuts down the service
// timeout specifies how long to wait for workers (0 = use default of 5 seconds)
func (s *Service) Shutdown(timeout time.Duration) {
	// Set shutdown flag
	s.shutdownMu.Lock()
	s.shuttingDown = true
	s.shutdownMu.Unlock()

	// Cancel all active downloads
	s.activeMu.Lock()
	for _, cancel := range s.activeDownloads {
		cancel()
	}
	s.activeMu.Unlock()

	// Close queue - workers will exit when queue is drained
	close(s.queue)

	// Use default timeout if not specified
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Workers finished gracefully
	case <-time.After(timeout):
		// Timeout - workers will be abandoned
	}

	// Update status of active downloads to paused in database
	s.mu.Lock()
	for _, d := range s.downloads {
		if d.Status == StatusDownloading {
			d.Status = StatusPaused
			s.store.UpdateStatus(d.ID, StatusPaused, "")
		}
	}
	s.mu.Unlock()
}
