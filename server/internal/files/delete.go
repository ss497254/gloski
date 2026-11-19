package files

import "os"

// Delete removes a file or directory tree. Protected roots, their contents
// (except below /tmp) and the home directory itself are refused.
func (s *Service) Delete(path string) error {
	absPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	if isProtectedPath(absPath) {
		return ErrDangerousPath
	}

	return os.RemoveAll(absPath)
}
