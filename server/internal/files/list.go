package files

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// FileEntry is one item in a directory listing.
type FileEntry struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Type        string    `json:"type"` // "file" or "directory"
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	Permissions string    `json:"permissions"`
}

// ListResponse is a directory listing.
type ListResponse struct {
	Path    string      `json:"path"`
	Entries []FileEntry `json:"entries"`
}

// List returns a directory listing with a background context.
func (s *Service) List(path string) (*ListResponse, error) {
	return s.ListWithContext(context.Background(), path)
}

// ListWithContext returns a directory listing and stops early when ctx is
// cancelled.
func (s *Service) ListWithContext(ctx context.Context, path string) (*ListResponse, error) {
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
		// Check context periodically for large directories.
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

		entryAbsPath := filepath.Join(absPath, entry.Name())
		fileEntries = append(fileEntries, FileEntry{
			Name:        entry.Name(),
			Path:        ToTildePath(entryAbsPath),
			Type:        entryType,
			Size:        info.Size(),
			Modified:    info.ModTime(),
			Permissions: info.Mode().String(),
		})
	}

	return &ListResponse{
		Path:    ToTildePath(absPath),
		Entries: fileEntries,
	}, nil
}
