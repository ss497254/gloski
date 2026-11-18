package handlers

import (
	"net/http"
	"strings"
)

// websocketCredentials extracts API key / JWT credentials from a WebSocket
// handshake. Browsers cannot set arbitrary headers, so query parameters are
// supported; proxies such as the control tower can inject X-API-Key instead.
func websocketCredentials(r *http.Request) (apiKey, token string) {
	apiKey = strings.TrimSpace(r.Header.Get("X-API-Key"))
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}

	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(header, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	return apiKey, token
}
