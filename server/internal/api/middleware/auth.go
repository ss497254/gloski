package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/api/response"
	"github.com/ss497254/gloski/internal/auth"
)

type contextKey string

const (
	// UserContextKey is the context key for authenticated user info
	UserContextKey contextKey = "user"
)

// Auth returns a middleware that validates API keys or JWT tokens
func Auth(authService *auth.Service) func(http.Handler) http.Handler {
	// Rate limit auth failures: 10 attempts per minute per IP
	authLimiter := NewRateLimiter(10, 1*time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			// Check auth rate limit before processing
			if !authLimiter.Allow(ip) {
				w.Header().Set("Retry-After", "60")
				response.Error(w, http.StatusTooManyRequests, "too many authentication attempts")
				return
			}

			// Try API key first
			if apiKey := extractAPIKey(r); apiKey != "" {
				if err := authService.ValidateAPIKey(apiKey); err == nil {
					ctx := context.WithValue(r.Context(), UserContextKey, "api_key")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Try JWT token
			if token := extractToken(r); token != "" {
				if err := authService.ValidateToken(token); err == nil {
					ctx := context.WithValue(r.Context(), UserContextKey, "jwt")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			response.Unauthorized(w, "invalid or missing authentication")
		})
	}
}

// extractAPIKey extracts API key from request
func extractAPIKey(r *http.Request) string {
	// Check X-API-Key header first
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}
	// Fall back to query parameter
	return r.URL.Query().Get("api_key")
}

// extractToken extracts JWT token from request
func extractToken(r *http.Request) string {
	// Check Authorization header (Bearer token)
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	// Fall back to query parameter
	return r.URL.Query().Get("token")
}
