package middleware

import (
	"errors"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/ss497254/gloski/internal/api/response"
)

// Default JSON body limit (1MB)
const DefaultJSONBodyLimit int64 = 1 * 1024 * 1024

var jsonBodyLimit atomic.Int64

func init() {
	jsonBodyLimit.Store(DefaultJSONBodyLimit)
}

// SetJSONBodyLimit sets the maximum size for JSON request bodies
func SetJSONBodyLimit(limit int64) {
	if limit > 0 {
		jsonBodyLimit.Store(limit)
	}
}

// GetJSONBodyLimit returns the current JSON body limit
func GetJSONBodyLimit() int64 {
	return jsonBodyLimit.Load()
}

// LimitJSONBody returns a middleware that limits request body size for JSON requests
// This only applies to requests with Content-Type: application/json
func LimitJSONBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")

		// Only limit JSON requests (not multipart/form-data for file uploads)
		if strings.HasPrefix(strings.ToLower(contentType), "application/json") {
			r.Body = http.MaxBytesReader(w, r.Body, jsonBodyLimit.Load())
		}

		next.ServeHTTP(w, r)
	})
}

// LimitRequestBody returns a middleware that limits all request bodies to the given size
func LimitRequestBody(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HandleMaxBytesError checks if an error is from MaxBytesReader and returns appropriate response
func HandleMaxBytesError(w http.ResponseWriter, err error) bool {
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			response.Error(w, http.StatusRequestEntityTooLarge, "request body too large")
			return true
		}
	}
	return false
}
