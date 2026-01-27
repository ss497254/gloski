package middleware

import (
	"net/http"

	"github.com/ss497254/gloski/internal/api/response"
)

// Default JSON body limit (1MB)
const DefaultJSONBodyLimit = 1 * 1024 * 1024

// JSONBodyLimit is the configured limit for JSON request bodies
var jsonBodyLimit int64 = DefaultJSONBodyLimit

// SetJSONBodyLimit sets the maximum size for JSON request bodies
func SetJSONBodyLimit(limit int64) {
	if limit > 0 {
		jsonBodyLimit = limit
	}
}

// GetJSONBodyLimit returns the current JSON body limit
func GetJSONBodyLimit() int64 {
	return jsonBodyLimit
}

// LimitJSONBody returns a middleware that limits request body size for JSON requests
// This only applies to requests with Content-Type: application/json
func LimitJSONBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")

		// Only limit JSON requests (not multipart/form-data for file uploads)
		if contentType == "application/json" || contentType == "application/json; charset=utf-8" {
			r.Body = http.MaxBytesReader(w, r.Body, jsonBodyLimit)
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
	if err != nil && err.Error() == "http: request body too large" {
		response.Error(w, http.StatusRequestEntityTooLarge, "request body too large")
		return true
	}
	return false
}
