// Package api provides interface definitions for services used by the API layer.
// These interfaces enable dependency injection and easier testing.
package api

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/ss497254/gloski/internal/downloads"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/internal/jobs"
	"github.com/ss497254/gloski/internal/system"
)

// Authenticator defines the interface for authentication services
type Authenticator interface {
	ValidateAPIKey(key string) error
	ValidateToken(token string) error
	HasAPIKey() bool
	HasJWT() bool
}

// FileService defines the interface for file operations
type FileService interface {
	List(path string) (*files.ListResponse, error)
	ListWithContext(ctx context.Context, path string) (*files.ListResponse, error)
	Read(path string) (string, error)
	Write(path, content string) error
	Mkdir(path string) error
	Delete(path string) error
	Rename(oldPath, newPath string) error
	Upload(destPath, filename string, reader io.Reader) error
	GetFileInfo(path string) (string, os.FileInfo, error)
	Search(path, query string, searchContent bool, limit int) ([]files.SearchResult, error)
	SearchWithContext(ctx context.Context, path, query string, searchContent bool, limit int) ([]files.SearchResult, error)

	// Chunked upload support
	InitChunkedUpload(destPath, filename string, totalSize, chunkSize int64) (*files.ChunkUploadInfo, error)
	UploadChunk(destPath, filename, uploadID string, chunkIndex int, reader io.Reader) error
	CompleteChunkedUpload(destPath, filename, uploadID string, totalChunks int) error
	AbortChunkedUpload(destPath, uploadID string) error
}

// JobService defines the interface for job management
type JobService interface {
	List() ([]*jobs.Job, error)
	Start(id, command, cwd string) (*jobs.Job, error)
	Get(id string) (*jobs.Job, error)
	GetLogs(id string) ([]string, error)
	Stop(id string) error
	Delete(id string) error
	Shutdown() error
}

// DownloadService defines the interface for download management
type DownloadService interface {
	List() []*downloads.Download
	Get(id string) (*downloads.Download, error)
	Add(url, destination, filename string) (*downloads.Download, error)
	Pause(id string) error
	Resume(id string) error
	Cancel(id string) error
	Retry(id string) error
	Delete(id string, deleteFile bool) error
	CreateShareLink(id string, expiresIn *int) (*downloads.ShareLink, error)
	RevokeShareLink(id, token string) error
	GetByShareToken(token string) (*downloads.Download, error)
	Shutdown(timeout time.Duration)
}

// SystemService defines the interface for system information
type SystemService interface {
	GetStats() (*system.Stats, error)
	GetStatsHistory(duration time.Duration) []system.Sample
	GetServerInfo() (*system.ServerInfo, error)
	GetProcesses(limit int) ([]system.Process, error)
}
