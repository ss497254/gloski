package files

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SearchResult is one file or content match.
type SearchResult struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Size    int64  `json:"size"`
	Match   string `json:"match,omitempty"`    // For content search: the matching line
	LineNum int    `json:"line_num,omitempty"` // For content search: line number
}

// SearchOptions bounds a search.
type SearchOptions struct {
	Timeout  time.Duration // Maximum search duration (default: 30s)
	MaxDepth int           // Maximum directory depth (default: 20)
}

// DefaultSearchOptions returns the default bounds.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Timeout:  30 * time.Second,
		MaxDepth: 20,
	}
}

// Search looks for files by name or content with a background context and the
// default options.
func (s *Service) Search(path, query string, searchContent bool, limit int) ([]SearchResult, error) {
	return s.SearchWithOptions(context.Background(), path, query, searchContent, limit, DefaultSearchOptions())
}

// SearchWithContext looks for files with a cancellable context and the default
// options.
func (s *Service) SearchWithContext(ctx context.Context, path, query string, searchContent bool, limit int) ([]SearchResult, error) {
	return s.SearchWithOptions(ctx, path, query, searchContent, limit, DefaultSearchOptions())
}

// searchSkipDirs are large directories that would make a content search slow
// for no benefit.
var searchSkipDirs = []string{"node_modules", ".git", "vendor", "__pycache__", ".cache", ".chunks"}

// SearchWithOptions looks for files by name, and optionally by content, below
// path. It returns partial results rather than an error when the timeout or
// context fires.
func (s *Service) SearchWithOptions(ctx context.Context, path, query string, searchContent bool, limit int, opts SearchOptions) ([]SearchResult, error) {
	absPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	results := []SearchResult{}
	query = strings.ToLower(query)
	baseDepth := strings.Count(absPath, string(os.PathSeparator))

	err = filepath.Walk(absPath, func(filePath string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return nil // Skip permission errors and files that vanished.
		}

		currentDepth := strings.Count(filePath, string(os.PathSeparator)) - baseDepth
		if opts.MaxDepth > 0 && currentDepth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Hidden directories are skipped, hidden files are not.
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && filePath != absPath {
			return filepath.SkipDir
		}

		if info.IsDir() {
			for _, skip := range searchSkipDirs {
				if info.Name() == skip {
					return filepath.SkipDir
				}
			}
		}

		if len(results) >= limit {
			return filepath.SkipAll
		}

		if strings.Contains(strings.ToLower(info.Name()), query) {
			entryType := "file"
			if info.IsDir() {
				entryType = "directory"
			}
			results = append(results, SearchResult{
				Path: ToTildePath(filePath),
				Name: info.Name(),
				Type: entryType,
				Size: info.Size(),
			})
			return nil
		}

		if searchContent && !info.IsDir() && info.Size() < MaxFileSize {
			matches := s.searchFileContent(filePath, query, 3) // Max 3 matches per file
			for _, match := range matches {
				if len(results) >= limit {
					return filepath.SkipAll
				}
				results = append(results, SearchResult{
					Path:    ToTildePath(filePath),
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

	// Timeout and cancellation return what was found so far.
	if err == context.DeadlineExceeded || err == context.Canceled {
		return results, nil
	}

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
			if len(line) > 200 {
				line = line[:200] + "..."
			}
			matches = append(matches, contentMatch{lineNum: lineNum, line: line})
			if len(matches) >= maxMatches {
				break
			}
		}
	}

	return matches
}
