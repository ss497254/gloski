package httpapi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/logger"
)

// maxLoggedBody caps what a single request or response contributes to a log
// line. File contents, uploads and terminal traffic are never logged.
const maxLoggedBody = 4 << 10

// requestIDHeader carries the same id in the browser console and the master
// log, so one user action can be followed across both.
const requestIDHeader = "X-Request-ID"

// responseWriter captures the status and a bounded copy of a JSON response.
// It keeps Hijack and Flush so WebSockets and streaming still work.
type responseWriter struct {
	http.ResponseWriter
	status  int
	body    bytes.Buffer
	capture bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.status == 0 {
		rw.status = code
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	if rw.capture && rw.body.Len() < maxLoggedBody && strings.HasPrefix(rw.Header().Get("Content-Type"), "application/json") {
		_, _ = rw.body.Write(p[:min(len(p), maxLoggedBody-rw.body.Len())])
	}
	return rw.ResponseWriter.Write(p)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *responseWriter) code() int {
	if rw.status == 0 {
		return http.StatusOK
	}
	return rw.status
}

// Logging returns a middleware that records one line per API call: a request
// id, who called it, what went in, what came back, the status and the time it
// took. Read requests are debug-level; writes and failures are info-level so
// a user action is visible in the default log.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/health" || strings.HasPrefix(path, "/assets/") {
			next.ServeHTTP(w, r)
			return
		}

		id := r.Header.Get(requestIDHeader)
		if len(id) == 0 || len(id) > 64 || strings.ContainsAny(id, "\r\n") {
			id = uuid.NewString()
		}
		w.Header().Set(requestIDHeader, id)

		capture := logBodies(path)
		requestBody := "-"
		if capture {
			requestBody = captureRequestBody(r)
		}
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, capture: capture}
		next.ServeHTTP(wrapped, r)
		elapsed := time.Since(start)

		log := logger.Debug
		if r.Method != http.MethodGet || wrapped.code() >= http.StatusBadRequest {
			log = logger.Info
		}
		log("api id=%s %s %s status=%d %s req=%s res=%s",
			id, r.Method, path, wrapped.code(), elapsed, requestBody, loggedJSON(wrapped.body.Bytes()))
	})
}

// logBodies reports whether a route may have its JSON bodies logged. File
// contents and job output are user data: their metadata is useful in a log,
// but the bytes themselves are not.
func logBodies(path string) bool {
	if strings.Contains(path, "/api/files/read") ||
		strings.Contains(path, "/api/files/write") ||
		strings.Contains(path, "/api/files/download") ||
		strings.Contains(path, "/api/files/upload") ||
		strings.HasSuffix(path, "/logs") {
		return false
	}
	return true
}

// captureRequestBody reads a bounded prefix of a JSON body and puts it back
// so handlers still see the full request. Other content types, including
// multipart uploads, are left untouched.
func captureRequestBody(r *http.Request) string {
	if r.Body == nil {
		return "-"
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return "-"
	}
	prefix, err := io.ReadAll(io.LimitReader(r.Body, maxLoggedBody+1))
	if err != nil {
		return "-"
	}
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(prefix), r.Body))
	if len(prefix) > maxLoggedBody {
		prefix = prefix[:maxLoggedBody]
	}
	return loggedJSON(prefix)
}

// loggedJSON redacts secrets and caps the text. A body that is not JSON is
// described rather than printed, so a binary file never lands in the log.
func loggedJSON(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 {
		return "-"
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return "<not-json>"
	}
	cleaned, err := json.Marshal(redact(value))
	if err != nil {
		return "<not-json>"
	}
	text := string(cleaned)
	if len(text) > maxLoggedBody {
		text = text[:maxLoggedBody] + "…"
	}
	return text
}

// redact removes values of sensitive fields at any depth.
func redact(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if sensitiveKey(key) {
				typed[key] = "[redacted]"
			} else {
				typed[key] = redact(item)
			}
		}
	case []any:
		for i, item := range typed {
			typed[i] = redact(item)
		}
	}
	return value
}

func sensitiveKey(key string) bool {
	key = strings.ToLower(key)
	for _, word := range []string{"token", "password", "secret", "key", "authorization", "credential"} {
		if strings.Contains(key, word) {
			return true
		}
	}
	return false
}
