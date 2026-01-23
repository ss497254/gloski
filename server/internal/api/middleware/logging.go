package middleware

import (
	"net/http"
	"time"

	"github.com/ss497254/gloski/internal/logger"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging returns a middleware that logs HTTP requests
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		logger.Debug("%s %s %d %v", r.Method, r.URL.Path, wrapped.status, duration)
	})
}
