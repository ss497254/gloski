package controltower

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ss497254/gloski/internal/actions"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/jobs"
)

func newRoutesTower(t *testing.T) *Tower {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.DataDir = t.TempDir()
	cfg.Master.LocalAgent = false
	cfg.Master.AgentGRPCHost = "127.0.0.1"
	cfg.Master.AgentGRPCPort = freePort(t)
	tower, err := New(cfg, "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tower.Close() })
	return tower
}

func TestCrossOriginRequestsAreRejected(t *testing.T) {
	tower := newRoutesTower(t)

	send := func(method, path, origin, contentType, body string, websocket bool) int {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Host = "127.0.0.1:8080"
		if origin != "" {
			request.Header.Set("Origin", origin)
		}
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}
		if websocket {
			request.Header.Set("Upgrade", "websocket")
			request.Header.Set("Connection", "Upgrade")
		}
		response := httptest.NewRecorder()
		tower.Router().ServeHTTP(response, request)
		return response.Code
	}

	const evil = "https://evil.example"
	const self = "http://127.0.0.1:8080"
	job := `{"command":"id"}`

	if code := send("GET", "/api/devices/x/api/terminal", evil, "", "", true); code != http.StatusForbidden {
		t.Errorf("cross-origin terminal WebSocket = %d, want 403", code)
	}
	if code := send("GET", "/api/fleet/ws", evil, "", "", true); code != http.StatusForbidden {
		t.Errorf("cross-origin fleet WebSocket = %d, want 403", code)
	}
	if code := send("POST", "/api/devices/x/api/jobs", evil, "application/json", job, false); code != http.StatusForbidden {
		t.Errorf("cross-origin job POST = %d, want 403", code)
	}
	// A form post from the dashboard's own origin still must be JSON.
	if code := send("POST", "/api/devices/x/api/jobs", self, "text/plain", job, false); code != http.StatusUnsupportedMediaType {
		t.Errorf("text/plain job POST = %d, want 415", code)
	}
	// Same-origin JSON reaches the handler; no agent is connected for "x".
	if code := send("POST", "/api/devices/x/api/jobs", self, "application/json", job, false); code != http.StatusBadGateway {
		t.Errorf("same-origin job POST = %d, want 502 (no agent)", code)
	}
	if code := send("GET", "/api/devices", evil, "", "", false); code != http.StatusOK {
		t.Errorf("cross-origin GET = %d, want 200", code)
	}
}

func TestWriteAgentErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"no agent", agents.ErrNoAgent, http.StatusBadGateway},
		{"wrapped timeout", fmt.Errorf("rpc: %w", agents.ErrRequestTimeout), http.StatusGatewayTimeout},
		{"deadline exceeded", context.DeadlineExceeded, http.StatusGatewayTimeout},
		{"missing device", fleet.ErrDeviceNotFound, http.StatusNotFound},
		{"missing deployment", fleet.ErrDeploymentNotFound, http.StatusNotFound},
		{"missing job", jobs.ErrJobNotFound, http.StatusNotFound},
		{"missing action", actions.ErrActionNotFound, http.StatusNotFound},
		{"not deployed", errNotDeployed, http.StatusBadRequest},
		{"stop unreachable", errStopUnreachable, http.StatusBadRequest},
		{"unexpected", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writeAgentError(recorder, tc.err)
			if recorder.Code != tc.want {
				t.Errorf("status = %d, want %d (body=%s)", recorder.Code, tc.want, recorder.Body.String())
			}
		})
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
