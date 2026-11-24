package files

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ss497254/gloski/internal/config"
)

func TestIsProtectedRoot(t *testing.T) {
	t.Setenv("HOME", "/home/tester")

	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"/etc", true},
		{"/etc/passwd", false},
		{"/usr", true},
		{"/usr/local/bin", false},
		{"/dev", true},
		{"/dev/sda", false},
		{"/tmp", true},
		{"/tmp/scratch", false},
		{"/root", true},
		{"/root/.ssh", false},
		{"/home/tester", true},
		{"/home/tester/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isProtectedRoot(tt.path); got != tt.want {
				t.Errorf("isProtectedRoot(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsProtectedTree(t *testing.T) {
	t.Setenv("HOME", "/home/tester")

	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"/etc", true},
		{"/etc/passwd", true},
		{"/usr/local/bin", true},
		{"/dev/sda", true},
		{"/proc/1", true},
		{"/sys/class", true},
		{"/run/gloski", true},
		{"/root/.ssh", true},
		{"/home/tester", true},
		{"/home/tester/file.txt", false},
		{"/tmp", true},
		{"/tmp/scratch", false},
		{"/srv/data", false},
		{"/home/other", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isProtectedTree(tt.path); got != tt.want {
				t.Errorf("isProtectedTree(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDeleteRefusesProtectedPaths(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	service := NewService(&config.Config{})

	for _, path := range []string{"/", "/etc/passwd", "/proc/1", "/home/tester"} {
		if err := service.Delete(path); !errors.Is(err, ErrDangerousPath) {
			t.Errorf("Delete(%q) error = %v, want ErrDangerousPath", path, err)
		}
	}
}

// TestRenameRefusesProtectedPaths checks that Rename refuses protected roots
// and paths below them, on either side of the operation. Paths under /tmp stay
// legal; TestIsProtectedTree covers that boundary.
func TestRenameRefusesProtectedPaths(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	service := NewService(&config.Config{})

	tests := []struct{ oldPath, newPath string }{
		{"/etc", "/tmp/etc"},
		{"/tmp/etc", "/etc"},
		{"/etc/passwd", "/etc/shadow"},
		{"/tmp/etc", "/etc/passwd"},
		{"/", "/tmp/root"},
		{"/home/tester", "/tmp/tester"},
	}
	for _, tt := range tests {
		if err := service.Rename(tt.oldPath, tt.newPath); !errors.Is(err, ErrDangerousPath) {
			t.Errorf("Rename(%q, %q) error = %v, want ErrDangerousPath", tt.oldPath, tt.newPath, err)
		}
	}
}

func TestRenameAllowsPathsUnderTmp(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	service := NewService(&config.Config{})

	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(oldPath, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := service.Rename(oldPath, newPath); err != nil {
		t.Fatalf("Rename(%q, %q) error = %v, want nil", oldPath, newPath, err)
	}
}
