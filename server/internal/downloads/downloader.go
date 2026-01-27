package downloads

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// Downloader handles the actual HTTP download process
type Downloader struct {
	client *http.Client
}

// NewDownloader creates a new downloader with configured HTTP client
func NewDownloader() *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 0, // No timeout for downloads
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  true, // We want to know actual file size
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

// DownloadResult contains the result of a download operation
type DownloadResult struct {
	Filename string
	FilePath string
	Size     int64
	Error    error
}

// ProgressWriter wraps an io.Writer to track download progress
type ProgressWriter struct {
	writer     io.Writer
	downloaded int64
	onProgress func(downloaded int64)
}

// Write implements io.Writer
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if n > 0 {
		atomic.AddInt64(&pw.downloaded, int64(n))
		if pw.onProgress != nil {
			pw.onProgress(atomic.LoadInt64(&pw.downloaded))
		}
	}
	return n, err
}

// FetchMetadata fetches the file metadata (size, filename) without downloading
func (d *Downloader) FetchMetadata(ctx context.Context, downloadURL string) (filename string, size int64, supportsRange bool, err error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", downloadURL, nil)
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Gloski/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		// Fall back to GET if HEAD fails
		return d.fetchMetadataWithGet(ctx, downloadURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return d.fetchMetadataWithGet(ctx, downloadURL)
	}

	filename = extractFilename(resp.Header, downloadURL)
	size = resp.ContentLength
	supportsRange = resp.Header.Get("Accept-Ranges") == "bytes"

	return filename, size, supportsRange, nil
}

// fetchMetadataWithGet fetches metadata using a GET request with Range header
func (d *Downloader) fetchMetadataWithGet(ctx context.Context, downloadURL string) (filename string, size int64, supportsRange bool, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Gloski/1.0")
	req.Header.Set("Range", "bytes=0-0")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	filename = extractFilename(resp.Header, downloadURL)

	// Check if range is supported
	if resp.StatusCode == http.StatusPartialContent {
		supportsRange = true
		// Parse Content-Range header to get total size
		contentRange := resp.Header.Get("Content-Range")
		if contentRange != "" {
			// Format: bytes 0-0/total
			var start, end, total int64
			if _, err := fmt.Sscanf(contentRange, "bytes %d-%d/%d", &start, &end, &total); err == nil {
				size = total
			}
		}
	} else if resp.StatusCode == http.StatusOK {
		size = resp.ContentLength
		supportsRange = false
	} else {
		return "", 0, false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return filename, size, supportsRange, nil
}

// Download downloads a file with progress tracking and resume support
func (d *Downloader) Download(ctx context.Context, downloadURL, destDir, filename string, resumeFrom int64, onProgress func(downloaded int64)) (*DownloadResult, error) {
	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	filePath := filepath.Join(destDir, filename)
	tempPath := filePath + ".part"

	// Open file for writing (append if resuming)
	var file *os.File
	var err error

	if resumeFrom > 0 {
		file, err = os.OpenFile(tempPath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			// If can't open existing file, start fresh
			resumeFrom = 0
			file, err = os.Create(tempPath)
		}
	} else {
		file, err = os.Create(tempPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Gloski/1.0")

	// Set Range header for resume
	if resumeFrom > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
	}

	// Execute request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// If server doesn't support range and we're resuming, start over
	if resumeFrom > 0 && resp.StatusCode == http.StatusOK {
		file.Close()
		file, err = os.Create(tempPath)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate file: %w", err)
		}
		defer file.Close()
		resumeFrom = 0
	}

	// Calculate total size
	totalSize := resp.ContentLength
	if resumeFrom > 0 && totalSize > 0 {
		totalSize += resumeFrom
	}

	// Create progress writer
	progressWriter := &ProgressWriter{
		writer:     file,
		downloaded: resumeFrom,
		onProgress: onProgress,
	}

	// Copy data
	_, err = io.Copy(progressWriter, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download interrupted: %w", err)
	}

	// Close file before renaming
	file.Close()

	// Rename temp file to final name
	if err := os.Rename(tempPath, filePath); err != nil {
		return nil, fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Get final file size
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return &DownloadResult{
		Filename: filename,
		FilePath: filePath,
		Size:     info.Size(),
	}, nil
}

// extractFilename extracts filename from Content-Disposition header or URL
func extractFilename(header http.Header, downloadURL string) string {
	// Try Content-Disposition header first
	if cd := header.Get("Content-Disposition"); cd != "" {
		_, params, err := mime.ParseMediaType(cd)
		if err == nil {
			if filename, ok := params["filename"]; ok && filename != "" {
				return sanitizeFilename(filename)
			}
		}
	}

	// Fall back to URL path
	parsedURL, err := url.Parse(downloadURL)
	if err == nil {
		filename := path.Base(parsedURL.Path)
		if filename != "" && filename != "." && filename != "/" {
			// Remove query string if present
			if idx := strings.Index(filename, "?"); idx != -1 {
				filename = filename[:idx]
			}
			return sanitizeFilename(filename)
		}
	}

	// Default filename
	return "download"
}

// sanitizeFilename removes or replaces invalid characters in filename
func sanitizeFilename(filename string) string {
	// Remove path separators and null bytes
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Trim spaces and dots from ends
	filename = strings.Trim(filename, " .")

	// Limit length
	if len(filename) > 255 {
		ext := filepath.Ext(filename)
		name := filename[:255-len(ext)]
		filename = name + ext
	}

	if filename == "" {
		return "download"
	}

	return filename
}

// GetPartialFileSize returns the size of the partial download file, or 0 if not exists
func GetPartialFileSize(destDir, filename string) int64 {
	tempPath := filepath.Join(destDir, filename+".part")
	info, err := os.Stat(tempPath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// CleanupPartialFile removes the partial download file
func CleanupPartialFile(destDir, filename string) error {
	tempPath := filepath.Join(destDir, filename+".part")
	err := os.Remove(tempPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
