package master

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/tailnet"
)

func diagnoseRequest(t *testing.T, tower *Tower, deviceID string) diagnosis {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/devices/"+deviceID+"/diagnose", nil)
	request.Host = "127.0.0.1:8080"
	response := httptest.NewRecorder()
	tower.Router().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("diagnose status = %d, body = %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Success bool      `json:"success"`
		Data    diagnosis `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success {
		t.Fatalf("diagnose response = %s", response.Body.String())
	}
	return envelope.Data
}

func TestDiagnoseUnmanagedOfflineDevice(t *testing.T) {
	tower := newRoutesTower(t)
	registered, err := tower.fleetSvc.Register(fleet.RegisterRequest{
		JoinToken: tower.fleetSvc.JoinToken(),
		Name:      "lonely",
	})
	if err != nil {
		t.Fatal(err)
	}

	result := diagnoseRequest(t, tower, registered.DeviceID)
	if result.Status != diagCritical {
		t.Fatalf("status = %q, want critical", result.Status)
	}
	if result.Headline != "The agent has no live session with the master." {
		t.Fatalf("headline = %q", result.Headline)
	}
	if len(result.Steps) != 3 {
		t.Fatalf("steps = %+v, want deployment, agent and build", result.Steps)
	}
	if result.Steps[0].Key != "deployment" || result.Steps[0].Status != diagSkipped {
		t.Fatalf("deployment step = %+v, want skipped", result.Steps[0])
	}
	if result.Steps[1].Key != "agent" || result.Steps[1].Status != diagCritical {
		t.Fatalf("agent step = %+v, want critical", result.Steps[1])
	}
	if result.Steps[2].Key != "build" || result.Steps[2].Status != diagSkipped {
		t.Fatalf("build step = %+v, want skipped", result.Steps[2])
	}
}

func TestDiagnoseStoppedDeviceDoesNotUseSSH(t *testing.T) {
	tower := newRoutesTower(t)
	deployment := &fleet.Deployment{
		ID:        "deploy-stopped",
		Name:      "stopped box",
		Host:      "127.0.0.1",
		SSHPort:   1, // would fail if the diagnosis tried to reach it
		AgentPort: 18081,
		MasterURL: "http://127.0.0.1:9090",
		Status:    fleet.DeployStopped,
	}
	if err := tower.fleetSvc.CreateDeployment(deployment); err != nil {
		t.Fatal(err)
	}
	registered, err := tower.fleetSvc.Register(fleet.RegisterRequest{
		JoinToken: tower.fleetSvc.JoinToken(),
		Name:      "stopped box",
		DeployID:  deployment.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	result := diagnoseRequest(t, tower, registered.DeviceID)
	if result.Status != diagWarning {
		t.Fatalf("status = %q, want warning", result.Status)
	}
	if result.Headline != "The agent is stopped on purpose." {
		t.Fatalf("headline = %q", result.Headline)
	}
	for _, step := range result.Steps {
		if step.Key == "ssh" || step.Key == "login" {
			t.Fatalf("stopped device was probed over SSH: %+v", result.Steps)
		}
	}
}

func TestDiagnoseSSHUsesDefaultPort(t *testing.T) {
	tower := newRoutesTower(t)
	deployment := &fleet.Deployment{
		ID:        "deploy-defaults",
		Name:      "defaults",
		Host:      "127.0.0.1",
		AgentPort: 18081,
		MasterURL: "http://127.0.0.1:9090",
		Status:    fleet.DeployRunning,
	}
	result := &diagnosis{Steps: []diagnosisStep{}}
	tower.diagnoseSSH(t.Context(), deployment, result)

	found := false
	for _, step := range result.Steps {
		if step.Key == "ssh" && strings.Contains(step.Summary, "127.0.0.1:22") {
			found = true
		}
	}
	if !found {
		t.Fatalf("SSH step did not use the default port: %+v", result.Steps)
	}
}

func TestDiagnosisHeadlinePicksFirstCriticalStep(t *testing.T) {
	result := diagnosis{Steps: []diagnosisStep{
		{Status: diagSkipped, Summary: "not needed"},
		{Status: diagWarning, Summary: "warning"},
		{Status: diagCritical, Summary: "the cause", Advice: "fix it"},
		{Status: diagCritical, Summary: "a later symptom"},
	}}
	result.finish()
	if result.Status != diagCritical || result.Headline != "the cause" || result.Advice != "fix it" {
		t.Fatalf("finish() = %+v", result)
	}
}

func TestFindTailnetPeer(t *testing.T) {
	peers := []tailnet.Node{{
		Hostname: "worker",
		DNSName:  "worker.tailnet.ts.net.",
		IPs:      []string{"100.64.0.2", "fd7a::2"},
	}}
	for _, host := range []string{"worker", "WORKER.tailnet.ts.net", "100.64.0.2", "fd7a::2"} {
		if _, found := findTailnetPeer(peers, host); !found {
			t.Fatalf("findTailnetPeer(%q) did not match", host)
		}
	}
	if _, found := findTailnetPeer(peers, "192.168.1.5"); found {
		t.Fatal("findTailnetPeer matched an unrelated address")
	}
}
