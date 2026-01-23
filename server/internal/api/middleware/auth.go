package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ss497254/gloski/internal/api/response"
	"github.com/ss497254/gloski/internal/auth"
)

type contextKey string

const (
	// UserContextKey is the context key for authenticated user info
	UserContextKey contextKey = "user"
	// AuthMethodKey is the context key for the auth method used (jwt or apikey)
	AuthMethodKey contextKey = "auth_method"
)

// Auth returns a middleware that validates JWT tokens or API keys
func Auth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try API key first (X-API-Key header or api_key query param)
			apiKey := extractAPIKey(r)
			if apiKey != "" {
				if err := authService.ValidateAPIKey(apiKey); err == nil {
					ctx := context.WithValue(r.Context(), UserContextKey, "authenticated")
					ctx = context.WithValue(ctx, AuthMethodKey, "apikey")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// Invalid API key - don't fall back to JWT, return error
				response.Unauthorized(w, "invalid API key")
				return
			}

			// Try JWT token
			token := extractToken(r)
			if token == "" {
				response.Unauthorized(w, "missing authorization")
				return
			}

			if err := authService.ValidateToken(token); err != nil {
				if errors.Is(err, auth.ErrTokenExpired) {
					response.Unauthorized(w, "token expired")
				} else {
					response.Unauthorized(w, "invalid token")
				}
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, "authenticated")
			ctx = context.WithValue(ctx, AuthMethodKey, "jwt")
			next.ServeHTTP(w, r.WithContext(ctx))
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

// extractToken extracts the JWT token from the request
func extractToken(r *http.Request) string {
	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	// Fall back to query parameter (for WebSocket connections)
	return r.URL.Query().Get("token")
}
