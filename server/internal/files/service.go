package files

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/config"
)

var (
	ErrPathNotAllowed = errors.New("path not allowed")
	ErrFileTooLarge   = errors.New("file too large")
	ErrBinaryFile     = errors.New("binary file not supported")
	ErrDangerousPath  = errors.New("dangerous path operation")
)

const (
	MaxFileSize = 10 * 1024 * 1024 // 10MB
)

type Service struct {
	config *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
	}
}

type FileEntry struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Type        string    `json:"type"` // "file" or "directory"
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	Permissions string    `json:"permissions"`
}

type ListResponse struct {
	Path    string      `json:"path"`
	Entries []FileEntry `json:"entries"`
}

func (s *Service) validatePath(path string) (string, error) {
	// Clean and resolve to absolute path
	path = filepath.Clean(path)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Check if path is allowed by config
	if !s.config.IsPathAllowed(absPath) {
		return "", ErrPathNotAllowed
	}

	return absPath, nil
}

func (s *Service) List(path string) (*ListResponse, error) {
	return s.ListWithContext(context.Background(), path)
}

func (s *Service) ListWithContext(ctx context.Context, path string) (*ListResponse, error) {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	absPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	fileEntries := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		// Check context periodically for large directories
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		entryType := "file"
		if entry.IsDir() {
			entryType = "directory"
		}

		fileEntries = append(fileEntries, FileEntry{
			Name:        entry.Name(),
			Path:        filepath.Join(absPath, entry.Name()),
			Type:        entryType,
			Size:        info.Size(),
			Modified:    info.ModTime(),
			Permissions: info.Mode().String(),
		})
	}

	return &ListResponse{
		Path:    absPath,
		Entries: fileEntries,
	}, nil
}

func (s *Service) Read(path string) (string, error) {
	absPath, err := s.validatePath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return "", errors.New("cannot read directory")
	}

	if info.Size() > MaxFileSize {
		return "", ErrFileTooLarge
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}

	if isBinary(content) {
		return "", ErrBinaryFile
	}

	return string(content), nil
}

func (s *Service) Write(path, content string) error {
	absPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(absPath, []byte(content), 0644)
}

func (s *Service) Mkdir(path string) error {
	absPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(absPath, 0755)
}

func (s *Service) Delete(path string) error {
	absPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	// Safety checks
	if absPath == "/" {
		return ErrDangerousPath
	}

	homeDir := os.Getenv("HOME")
	if homeDir != "" && absPath == homeDir {
		return ErrDangerousPath
	}

	// Don't allow deleting system directories
	dangerousPaths := []string{"/etc", "/usr", "/bin", "/sbin", "/var", "/boot", "/lib", "/lib64"}
	for _, dp := range dangerousPaths {
		if absPath == dp || strings.HasPrefix(absPath, dp+"/") {
			return ErrDangerousPath
		}
	}

	return os.RemoveAll(absPath)
}

// Rename renames/moves a file or directory
func (s *Service) Rename(oldPath, newPath string) error {
	absOldPath, err := s.validatePath(oldPath)
	if err != nil {
		return err
	}

	absNewPath, err := s.validatePath(newPath)
	if err != nil {
		return err
	}

	// Safety checks - don't allow renaming system paths
	dangerousPaths := []string{"/", "/etc", "/usr", "/bin", "/sbin", "/var", "/boot", "/lib", "/lib64"}
	for _, dp := range dangerousPaths {
		if absOldPath == dp || absNewPath == dp {
			return ErrDangerousPath
		}
	}

	homeDir := os.Getenv("HOME")
	if homeDir != "" && (absOldPath == homeDir || absNewPath == homeDir) {
		return ErrDangerousPath
	}

	// Check if source exists
	if _, err := os.Stat(absOldPath); os.IsNotExist(err) {
		return errors.New("source path does not exist")
	}

	// Check if destination already exists
	if _, err := os.Stat(absNewPath); err == nil {
		return errors.New("destination path already exists")
	}

	return os.Rename(absOldPath, absNewPath)
}

