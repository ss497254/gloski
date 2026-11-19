package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ss497254/gloski/proto/agentv1"
)

// handleTransfer moves one file over HTTP and reports the outcome on the
// stream. The stream carries only this request and the final result, so a
// multi-gigabyte transfer never delays terminals or commands.
func (w *Runtime) handleTransfer(id string, transfer *agentv1.FileTransfer) {
	result := &agentv1.FileResult{}
	if err := w.runTransfer(transfer); err != nil {
		result = fileError(err)
	}
	_ = w.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload:   &agentv1.AgentMessage_FileResult{FileResult: result},
	})
}

// runTransfer makes the one HTTP request the master asked for. The agent is
// always the client: it either sends the file or fetches it.
func (w *Runtime) runTransfer(transfer *agentv1.FileTransfer) error {
	client, base, err := w.transferClient()
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(base, "/") + "/agent/transfer/" + url.PathEscape(transfer.GetTransferId())

	switch transfer.GetDirection() {
	case agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_DOWNLOAD:
		return w.sendFile(client, endpoint, transfer)
	case agentv1.FileTransferDirection_FILE_TRANSFER_DIRECTION_UPLOAD:
		return w.receiveFile(client, endpoint, transfer)
	default:
		return errors.New("unsupported file transfer direction")
	}
}

// transferClient builds an HTTP client for the master's agent port. The local
// agent talks over the same unix socket it uses for gRPC; remote agents use
// the address they already dial.
func (w *Runtime) transferClient() (*http.Client, string, error) {
	raw := strings.TrimSpace(w.cfg.Agent.MasterURL)
	// One request per transfer, so keep-alives are pointless. A transfer may
	// take as long as the file needs; only the first response has a deadline.
	transport := &http.Transport{DisableKeepAlives: true, ResponseHeaderTimeout: 30 * time.Second}
	if strings.HasPrefix(raw, "unix://") {
		socket := strings.TrimPrefix(raw, "unix://")
		transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}
		return &http.Client{Transport: transport}, "http://agent", nil
	}

	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return nil, "", fmt.Errorf("invalid master URL %q", w.cfg.Agent.MasterURL)
	}
	return &http.Client{Transport: transport}, parsed.Scheme + "://" + parsed.Host, nil
}

// sendFile supplies a download: the master reads the body and writes it to
// the browser.
func (w *Runtime) sendFile(client *http.Client, endpoint string, transfer *agentv1.FileTransfer) error {
	absPath, info, err := w.files.GetFileInfo(transfer.GetPath())
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("cannot transfer a directory")
	}

	file, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer file.Close()

	offset := transfer.GetOffset()
	if offset < 0 || offset > info.Size() {
		return errors.New("invalid transfer offset")
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	var body io.Reader = file
	if length := transfer.GetLength(); length > 0 {
		body = io.LimitReader(file, length)
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/octet-stream")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return transferRejected(response)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	return nil
}

// receiveFile fetches an upload and writes it atomically, so a broken
// transfer leaves the previous file in place.
func (w *Runtime) receiveFile(client *http.Client, endpoint string, transfer *agentv1.FileTransfer) error {
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return transferRejected(response)
	}
	_, err = w.files.WriteStream(transfer.GetPath(), response.Body)
	return err
}

// transferRejected turns the master's error response into a message for the
// stream.
func transferRejected(response *http.Response) error {
	message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	text := strings.TrimSpace(string(message))
	if text == "" {
		text = response.Status
	}
	return fmt.Errorf("master rejected transfer: %s", text)
}
