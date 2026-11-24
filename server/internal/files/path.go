package files

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ss497254/gloski/internal/config"
)

// Errors returned for expected filesystem failures.
var (
	// ErrPathNotAllowed means the path is outside the configured allowed roots.
	ErrPathNotAllowed = errors.New("path not allowed")
	// ErrFileTooLarge means the file exceeds MaxFileSize.
	ErrFileTooLarge = errors.New("file too large")
	// ErrBinaryFile means the editor refused a file that looks binary.
	ErrBinaryFile = errors.New("binary file not supported")
	// ErrDangerousPath means a destructive operation targeted a system path,
	// the home directory itself, or a path below a protected root.
	ErrDangerousPath = errors.New("dangerous path operation")
)

// MaxFileSize is the largest file Read will load for the editor.
const MaxFileSize = 10 * 1024 * 1024 // 10 MB

// protectedPaths are the system roots a delete or rename may not touch. Both
// refuse the root and anything below it; only paths under /tmp are exempt,
// where operating on individual files is normal.
var protectedPaths = []string{
	"/", "/etc", "/usr", "/bin", "/sbin", "/var", "/boot",
	"/lib", "/lib64", "/dev", "/proc", "/sys", "/run", "/tmp", "/root",
}

// Service performs filesystem operations under the configured path policy.
type Service struct {
	config *config.Config
}

// NewService creates a files service using cfg for its allowed roots.
func NewService(cfg *config.Config) *Service {
	return &Service{config: cfg}
}

// ExpandTilde expands a leading ~ or ~/ to the user's home directory.
func ExpandTilde(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[2:]), nil
	}
	return path, nil
}

// ToTildePath converts an absolute path under the home directory back to ~
// notation. Other paths are returned unchanged.
func ToTildePath(absPath string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return absPath
	}

	if absPath == homeDir {
		return "~"
	}

	if strings.HasPrefix(absPath, homeDir+string(filepath.Separator)) {
		return "~" + absPath[len(homeDir):]
	}

	return absPath
}

// NormalizePath validates a path and returns it in tilde notation when it is
// under the home directory.
func (s *Service) NormalizePath(path string) (string, error) {
	absPath, err := s.validatePath(path)
	if err != nil {
		return "", err
	}
	return ToTildePath(absPath), nil
}

// ResolvePath expands, cleans and checks a path against the allowed roots,
// returning the absolute path. The file itself does not have to exist.
func (s *Service) ResolvePath(path string) (string, error) {
	return s.validatePath(path)
}

func (s *Service) validatePath(path string) (string, error) {
	expandedPath, err := ExpandTilde(path)
	if err != nil {
		return "", err
	}

	expandedPath = filepath.Clean(expandedPath)

	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return "", err
	}

	// Resolve symlinks to prevent traversal attacks. Paths that don't exist
	// yet (writes and nested mkdir) still resolve their nearest existing
	// ancestor, so a new directory tree cannot escape an allowed root.
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		absPath, err = resolveMissingPath(absPath)
		if err != nil {
			return "", err
		}
	} else {
		absPath = resolved
	}

	if !s.config.IsPathAllowed(absPath) {
		return "", ErrPathNotAllowed
	}

	return absPath, nil
}

// resolveMissingPath resolves the nearest existing ancestor and reapplies the
// missing path segments. This keeps symlink validation effective while still
// allowing files and nested directories to be created.
func resolveMissingPath(absPath string) (string, error) {
	var missing []string
	current := absPath

	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

// isProtectedRoot reports whether path is a protected root or the home
// directory itself. It is the base check used by isProtectedTree.
func isProtectedRoot(path string) bool {
	if path == "/" {
		return true
	}
	if home := os.Getenv("HOME"); home != "" && path == home {
		return true
	}
	for _, root := range protectedPaths {
		if path == root {
			return true
		}
	}
	return false
}

// isProtectedTree reports whether path is at or below a protected root. /tmp
// is the exception: its contents are normal files. Delete and Rename both use
// this check so neither can modify files inside system directories.
func isProtectedTree(path string) bool {
	if isProtectedRoot(path) {
		return true
	}
	for _, root := range protectedPaths {
		if root != "/tmp" && strings.HasPrefix(path, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
