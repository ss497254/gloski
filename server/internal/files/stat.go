package files

import "os"

// Stat returns the file info for a path.
func (s *Service) Stat(path string) (os.FileInfo, error) {
	absPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	return os.Stat(absPath)
}

// Exists reports whether a path exists and is allowed.
func (s *Service) Exists(path string) bool {
	absPath, err := s.validatePath(path)
	if err != nil {
		return false
	}

	_, err = os.Stat(absPath)
	return err == nil
}

// GetFileInfo returns the resolved absolute path and file info. It is used by
// the agent's piece reads and writes, which need the real path.
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
