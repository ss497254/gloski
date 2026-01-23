package files

import (
	"bufio"
	"errors"
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
	fullPath := filepath.Join(absPath, filename)

	// Create the file
	f, err := os.Create(fullPath)
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

// Search searches for files by name or content
func (s *Service) Search(path, query string, searchContent bool, limit int) ([]SearchResult, error) {
	absPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	results := []SearchResult{}
	query = strings.ToLower(query)

	err = filepath.Walk(absPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors (permission denied, etc.)
		}

		// Skip hidden directories (but not hidden files)
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && filePath != absPath {
			return filepath.SkipDir
		}

		// Skip node_modules and other large directories
		skipDirs := []string{"node_modules", ".git", "vendor", "__pycache__", ".cache"}
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

	return results, err
}

type contentMatch struct {
	lineNum int
	line    string
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
