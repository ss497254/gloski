package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestLoggedJSONRedactsSecrets(t *testing.T) {
	body := []byte(`{"name":"ct","join_token":"abc","nested":{"password":"xyz","safe":"yes"},"ssh_key":"ssh-rsa AAA","items":[{"api_key":"nope"}]}`)
	got := loggedJSON(body)
	for _, hidden := range []string{"abc", "xyz", "nope", "ssh-rsa AAA"} {
		if strings.Contains(got, hidden) {
			t.Errorf("secret %q leaked into %s", hidden, got)
		}
	}
	for _, kept := range []string{"ct", "yes"} {
		if !strings.Contains(got, kept) {
			t.Errorf("ordinary value %q missing from %s", kept, got)
		}
	}
}

func TestLoggedJSONSkipsBinary(t *testing.T) {
	if got := loggedJSON([]byte{0, 1, 2, 3}); got != "<not-json>" {
		t.Errorf("loggedJSON() = %q, want <not-json>", got)
	}
}

func TestCaptureRequestBodyRestoresBody(t *testing.T) {
	original := `{"command":"echo hello"}`
	request := httptest.NewRequest(http.MethodPost, "/api/devices/d/api/jobs", strings.NewReader(original))
	request.Header.Set("Content-Type", "application/json")

	logged := captureRequestBody(request)
	if !strings.Contains(logged, "echo hello") {
		t.Errorf("logged request = %q", logged)
	}
	restored := new(bytes.Buffer)
	_, _ = restored.ReadFrom(request.Body)
	if restored.String() != original {
		t.Errorf("handler saw %q, want %q", restored.String(), original)
	}
}

func TestLogBodiesSkipsContentRoutes(t *testing.T) {
	for _, path := range []string{
		"/api/devices/d/api/files/read",
		"/api/devices/d/api/files/write",
		"/api/devices/d/api/files/download",
		"/api/devices/d/api/files/upload",
		"/api/devices/d/api/jobs/j/logs",
	} {
		if logBodies(path) {
			t.Errorf("logBodies(%q) = true, want false", path)
		}
	}
	if !logBodies("/api/devices/d/api/jobs") {
		t.Error("ordinary API route should still log bodies")
	}
}

func TestLoggedResponseMarksTruncatedJSON(t *testing.T) {
	writer := &responseWriter{capture: true, truncated: true}
	if got := loggedResponse(writer); !strings.Contains(got, "over 4096 bytes") {
		t.Errorf("loggedResponse() = %q, want a truncation marker", got)
	}
}

func TestLoggingAllowsWebSocketUpgrade(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade through Logging failed: %v", err)
			return
		}
		defer conn.Close()
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, message)
	})))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(message) != "hello" {
		t.Errorf("echoed %q, want hello", message)
	}
}

func TestLoggingSetsRequestIDHeader(t *testing.T) {
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		Success(w, map[string]string{"status": "ok"})
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/devices", nil))

	if id := recorder.Header().Get(requestIDHeader); id == "" {
		t.Error("no request id header")
	}
}

func TestLoggingKeepsCallerRequestID(t *testing.T) {
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		Success(w, map[string]string{"status": "ok"})
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/devices", nil)
	request.Header.Set(requestIDHeader, "from-browser")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if id := recorder.Header().Get(requestIDHeader); id != "from-browser" {
		t.Errorf("request id = %q, want from-browser", id)
	}
}
