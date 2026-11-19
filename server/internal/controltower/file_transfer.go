package controltower

import (
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/proto/agentv1"
)

// transfer is one file moving between a browser request and an agent. The
// bytes travel over HTTP on the agent port; only the request and the result
// use the gRPC stream. The pipe is unbuffered, so a slow browser or disk slows
// the transfer instead of filling memory.
type transfer struct {
	id        string
	direction agentv1.FileTransferDirection
	reader    *io.PipeReader
	writer    *io.PipeWriter

	done chan struct{}
	once sync.Once
	mu   sync.Mutex
	err  error
}

// complete settles the transfer with the agent's outcome. A failure closes
// the pipe so a waiting browser request does not hang.
func (tr *transfer) complete(err error) {
	tr.once.Do(func() {
		tr.mu.Lock()
		tr.err = err
		tr.mu.Unlock()
		if err != nil {
			_ = tr.reader.CloseWithError(err)
			_ = tr.writer.CloseWithError(err)
		} else {
			// The agent is done; readers must see end of file, and a writer
			// that is still open stops.
			_ = tr.writer.Close()
		}
		close(tr.done)
	})
}

// wait blocks until the agent reports an outcome.
func (tr *transfer) wait() error {
	<-tr.done
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.err
}

// beginTransfer registers a one-time transfer and asks the device to run it.
func (t *Tower) beginTransfer(deviceID string, request *agentv1.FileTransfer) (*transfer, error) {
	request.TransferId = uuid.NewString()
	reader, writer := io.Pipe()
	tr := &transfer{
		id:        request.GetTransferId(),
		direction: request.GetDirection(),
		reader:    reader,
		writer:    writer,
		done:      make(chan struct{}),
	}

	t.transfersMu.Lock()
	if t.transfers == nil {
		t.transfers = make(map[string]*transfer)
	}
	t.transfers[tr.id] = tr
	t.transfersMu.Unlock()

	result, err := t.agentService.StartTransfer(deviceID, request)
	if err != nil {
		t.dropTransfer(tr.id)
		tr.complete(err)
		return nil, err
	}
	go func() {
		tr.complete(<-result)
		t.dropTransfer(tr.id)
	}()
	return tr, nil
}

// takeTransfer removes a transfer from the registry: an id is used once.
func (t *Tower) takeTransfer(id string) *transfer {
	t.transfersMu.Lock()
	defer t.transfersMu.Unlock()
	tr := t.transfers[id]
	delete(t.transfers, id)
	return tr
}

func (t *Tower) dropTransfer(id string) {
	t.transfersMu.Lock()
	delete(t.transfers, id)
	t.transfersMu.Unlock()
}

// handleAgentTransfer carries the bytes of one transfer. The agent is always
// the HTTP client: it posts a download or fetches an upload.
func (t *Tower) handleAgentTransfer(w http.ResponseWriter, r *http.Request) {
	tr := t.takeTransfer(r.PathValue("id"))
	if tr == nil {
		http.NotFound(w, r)
		return
	}

	switch {
	case r.Method == http.MethodPost && tr.direction == agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_DOWNLOAD:
		_, err := io.Copy(tr.writer, r.Body)
		_ = tr.writer.CloseWithError(err)
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			logger.Warn("Transfer %s from device failed: %v", tr.id, err)
		}
		w.WriteHeader(http.StatusOK)
	case r.Method == http.MethodGet && tr.direction == agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_UPLOAD:
		w.Header().Set("Content-Type", "application/octet-stream")
		if _, err := io.Copy(w, tr.reader); err != nil {
			// Abort instead of ending the body normally: the agent must not
			// mistake a short upload for a complete file.
			logger.Warn("Transfer %s to device failed: %v", tr.id, err)
			panic(http.ErrAbortHandler)
		}
	default:
		http.NotFound(w, r)
	}
}

// startDownload asks the agent to send a file range to the master.
func (t *Tower) startDownload(deviceID, path string, offset, length int64) (*transfer, error) {
	return t.beginTransfer(deviceID, &agentv1.FileTransfer{
		Direction: agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_DOWNLOAD,
		Path:      path,
		Offset:    offset,
		Length:    length,
	})
}

// startUpload asks the agent to fetch a file from the master.
func (t *Tower) startUpload(deviceID, path string) (*transfer, error) {
	return t.beginTransfer(deviceID, &agentv1.FileTransfer{
		Direction: agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_UPLOAD,
		Path:      path,
	})
}
