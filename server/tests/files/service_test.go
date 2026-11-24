package files_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/files"
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

func TestWriteStreamReplacesFile(t *testing.T) {
	service := files.NewService(&config.Config{})
	target := filepath.Join(t.TempDir(), "dir", "file.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	written, err := service.WriteStream(target, strings.NewReader("new contents"))
	if err != nil {
		t.Fatalf("WriteStream() error = %v", err)
	}
	if written != int64(len("new contents")) {
		t.Errorf("WriteStream() wrote %d bytes, want %d", written, len("new contents"))
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new contents" {
		t.Errorf("content = %q, want %q", content, "new contents")
	}
}

func TestWriteStreamLeavesFileOnFailure(t *testing.T) {
	service := files.NewService(&config.Config{})
	target := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := service.WriteStream(target, &failingReader{}); err == nil {
		t.Fatal("WriteStream() succeeded, want an error")
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "old" {
		t.Errorf("content = %q, want the previous file %q", content, "old")
	}
	temps, err := filepath.Glob(filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".gloski-upload-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temps) != 0 {
		t.Errorf("temporary upload files were left behind: %v", temps)
	}
}

func TestWriteStreamConcurrentSameDestination(t *testing.T) {
	service := files.NewService(&config.Config{})
	target := filepath.Join(t.TempDir(), "file.txt")
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})

	firstDone := make(chan error, 1)
	go func() {
		_, err := service.WriteStream(target, &blockingReader{
			started: firstStarted,
			release: releaseFirst,
			content: "first upload",
		})
		firstDone <- err
	}()
	<-firstStarted

	if _, err := service.WriteStream(target, strings.NewReader("second upload")); err != nil {
		t.Fatalf("second WriteStream() error = %v", err)
	}
	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatalf("first WriteStream() error = %v", err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(content); got != "first upload" && got != "second upload" {
		t.Errorf("final content = %q, want one complete upload", got)
	}
}

type blockingReader struct {
	started chan struct{}
	release chan struct{}
	content string
	read    bool
}

func (r *blockingReader) Read(p []byte) (int, error) {
	if !r.read {
		r.read = true
		close(r.started)
		<-r.release
		return copy(p, r.content), nil
	}
	return 0, io.EOF
}

type failingReader struct{ sent bool }

func (r *failingReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, "partial"), nil
	}
	return 0, errors.New("connection lost")
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
