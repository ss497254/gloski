package controltower

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ss497254/gloski/proto/agentv1"
)

// newTestTransfer builds a registered transfer without an agent, so the HTTP
// half can be tested on its own.
func newTestTransfer(handlers *deviceHandlers, direction agentv1.FileTransferDirection) *transfer {
	reader, writer := io.Pipe()
	tr := &transfer{
		id:        "transfer-1",
		direction: direction,
		reader:    reader,
		writer:    writer,
		done:      make(chan struct{}),
	}
	handlers.transfersMu.Lock()
	handlers.transfers[tr.id] = tr
	handlers.transfersMu.Unlock()
	return tr
}

func TestAgentTransferReceivesDownload(t *testing.T) {
	handlers := &deviceHandlers{transfers: make(map[string]*transfer)}
	tr := newTestTransfer(handlers, agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_DOWNLOAD)

	received := make(chan string, 1)
	go func() {
		content, _ := io.ReadAll(tr.reader)
		received <- string(content)
	}()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/agent/transfer/transfer-1", strings.NewReader("file bytes"))
	request.SetPathValue("id", "transfer-1")
	handlers.handleAgentTransfer(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if content := <-received; content != "file bytes" {
		t.Errorf("reader got %q, want %q", content, "file bytes")
	}
}

func TestAgentTransferServesUpload(t *testing.T) {
	handlers := &deviceHandlers{transfers: make(map[string]*transfer)}
	tr := newTestTransfer(handlers, agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_UPLOAD)

	go func() {
		_, _ = io.WriteString(tr.writer, "upload bytes")
		_ = tr.writer.Close()
	}()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/agent/transfer/transfer-1", nil)
	request.SetPathValue("id", "transfer-1")
	handlers.handleAgentTransfer(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "upload bytes" {
		t.Errorf("body = %q, want %q", recorder.Body.String(), "upload bytes")
	}
}

func TestAgentTransferIdIsUsedOnce(t *testing.T) {
	handlers := &deviceHandlers{transfers: make(map[string]*transfer)}
	tr := newTestTransfer(handlers, agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_UPLOAD)

	go func() {
		_, _ = io.WriteString(tr.writer, "bytes")
		_ = tr.writer.Close()
	}()

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "/agent/transfer/transfer-1", nil)
	firstRequest.SetPathValue("id", "transfer-1")
	handlers.handleAgentTransfer(first, firstRequest)
	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, "/agent/transfer/transfer-1", nil)
	secondRequest.SetPathValue("id", "transfer-1")
	handlers.handleAgentTransfer(second, secondRequest)

	if second.Code != http.StatusNotFound {
		t.Errorf("second use status = %d, want %d", second.Code, http.StatusNotFound)
	}
}
