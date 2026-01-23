package middleware

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
		MaxAge:         86400,
	}
}

// CORS returns a middleware that handles CORS
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			allowedOrigin := ""
			for _, o := range config.AllowedOrigins {
				if o == "*" {
					allowed = true
					allowedOrigin = "*"
					break
				}
				if o == origin {
					allowed = true
					allowedOrigin = origin
					break
				}
			}

			// Set CORS headers
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", joinStrings(config.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", joinStrings(config.AllowedHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))

			// Handle preflight OPTIONS request
			// This must be handled BEFORE the router to avoid 404 on OPTIONS
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Wrap the response writer to preserve http.Hijacker for WebSocket upgrades
			next.ServeHTTP(&hijackableResponseWriter{w}, r)
		})
	}
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// hijackableResponseWriter wraps http.ResponseWriter and implements http.Hijacker
// This is needed for WebSocket connections to work through the CORS middleware
type hijackableResponseWriter struct {
	http.ResponseWriter
}

// Hijack implements http.Hijacker interface by delegating to the underlying ResponseWriter
func (w *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
