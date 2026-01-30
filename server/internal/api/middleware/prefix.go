package middleware

import (
	"net/http"
	"strings"
)

// Prefix returns a middleware that adds a prefix to all API routes
// If prefix is empty, it returns the handler unchanged
func Prefix(prefix string) Middleware {
	if prefix == "" {
		return func(h http.Handler) http.Handler {
			return h
		}
	}

	// Ensure prefix starts with / and doesn't end with /
	prefix = "/" + strings.Trim(prefix, "/")

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, prefix) {
				// Remove the prefix from the path
				r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
				// If the path becomes empty or just "/", ensure it's handled properly
				if r.URL.Path == "" {
					r.URL.Path = "/"
				}
				h.ServeHTTP(w, r)
			} else {
				// Path doesn't match the prefix, return 404
				http.NotFound(w, r)
			}
		})
	}
}
