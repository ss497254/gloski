package files_test

import (
	"github.com/ss497254/gloski/internal/files"
	"os"
	"path/filepath"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "tilde only",
			input:    "~",
			expected: homeDir,
			wantErr:  false,
		},
		{
			name:     "tilde with path",
			input:    "~/Documents",
			expected: filepath.Join(homeDir, "Documents"),
			wantErr:  false,
		},
		{
			name:     "absolute path",
			input:    "/usr/local",
			expected: "/usr/local",
			wantErr:  false,
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := files.ExpandTilde(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("files.ExpandTilde() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("files.ExpandTilde() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToTildePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "home directory exactly",
			input:    homeDir,
			expected: "~",
		},
		{
			name:     "path within home",
			input:    filepath.Join(homeDir, "Documents", "file.txt"),
			expected: "~/Documents/file.txt",
		},
		{
			name:     "path outside home",
			input:    "/usr/local/bin",
			expected: "/usr/local/bin",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := files.ToTildePath(tt.input)
			if result != tt.expected {
				t.Errorf("files.ToTildePath() = %v, want %v", result, tt.expected)
			}
		})
	}
}
