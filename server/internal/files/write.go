package files

import (
	"os"
	"path/filepath"
)

// Write replaces a file with content, creating parent directories and setting
// mode 0644.
func (s *Service) Write(path, content string) error {
	absPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(absPath, []byte(content), 0644)
}

// Mkdir creates a directory and any missing parents with mode 0755.
func (s *Service) Mkdir(path string) error {
	absPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(absPath, 0755)
}
