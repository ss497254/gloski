package files

import (
	"errors"
	"os"
)

// Rename moves or renames a path. The source must exist and the destination
// must not. Like Delete, it refuses protected roots and anything below them;
// only paths under /tmp are exempt.
func (s *Service) Rename(oldPath, newPath string) error {
	absOldPath, err := s.validatePath(oldPath)
	if err != nil {
		return err
	}

	absNewPath, err := s.validatePath(newPath)
	if err != nil {
		return err
	}

	if isProtectedTree(absOldPath) || isProtectedTree(absNewPath) {
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
