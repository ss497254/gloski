package downloads

import (
	"time"
)

// DownloadStatus represents the current state of a download
type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusPaused      DownloadStatus = "paused"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
	StatusCancelled   DownloadStatus = "cancelled"
)

// ShareLink represents a shareable link for a download
type ShareLink struct {
	Token     string     `json:"token"`
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Download represents a download task
type Download struct {
	ID          string         `json:"id"`
	URL         string         `json:"url"`
	Destination string         `json:"destination"` // Directory path
	Filename    string         `json:"filename"`
	FilePath    string         `json:"file_path"` // Full path: Destination + Filename
	Status      DownloadStatus `json:"status"`
	Progress    int64          `json:"progress"` // Bytes downloaded
	Total       int64          `json:"total"`    // Total bytes (-1 if unknown)
	Speed       int64          `json:"speed"`    // Bytes per second
	Error       string         `json:"error,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Retries     int            `json:"retries"`
	MaxRetries  int            `json:"max_retries"`
	ShareLinks  []ShareLink    `json:"share_links,omitempty"`
}

// IsActive returns true if the download is currently active
func (d *Download) IsActive() bool {
	return d.Status == StatusDownloading || d.Status == StatusPending
}

// CanPause returns true if the download can be paused
func (d *Download) CanPause() bool {
	return d.Status == StatusDownloading
}

// CanResume returns true if the download can be resumed
func (d *Download) CanResume() bool {
	return d.Status == StatusPaused
}

// CanRetry returns true if the download can be retried
func (d *Download) CanRetry() bool {
	return d.Status == StatusFailed || d.Status == StatusCancelled
}

// CanCancel returns true if the download can be cancelled
func (d *Download) CanCancel() bool {
	return d.Status == StatusDownloading || d.Status == StatusPaused || d.Status == StatusPending
}

// GetValidShareLink returns a share link if the token is valid and not expired
func (d *Download) GetValidShareLink(token string) *ShareLink {
	for i := range d.ShareLinks {
		link := &d.ShareLinks[i]
		if link.Token == token {
			if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
				return nil // Expired
			}
			return link
		}
	}
	return nil
}

// RemoveShareLink removes a share link by token
func (d *Download) RemoveShareLink(token string) bool {
	for i, link := range d.ShareLinks {
		if link.Token == token {
			d.ShareLinks = append(d.ShareLinks[:i], d.ShareLinks[i+1:]...)
			return true
		}
	}
	return false
}

// AddDownloadRequest represents a request to add a new download
type AddDownloadRequest struct {
	URL         string `json:"url"`
	Destination string `json:"destination"`
	Filename    string `json:"filename,omitempty"` // Optional, auto-detect if empty
}

// CreateShareRequest represents a request to create a share link
type CreateShareRequest struct {
	ExpiresIn *int `json:"expires_in,omitempty"` // Seconds, nil = never expires
}
