package controltower

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/fleet"
)

func TestDashboardIncludesCheckMatrix(t *testing.T) {
	tower := newRoutesTower(t)
	ids := map[string]string{}
	for _, name := range []string{"matrix-a", "matrix-b"} {
		registered, err := tower.fleetSvc.Register(fleet.RegisterRequest{
			JoinToken: tower.fleetSvc.JoinToken(),
			Name:      name,
		})
		if err != nil {
			t.Fatal(err)
		}
		ids[name] = registered.DeviceID
	}
	results := []checks.Result{
		{CheckID: "disk_root", DeviceID: ids["matrix-a"], Status: checks.StatusCritical, Summary: "full", RanAt: time.Now()},
		{CheckID: "ping_tcp", DeviceID: ids["matrix-b"], Status: checks.StatusSkipped, Summary: "not applicable", RanAt: time.Now()},
	}
	for _, result := range results {
		if err := tower.checkStore.Save(t.Context(), result); err != nil {
			t.Fatal(err)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	request.Host = "127.0.0.1:8080"
	response := httptest.NewRecorder()
	tower.Router().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, body = %s", response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			CheckMatrix checkMatrix `json:"check_matrix"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success || len(envelope.Data.CheckMatrix.Rows) != 2 {
		t.Fatalf("dashboard response = %s", response.Body.String())
	}
	if envelope.Data.CheckMatrix.Rows[0].DeviceName != "matrix-a" {
		t.Fatalf("critical device did not sort first: %+v", envelope.Data.CheckMatrix.Rows[0])
	}
	if len(envelope.Data.CheckMatrix.Groups) < 5 {
		t.Fatalf("matrix groups = %v", envelope.Data.CheckMatrix.Groups)
	}
}
