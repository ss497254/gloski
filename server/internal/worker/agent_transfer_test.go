package worker

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/proto/agentv1"
)

func newTransferRuntime(t *testing.T, masterURL string) *Runtime {
	t.Helper()
	cfg := &config.Config{}
	cfg.Agent.MasterURL = masterURL
	return &Runtime{cfg: cfg, files: files.NewService(cfg)}
}

func TestDownloadTransferSendsRequestedRange(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(source, []byte("0123456789"), 0644); err != nil {
		t.Fatal(err)
	}

	var method string
	var received []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		received, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	runtime := newTransferRuntime(t, server.URL)
	err := runtime.runTransfer(&agentv1.FileTransfer{
		Direction: agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_DOWNLOAD,
		Path:      source,
		Offset:    2,
		Length:    5,
	})
	if err != nil {
		t.Fatalf("runTransfer() error = %v", err)
	}
	if method != http.MethodPost {
		t.Errorf("agent used %s, want POST", method)
	}
	if string(received) != "23456" {
		t.Errorf("master received %q, want %q", received, "23456")
	}
}

func TestUploadTransferWritesFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("agent used %s, want GET", r.Method)
		}
		_, _ = io.WriteString(w, "uploaded contents")
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "nested", "destination.txt")
	runtime := newTransferRuntime(t, server.URL)
	err := runtime.runTransfer(&agentv1.FileTransfer{
		Direction: agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_UPLOAD,
		Path:      destination,
	})
	if err != nil {
		t.Fatalf("runTransfer() error = %v", err)
	}

	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "uploaded contents" {
		t.Errorf("destination = %q, want %q", content, "uploaded contents")
	}
}