func (s *Service) Stat(path string) (os.FileInfo, error) {
	absPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	return os.Stat(absPath)
}

func (s *Service) Exists(path string) bool {
	absPath, err := s.validatePath(path)
	if err != nil {
		return false
	}

	_, err = os.Stat(absPath)
	return !os.IsNotExist(err)
}

func isBinary(content []byte) bool {
	checkLen := len(content)
	if checkLen > 8000 {
		checkLen = 8000
	}

	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

// Upload saves an uploaded file to the specified directory
func (s *Service) Upload(destPath, filename string, reader io.Reader) error {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return err
	}

	// Ensure destination is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("destination must be a directory")
	}

	// Clean filename to prevent path traversal
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == ".." {
		return errors.New("invalid filename")
	}

	fullPath := filepath.Join(absPath, filename)

	// Validate the full path is also allowed (safety check)
	if _, err := s.validatePath(fullPath); err != nil {
		return err
	}

	// Create the file with explicit permissions
	f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	return err
}

// GetFileInfo returns the absolute path and file info for a given path
func (s *Service) GetFileInfo(path string) (string, os.FileInfo, error) {
	absPath, err := s.validatePath(path)
	if err != nil {
		return "", nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", nil, err
	}

	return absPath, info, nil
}

// SearchResult represents a single search result
type SearchResult struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Size    int64  `json:"size"`
	Match   string `json:"match,omitempty"`    // For content search: the matching line
	LineNum int    `json:"line_num,omitempty"` // For content search: line number
}

// SearchOptions configures search behavior
type SearchOptions struct {
	Timeout  time.Duration // Maximum search duration (default: 30s)
	MaxDepth int           // Maximum directory depth (default: 20)
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Timeout:  30 * time.Second,
		MaxDepth: 20,
	}
}

// Search searches for files by name or content
func (s *Service) Search(path, query string, searchContent bool, limit int) ([]SearchResult, error) {
	return s.SearchWithOptions(context.Background(), path, query, searchContent, limit, DefaultSearchOptions())
}

// SearchWithContext searches for files with a context for cancellation
func (s *Service) SearchWithContext(ctx context.Context, path, query string, searchContent bool, limit int) ([]SearchResult, error) {
	return s.SearchWithOptions(ctx, path, query, searchContent, limit, DefaultSearchOptions())
}

// SearchWithOptions searches for files with custom options
func (s *Service) SearchWithOptions(ctx context.Context, path, query string, searchContent bool, limit int, opts SearchOptions) ([]SearchResult, error) {
	absPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	// Create a context with timeout
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	results := []SearchResult{}
	query = strings.ToLower(query)

	// Track directory depth
	baseDepth := strings.Count(absPath, string(os.PathSeparator))

	err = filepath.Walk(absPath, func(filePath string, info os.FileInfo, err error) error {
		// Check for context cancellation/timeout
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return nil // Skip errors (permission denied, etc.)
		}

		// Check depth limit
		currentDepth := strings.Count(filePath, string(os.PathSeparator)) - baseDepth
		if opts.MaxDepth > 0 && currentDepth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden directories (but not hidden files)
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && filePath != absPath {
			return filepath.SkipDir
		}

		// Skip node_modules and other large directories
		skipDirs := []string{"node_modules", ".git", "vendor", "__pycache__", ".cache", ".chunks"}
		if info.IsDir() {
			for _, sd := range skipDirs {
				if info.Name() == sd {
					return filepath.SkipDir
				}
			}
		}

		if len(results) >= limit {
			return filepath.SkipAll
		}

		// Search by filename
		if strings.Contains(strings.ToLower(info.Name()), query) {
			entryType := "file"
			if info.IsDir() {
				entryType = "directory"
			}
			results = append(results, SearchResult{
				Path: filePath,
				Name: info.Name(),
				Type: entryType,
				Size: info.Size(),
			})
			return nil
		}

		// Search content (only for files, skip binary and large files)
		if searchContent && !info.IsDir() && info.Size() < MaxFileSize {
			matches := s.searchFileContent(filePath, query, 3) // Max 3 matches per file
			for _, match := range matches {
				if len(results) >= limit {
					return filepath.SkipAll
				}
				results = append(results, SearchResult{
					Path:    filePath,
					Name:    info.Name(),
					Type:    "file",
					Size:    info.Size(),
					Match:   match.line,
					LineNum: match.lineNum,
				})
			}
		}

		return nil
	})

	// Don't return timeout/cancel errors as failures
	if err == context.DeadlineExceeded || err == context.Canceled {
		// Return partial results with no error - results are what we found before timeout
		return results, nil
	}

	return results, err
}

