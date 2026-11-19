package files

import (
	"errors"
	"os"
)

// Rename moves or renames a path. The source must exist and the destination
// must not. Both paths are subject to the same protected-path policy as
// Delete.
func (s *Service) Rename(oldPath, newPath string) error {
	absOldPath, err := s.validatePath(oldPath)
	if err != nil {
		return err
	}

	absNewPath, err := s.validatePath(newPath)
	if err != nil {
		return err
	}

	if isProtectedPath(absOldPath) || isProtectedPath(absNewPath) {
		return ErrDangerousPath
	}

	if _, err := os.Stat(absOldPath); os.IsNotExist(err) {
		return errors.New("source path does not exist")
	}

	if _, err := os.Stat(absNewPath); err == nil {
		return errors.New("destination path already exists")
	}

	return os.Rename(absOldPath, absNewPath)
}
