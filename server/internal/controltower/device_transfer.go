package controltower

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/proto/agentv1"
)

// maxUploadBytes bounds a single upload.
const maxUploadBytes = 8 << 30

// inlineTypes are the only content types served for display in the browser.
// Device files are untrusted as far as the dashboard is concerned: an HTML or
// SVG file rendered from this origin could script the dashboard, so anything
// not listed here is sent as a download.
var inlineTypes = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".webp": "image/webp", ".avif": "image/avif", ".bmp": "image/bmp", ".ico": "image/x-icon",
	".mp4": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime", ".mkv": "video/x-matroska",
	".mp3": "audio/mpeg", ".wav": "audio/wav", ".ogg": "audio/ogg", ".flac": "audio/flac",
	".m4a": "audio/mp4", ".opus": "audio/ogg",
	".pdf": "application/pdf",
}

// agentFile reads a file on a device through one HTTP transfer. It implements
// io.ReadSeeker so http.ServeContent can serve ranges, which is what lets a
// video seek without fetching the whole file.
type agentFile struct {
	tower    *Tower
	deviceID string
	path     string
	size     int64
	offset   int64
	current  *transfer
}

// Read returns the next piece of the device file. The first read opens a
// transfer for the rest of the file; the pipe then delivers it in whatever
// size the caller asks for.
func (f *agentFile) Read(buffer []byte) (int, error) {
	if f.offset >= f.size {
		return 0, io.EOF
	}
	if f.current == nil {
		tr, err := f.tower.startDownload(f.deviceID, f.path, f.offset, f.size-f.offset)
		if err != nil {
			return 0, err
		}
		f.current = tr
	}

	n, err := f.current.reader.Read(buffer)
	f.offset += int64(n)
	if err != nil {
		f.closeCurrent()
	}
	return n, err
}

// Seek moves the read position, enabling Range requests. A seek discards the
// open transfer; the next read starts a new one at the new position.
func (f *agentFile) Seek(offset int64, whence int) (int64, error) {
	f.closeCurrent()
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += f.offset
	case io.SeekEnd:
		offset += f.size
	default:
		return 0, errors.New("invalid whence")
	}
	if offset < 0 {
		return 0, errors.New("negative position")
	}
	f.offset = offset
	return offset, nil
}

// Close ends the current transfer, if any.
func (f *agentFile) Close() error {
	f.closeCurrent()
	return nil
}

func (f *agentFile) closeCurrent() {
	if f.current != nil {
		_ = f.current.reader.Close()
		f.current = nil
	}
}

func (t *Tower) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	path := r.URL.Query().Get("path")
	stat := t.performFile(w, deviceID, &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_STAT, Path: path})
	if stat == nil {
		return
	}
	if stat.GetIsDir() {
		httpapi.BadRequest(w, "cannot download a directory")
		return
	}

	name := filepath.Base(path)
	if contentType, ok := inlineTypes[strings.ToLower(filepath.Ext(name))]; ok {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": name}))
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	}
	file := &agentFile{tower: t, deviceID: deviceID, path: path, size: stat.GetSize()}
	defer file.Close()
	http.ServeContent(w, r, name, time.Unix(stat.GetModTimeUnix(), 0), file)
}

// handleUploadFile streams a multipart upload to the device. The body is never
// buffered whole, in memory or on the master's disk.
func (t *Tower) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		httpapi.BadRequest(w, "invalid upload")
		return
	}

	deviceID := r.PathValue("id")
	destination := "~"
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			httpapi.BadRequest(w, "file is required")
			return
		}
		if err != nil {
			httpapi.BadRequest(w, "invalid upload")
			return
		}
		switch part.FormName() {
		case "path":
			// The destination must precede the file so the upload can be
			// streamed; the dashboard sends it first.
			value, err := io.ReadAll(io.LimitReader(part, 4096))
			if err != nil {
				httpapi.BadRequest(w, "invalid upload")
				return
			}
			if trimmed := strings.TrimSpace(string(value)); trimmed != "" {
				destination = trimmed
			}
		case "file":
			// Browsers send a bare name, but the field is client-controlled:
			// keep only the final element so it cannot climb out of the
			// destination.
			name := filepath.Base(filepath.FromSlash(part.FileName()))
			if name == "." || name == ".." || name == string(filepath.Separator) {
				httpapi.BadRequest(w, "invalid file name")
				return
			}
			target := strings.TrimRight(destination, "/") + "/" + name
			t.uploadToDevice(w, r, deviceID, target, part)
			return
		}
	}
}

// uploadToDevice feeds one browser file part to the device and waits for the
// device to confirm that it wrote the file.
func (t *Tower) uploadToDevice(w http.ResponseWriter, r *http.Request, deviceID, path string, content io.Reader) {
	tr, err := t.startUpload(deviceID, path)
	if err != nil {
		writeAgentError(w, err)
		return
	}

	type copyResult struct {
		written int64
		err     error
	}
	copied := make(chan copyResult, 1)
	go func() {
		written, err := io.Copy(tr.writer, content)
		_ = tr.writer.CloseWithError(err)
		copied <- copyResult{written: written, err: err}
	}()

	var copy copyResult
	select {
	case copy = <-copied:
	case <-tr.done:
		// The device stopped early; unblock the copy and collect its result.
		_ = tr.reader.CloseWithError(errors.New("transfer ended"))
		copy = <-copied
	case <-r.Context().Done():
		_ = tr.reader.CloseWithError(r.Context().Err())
		<-copied
		return
	}

	var tooLarge *http.MaxBytesError
	if errors.As(copy.err, &tooLarge) {
		httpapi.Error(w, http.StatusRequestEntityTooLarge, "upload is too large")
		return
	}
	if err := tr.wait(); err != nil {
		httpapi.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if copy.err != nil {
		httpapi.BadRequest(w, "failed to read upload")
		return
	}
	httpapi.Success(w, map[string]interface{}{"filename": filepath.Base(path), "size": copy.written})
}
