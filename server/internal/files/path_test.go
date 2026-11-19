package files

import (
	"errors"
	"testing"

	"github.com/ss497254/gloski/internal/config"
)

func TestIsProtectedPath(t *testing.T) {
	t.Setenv("HOME", "/home/tester")

	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"/etc", true},
		{"/etc/passwd", true},
		{"/usr/local/bin", true},
		{"/bin/sh", true},
		{"/sbin/init", true},
		{"/var/lib/gloski", true},
		{"/boot/vmlinuz", true},
		{"/lib/modules", true},
		{"/lib64/ld-linux", true},
		{"/dev/sda", true},
		{"/proc/1", true},
		{"/sys/class", true},
		{"/run/gloski", true},
		{"/tmp", true},
		{"/tmp/scratch", false},
		{"/root", true},
		{"/root/.ssh", true},
		{"/home/tester", true},
		{"/home/tester/file.txt", false},
		{"/srv/data", false},
		{"/home/other", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isProtectedPath(tt.path); got != tt.want {
				t.Errorf("isProtectedPath(%q) = %v, want %v", tt.path, got, tt.want)
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

func TestRenameRefusesProtectedPaths(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	service := NewService(&config.Config{})

	tests := []struct{ oldPath, newPath string }{
		{"/etc/passwd", "/tmp/passwd"},
		{"/tmp/passwd", "/etc/passwd"},
		{"/", "/tmp/root"},
		{"/home/tester", "/tmp/tester"},
	}
	for _, tt := range tests {
		if err := service.Rename(tt.oldPath, tt.newPath); !errors.Is(err, ErrDangerousPath) {
			t.Errorf("Rename(%q, %q) error = %v, want ErrDangerousPath", tt.oldPath, tt.newPath, err)
		}
	}
}
