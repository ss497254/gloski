package master

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/actions"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/fleet"
)

func newActionTestTower(t *testing.T) *Tower {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.DataDir = t.TempDir()
	cfg.Master.LocalAgent = false
	cfg.Master.AgentGRPCHost = "127.0.0.1"
	cfg.Master.AgentGRPCPort = freePort(t)
	cfg.Master.SSHKeyFile = filepath.Join(cfg.DataDir, "missing-key")
	tower, err := New(cfg, "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tower.Close() })
	return tower
}

func TestFailedStartIsRecorded(t *testing.T) {
	tower := newActionTestTower(t)
	deployment := &fleet.Deployment{
		ID: "dep", Name: "box", Host: "127.0.0.1", SSHPort: 1,
		AgentPort: 18081, MasterURL: "http://127.0.0.1:9090", Status: fleet.DeployStopped,
	}
	if err := tower.fleetSvc.CreateDeployment(deployment); err != nil {
		t.Fatal(err)
	}
	registered, err := tower.fleetSvc.Register(fleet.RegisterRequest{
		JoinToken: tower.fleetSvc.JoinToken(), Name: "box", DeployID: deployment.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/devices/"+registered.DeviceID+"/start", nil)
	request.Host = "127.0.0.1:8080"
	response := httptest.NewRecorder()
	tower.Router().ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("start status = %d, body = %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/devices/"+registered.DeviceID+"/actions", nil)
	request.Host = "127.0.0.1:8080"
	response = httptest.NewRecorder()
	tower.Router().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("actions status = %d, body = %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Data struct {
			Actions []actions.Action `json:"actions"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	list := envelope.Data.Actions
	if len(list) != 1 {
		t.Fatalf("actions = %+v, want one recorded start", list)
	}
	if list[0].Kind != actions.KindAgentStart || list[0].From != fleet.DeployStopped ||
		list[0].Result != actions.ResultFailed || !list[0].Undoable || list[0].Error == "" {
		t.Fatalf("recorded action = %+v", list[0])
	}
}

func TestUndoOperationDirection(t *testing.T) {
	cases := []struct {
		kind string
		want undoOperation
	}{
		{actions.KindAgentStop, undoStart},
		{actions.KindAgentRestart, undoStart},
		{actions.KindAgentStart, undoStop},
	}
	for _, tc := range cases {
		got, ok := undoOperationFor(tc.kind)
		if !ok || got != tc.want {
			t.Fatalf("undoOperationFor(%q) = %v, %v; want %v", tc.kind, got, ok, tc.want)
		}
	}
	if _, ok := undoOperationFor(actions.KindTrustHost); ok {
		t.Fatal("trust_host must not be undoable")
	}
}

func TestUndoRejectsNonReversibleAction(t *testing.T) {
	tower := newActionTestTower(t)
	device, err := tower.fleetSvc.Register(fleet.RegisterRequest{
		JoinToken: tower.fleetSvc.JoinToken(), Name: "box",
	})
	if err != nil {
		t.Fatal(err)
	}
	action := actions.Action{
		ID: uuid.NewString(), DeviceID: device.DeviceID, Kind: actions.KindTrustHost, Result: actions.ResultOK,
	}
	if err := tower.actionStore.Record(context.Background(), action); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost,
		"/api/devices/"+device.DeviceID+"/actions/"+action.ID+"/undo", nil)
	request.Host = "127.0.0.1:8080"
	response := httptest.NewRecorder()
	tower.Router().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("undo status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestFailedUndoIsRecordedAndOriginalStaysUndoable(t *testing.T) {
	tower := newActionTestTower(t)
	device, err := tower.fleetSvc.Register(fleet.RegisterRequest{
		JoinToken: tower.fleetSvc.JoinToken(), Name: "box",
	})
	if err != nil {
		t.Fatal(err)
	}
	original := actions.Action{
		ID: uuid.NewString(), DeviceID: device.DeviceID, Kind: actions.KindAgentStop,
		From: fleet.DeployRunning, To: fleet.DeployStopped, Result: actions.ResultOK, Undoable: true,
	}
	if err := tower.actionStore.Record(context.Background(), original); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost,
		"/api/devices/"+device.DeviceID+"/actions/"+original.ID+"/undo", nil)
	request.Host = "127.0.0.1:8080"
	response := httptest.NewRecorder()
	tower.Router().ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("undo status = %d, body = %s", response.Code, response.Body.String())
	}

	list, err := tower.actionStore.Recent(context.Background(), device.DeviceID, 10)
	if err != nil || len(list) != 2 {
		t.Fatalf("actions = %+v, %v; want original and failed undo", list, err)
	}
	if list[0].Kind != actions.KindUndo || list[0].Result != actions.ResultFailed || list[0].Error == "" {
		t.Fatalf("failed undo not recorded: %+v", list[0])
	}
	if list[1].ID != original.ID || list[1].UndoneAt != nil {
		t.Fatalf("original action should remain undoable: %+v", list[1])
	}
}

func TestUndoRejectsFailedAction(t *testing.T) {
	tower := newActionTestTower(t)
	device, err := tower.fleetSvc.Register(fleet.RegisterRequest{
		JoinToken: tower.fleetSvc.JoinToken(), Name: "box",
	})
	if err != nil {
		t.Fatal(err)
	}
	failed := actions.Action{
		ID: uuid.NewString(), DeviceID: device.DeviceID, Kind: actions.KindAgentStart,
		Result: actions.ResultFailed, Undoable: true,
	}
	if err := tower.actionStore.Record(context.Background(), failed); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost,
		"/api/devices/"+device.DeviceID+"/actions/"+failed.ID+"/undo", nil)
	request.Host = "127.0.0.1:8080"
	response := httptest.NewRecorder()
	tower.Router().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("undo status = %d, body = %s", response.Code, response.Body.String())
	}
}
