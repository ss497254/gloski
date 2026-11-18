package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginAllowed(t *testing.T) {
	tests := []struct {
		name      string
		origin    string
		host      string
		forwarded string
		allowed   []string
		want      bool
	}{
		{name: "no origin is a non-browser client", host: "127.0.0.1:8080", want: true},
		{name: "same host", origin: "http://127.0.0.1:8080", host: "127.0.0.1:8080", want: true},
		{name: "default https port", origin: "https://lab.example.com", host: "lab.example.com:443", want: true},
		{name: "other site", origin: "https://evil.example", host: "127.0.0.1:8080", want: false},
		{name: "same host different port", origin: "http://127.0.0.1:3000", host: "127.0.0.1:8080", want: false},
		{name: "opaque origin", origin: "null", host: "127.0.0.1:8080", want: false},
		{name: "proxy forwarded host", origin: "https://lab.example.com", host: "127.0.0.1:8080", forwarded: "lab.example.com", want: true},
		{name: "proxy forwarded host mismatch", origin: "https://evil.example", host: "127.0.0.1:8080", forwarded: "lab.example.com", want: false},
		{name: "allowed origin", origin: "http://localhost:3000", host: "localhost:8080", allowed: []string{"http://localhost:3000"}, want: true},
		{name: "allowed bare host", origin: "https://lab.example.com", host: "127.0.0.1:8080", allowed: []string{"lab.example.com"}, want: true},
		{name: "wildcard", origin: "https://evil.example", host: "127.0.0.1:8080", allowed: []string{"*"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/x", nil)
			r.Host = tt.host
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if tt.forwarded != "" {
				r.Header.Set("X-Forwarded-Host", tt.forwarded)
			}
			if got := OriginAllowed(r, tt.allowed); got != tt.want {
				t.Fatalf("OriginAllowed = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSameOriginMiddleware(t *testing.T) {
	handler := SameOrigin(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	send := func(method string, websocket bool) int {
		r := httptest.NewRequest(method, "/api/x", nil)
		r.Host = "127.0.0.1:8080"
		r.Header.Set("Origin", "https://evil.example")
		if websocket {
			r.Header.Set("Upgrade", "websocket")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}
	if code := send(http.MethodGet, false); code != http.StatusNoContent {
		t.Fatalf("cross-origin GET = %d, want pass-through", code)
	}
	if code := send(http.MethodPost, false); code != http.StatusForbidden {
		t.Fatalf("cross-origin POST = %d, want 403", code)
	}
	if code := send(http.MethodGet, true); code != http.StatusForbidden {
		t.Fatalf("cross-origin WebSocket = %d, want 403", code)
	}
}
