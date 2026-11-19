package controltower

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/proto/agentv1"
)

const (
	// transferPiece is how much of a file crosses the agent stream per
	// request. Each piece is a round trip, so the master never holds more
	// than this in memory and a slow browser slows the transfer instead of
	// filling buffers.
	transferPiece = 1 << 20
	// maxUploadBytes bounds a single upload.
	maxUploadBytes = 8 << 30
)

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

// agentFile reads a file on a device piece by piece. It implements
// io.ReadSeeker so http.ServeContent can serve ranges, which is what lets a
// video seek without fetching the whole file.
type agentFile struct {
	tower    *Tower
	deviceID string
	path     string
	size     int64
	offset   int64
}

// Read returns the next piece of the device file.
func (f *agentFile) Read(buffer []byte) (int, error) {
	if f.offset >= f.size {
		return 0, io.EOF
	}
	length := int64(len(buffer))
	if length > transferPiece {
		length = transferPiece
	}
	result, err := f.tower.agentService.File(f.deviceID, &agentv1.FileRequest{
		Op: agentv1.FileOp_FILE_OP_READ_AT, Path: f.path, Offset: f.offset, Length: length,
	})
	if err != nil {
		return 0, err
	}
	if result.GetCode() != 0 {
		return 0, errors.New(result.GetMessage())
	}
	n := copy(buffer, result.GetContent())
	if n == 0 {
		// The file shrank after it was measured.
		return 0, io.ErrUnexpectedEOF
	}
	f.offset += int64(n)
	return n, nil
}

// Seek moves the read position, enabling Range requests.
func (f *agentFile) Seek(offset int64, whence int) (int64, error) {
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
	http.ServeContent(w, r, name, time.Unix(stat.GetModTimeUnix(), 0), file)
}

// handleUploadFile streams a multipart upload to the device in pieces. The
// body is never buffered whole, in memory or on the master's disk.
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
			size, err := t.streamUpload(deviceID, target, part)
			if err != nil {
				var agentErr fileOpError
				if errors.As(err, &agentErr) {
					httpapi.Error(w, http.StatusBadRequest, agentErr.Error())
					return
				}
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) {
					httpapi.Error(w, http.StatusRequestEntityTooLarge, "upload is too large")
					return
				}
				writeAgentError(w, err)
				return
			}
			httpapi.Success(w, map[string]interface{}{"filename": name, "size": size})
			return
		}
	}
}

// fileOpError is a failure reported by the device (permission denied, disk
// full) as opposed to a failure to reach it.
type fileOpError string

// Error returns the device's message for the failed operation.
func (e fileOpError) Error() string { return string(e) }

// streamUpload sends content to the device one piece at a time and returns
// the number of bytes written. The device assembles the pieces in a
// temporary file and only replaces the target when the last one arrives.
func (t *Tower) streamUpload(deviceID, path string, content io.Reader) (int64, error) {
	buffer := make([]byte, transferPiece)
	var offset int64
	for {
		n, readErr := io.ReadFull(content, buffer)
		final := errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF)
		if readErr != nil && !final {
			return offset, fmt.Errorf("failed to read upload: %w", readErr)
		}
		result, err := t.agentService.File(deviceID, &agentv1.FileRequest{
			Op: agentv1.FileOp_FILE_OP_WRITE_AT, Path: path, Offset: offset, Content: buffer[:n], Final: final,
		})
		if err != nil {
			return offset, err
		}
		if result.GetCode() != 0 {
			return offset, fileOpError(result.GetMessage())
		}
		offset += int64(n)
		if final {
			return offset, nil
		}
	}
}
