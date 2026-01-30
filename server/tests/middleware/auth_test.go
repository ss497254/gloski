package middleware_test

import (
	"net/http"
	"testing"

	"github.com/ss497254/gloski/internal/api/middleware"
	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/tests/testutil"
)

func TestAuthMiddleware(t *testing.T) {
	// Create auth service with test API key
	cfg := &config.Config{
		APIKey: "test-secret-key",
	}
	authService, err := auth.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create auth service: %v", err)
	}

	// Create a test handler that requires auth
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	// Wrap with auth middleware
	authMiddleware := middleware.Auth(authService)
	protectedHandler := authMiddleware(testHandler)

	tests := []struct {
		name       string
		headers    map[string]string
		path       string
		wantStatus int
	}{
		{
			name: "valid API key in header",
			headers: map[string]string{
				"X-API-Key": "test-secret-key",
			},
			path:       "/api/test",
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid API key in query",
			headers:    map[string]string{},
			path:       "/api/test?api_key=test-secret-key",
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid API key",
			headers: map[string]string{
				"X-API-Key": "wrong-key",
			},
			path:       "/api/test",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing authentication",
			headers:    map[string]string{},
			path:       "/api/test",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "empty API key",
			headers: map[string]string{
				"X-API-Key": "",
			},
			path:       "/api/test",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := testutil.MakeRequest(t, protectedHandler, testutil.HTTPRequest{
				Method:  http.MethodGet,
				Path:    tt.path,
				Headers: tt.headers,
			})

			testutil.AssertStatus(t, w.Code, tt.wantStatus)

			if tt.wantStatus == http.StatusOK {
				body := w.Body.String()
				if body != "authenticated" {
					t.Errorf("body = %s, want 'authenticated'", body)
				}
			}
		})
	}
}

func TestAuthMiddleware_ContextValue(t *testing.T) {
	cfg := &config.Config{
		APIKey: "test-key",
	}
	authService, err := auth.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create auth service: %v", err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user context was set
		user := r.Context().Value(middleware.UserContextKey)
		if user == nil {
			t.Error("user context not set")
		}
		if user != "api_key" {
			t.Errorf("user context = %v, want 'api_key'", user)
		}
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := middleware.Auth(authService)
	protectedHandler := authMiddleware(testHandler)

	w := testutil.MakeRequest(t, protectedHandler, testutil.HTTPRequest{
		Method: http.MethodGet,
		Path:   "/test",
		Headers: map[string]string{
			"X-API-Key": "test-key",
		},
	})

	testutil.AssertStatus(t, w.Code, http.StatusOK)
}