type contentMatch struct {
	lineNum int
	line    string
}

// ─────────────────────────────────────────────────────────────────────────────
// Chunked Upload Support
// ─────────────────────────────────────────────────────────────────────────────

// ChunkUploadInfo holds information about a chunked upload session
type ChunkUploadInfo struct {
	UploadID    string `json:"upload_id"`
	Filename    string `json:"filename"`
	Destination string `json:"destination"`
	TotalSize   int64  `json:"total_size"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
}

// InitChunkedUpload creates a new chunked upload session
func (s *Service) InitChunkedUpload(destPath, filename string, totalSize, chunkSize int64) (*ChunkUploadInfo, error) {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return nil, err
	}

	// Ensure destination is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("destination must be a directory")
	}

	// Clean filename
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == ".." {
		return nil, errors.New("invalid filename")
	}

	// Generate upload ID
	uploadID := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filename)

	totalChunks := int((totalSize + chunkSize - 1) / chunkSize)

	return &ChunkUploadInfo{
		UploadID:    uploadID,
		Filename:    filename,
		Destination: absPath,
		TotalSize:   totalSize,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
	}, nil
}

// UploadChunk uploads a single chunk of a file
func (s *Service) UploadChunk(destPath, filename, uploadID string, chunkIndex int, reader io.Reader) error {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return err
	}

	// Create temp directory for chunks if not exists
	chunkDir := filepath.Join(absPath, ".chunks", uploadID)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunk directory: %w", err)
	}

	// Write chunk to temp file
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%06d", chunkIndex))
	f, err := os.Create(chunkPath)
	if err != nil {
		return fmt.Errorf("failed to create chunk file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("failed to write chunk: %w", err)
	}

	return nil
}

// CompleteChunkedUpload assembles all chunks into the final file
func (s *Service) CompleteChunkedUpload(destPath, filename, uploadID string, totalChunks int) error {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return err
	}

	chunkDir := filepath.Join(absPath, ".chunks", uploadID)

	// Verify all chunks exist
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%06d", i))
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			return fmt.Errorf("missing chunk %d", i)
		}
	}

	// Create final file
	finalPath := filepath.Join(absPath, filename)
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return fmt.Errorf("failed to create final file: %w", err)
	}
	defer finalFile.Close()

	// Assemble chunks
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%06d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			finalFile.Close()
			os.Remove(finalPath)
			return fmt.Errorf("failed to open chunk %d: %w", i, err)
		}

		if _, err := io.Copy(finalFile, chunkFile); err != nil {
			chunkFile.Close()
			finalFile.Close()
			os.Remove(finalPath)
			return fmt.Errorf("failed to copy chunk %d: %w", i, err)
		}
		chunkFile.Close()
	}

	// Cleanup chunk directory
	os.RemoveAll(chunkDir)

	return nil
}

// AbortChunkedUpload cleans up an incomplete chunked upload
func (s *Service) AbortChunkedUpload(destPath, uploadID string) error {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return err
	}

	chunkDir := filepath.Join(absPath, ".chunks", uploadID)
	return os.RemoveAll(chunkDir)
}

func (s *Service) searchFileContent(path, query string, maxMatches int) []contentMatch {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	matches := []contentMatch{}
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.Contains(strings.ToLower(line), query) {
			// Truncate long lines
			if len(line) > 200 {
				line = line[:200] + "..."
			}
			matches = append(matches, contentMatch{
				lineNum: lineNum,
				line:    line,
			})
			if len(matches) >= maxMatches {
				break
			}
		}
	}

	return matches
}
