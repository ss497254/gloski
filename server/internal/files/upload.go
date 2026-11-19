package files

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Upload streams reader into filename inside destPath. It replaces an existing
// file. This is the whole-request path; browser uploads use a FileTransfer
// over HTTP instead.
func (s *Service) Upload(destPath, filename string, reader io.Reader) error {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("destination must be a directory")
	}

	filename, err = cleanFilename(filename)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(absPath, filename)

	// Validate the full path too, in case a symlinked directory moved it.
	if _, err := s.validatePath(fullPath); err != nil {
		return err
	}

	f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	return err
}

// cleanFilename keeps only the final path element. Callers may pass a name
// from a client, so it must not contain a directory.
func cleanFilename(name string) (string, error) {
	name = filepath.Base(name)
	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
		return "", errors.New("invalid filename")
	}
	return name, nil
}
