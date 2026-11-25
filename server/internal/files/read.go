package files

import (
	"os"
)

// Read returns a text file as a string. Files above MaxFileSize and files
// containing a NUL byte are refused.
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
		return "", ErrIsDirectory
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

// isBinary reports whether the first 8 KB contain a NUL byte. That is the
// usual heuristic for "the editor should not show this".
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
