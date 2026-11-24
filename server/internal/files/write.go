package files

import (
	"errors"
	"io"
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

// WriteStream copies content into path atomically: bytes accumulate in a
// temporary file beside the target, which replaces it only when the copy
// completes. A failed or interrupted transfer therefore leaves the old file
// untouched.
func (s *Service) WriteStream(path string, content io.Reader) (int64, error) {
	absPath, err := s.validatePath(path)
	if err != nil {
		return 0, err
	}
	if info, err := os.Stat(absPath); err == nil && info.IsDir() {
		return 0, errors.New("destination is a directory")
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return 0, err
	}

	file, err := os.CreateTemp(filepath.Dir(absPath), "."+filepath.Base(absPath)+".gloski-upload-*")
	if err != nil {
		return 0, err
	}
	temp := file.Name()
	if err := file.Chmod(0644); err != nil {
		_ = file.Close()
		_ = os.Remove(temp)
		return 0, err
	}
	written, copyErr := io.Copy(file, content)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(temp)
		return written, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(temp)
		return written, closeErr
	}
	if err := os.Rename(temp, absPath); err != nil {
		_ = os.Remove(temp)
		return written, err
	}
	return written, nil
}

// Mkdir creates a directory and any missing parents with mode 0755.
func (s *Service) Mkdir(path string) error {
	absPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(absPath, 0755)
}
