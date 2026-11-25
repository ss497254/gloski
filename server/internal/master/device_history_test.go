package master

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/metrics"
)

func TestDeviceHistory(t *testing.T) {
	tower := newRoutesTower(t)
	registered, err := tower.fleetSvc.Register(fleet.RegisterRequest{
		JoinToken: tower.fleetSvc.JoinToken(),
		Name:      "history-device",
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	if _, err := tower.metricStore.Record(t.Context(), registered.DeviceID, now, []metrics.Sample{
		{Metric: metrics.MetricCPU, Value: 12},
		{Metric: metrics.MetricMemory, Value: 60},
	}); err != nil {
		t.Fatal(err)
	}

	send := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Host = "127.0.0.1:8080"
		response := httptest.NewRecorder()
		tower.Router().ServeHTTP(response, request)
		return response
	}

	response := send("/api/devices/" + registered.DeviceID + "/history?metric=cpu&range=1h")
	if response.Code != http.StatusOK {
		t.Fatalf("history status = %d, body = %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Samples []metrics.Sample `json:"samples"`
			Metric  string           `json:"metric"`
			Range   string           `json:"range"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success || len(envelope.Data.Samples) != 1 {
		t.Fatalf("history response = %s", response.Body.String())
	}
	if envelope.Data.Samples[0].Metric != metrics.MetricCPU || envelope.Data.Samples[0].Value != 12 {
		t.Fatalf("unexpected sample: %+v", envelope.Data.Samples[0])
	}
	if envelope.Data.Metric != metrics.MetricCPU || envelope.Data.Range != "1h0m0s" {
		t.Fatalf("unexpected metadata: %+v", envelope.Data)
	}

	if got := send("/api/devices/" + registered.DeviceID + "/history?range=nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad range status = %d, want 400", got)
	}
	if got := send("/api/devices/" + registered.DeviceID + "/history?limit=0").Code; got != http.StatusBadRequest {
		t.Fatalf("bad limit status = %d, want 400", got)
	}
	if got := send("/api/devices/" + registered.DeviceID + "/history?limit=999999").Code; got != http.StatusBadRequest {
		t.Fatalf("oversized limit status = %d, want 400", got)
	}
	if got := send("/api/devices/missing/history").Code; got != http.StatusNotFound {
		t.Fatalf("unknown device status = %d, want 404", got)
	}
}
