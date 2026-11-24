package controltower

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSelfTestStatus(t *testing.T) {
	cases := []struct {
		name  string
		lines []selfTestLine
		want  string
	}{
		{"all skipped", []selfTestLine{{Status: diagSkipped}}, diagOK},
		{"warnings", []selfTestLine{{Status: diagOK}, {Status: diagWarning}}, diagWarning},
		{"critical wins", []selfTestLine{{Status: diagCritical}, {Status: diagWarning}}, diagCritical},
	}
	for _, tc := range cases {
		if got := selfTestStatus(tc.lines); got != tc.want {
			t.Fatalf("%s: status = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSelfTestEndpoint(t *testing.T) {
	tower := newRoutesTower(t)
	request := httptest.NewRequest(http.MethodGet, "/api/system/self-test", nil)
	request.Host = "127.0.0.1:8080"
	response := httptest.NewRecorder()
	tower.Router().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Success bool           `json:"success"`
		Data    selfTestResult `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success || envelope.Data.GeneratedAt.IsZero() {
		t.Fatalf("response = %s", response.Body.String())
	}
	if envelope.Data.Status != selfTestStatus(envelope.Data.Lines) {
		t.Fatalf("overall status = %q, lines = %+v", envelope.Data.Status, envelope.Data.Lines)
	}

	keys := make(map[string]selfTestLine)
	for _, line := range envelope.Data.Lines {
		if line.Summary == "" {
			t.Fatalf("line %q has no summary", line.Key)
		}
		keys[line.Key] = line
	}
	for _, key := range []string{"database", "size", "data-dir", "join-token", "free-space", "backups", "local-agent", "builds", "log", "maintenance", "runtime"} {
		if _, found := keys[key]; !found {
			t.Fatalf("missing self-test line %q (%+v)", key, keys)
		}
	}
	if keys["backups"].Status != diagSkipped || keys["log"].Status != diagSkipped {
		t.Fatalf("not-yet-applicable checks must be skipped: %+v", keys)
	}
	if keys["database"].Status != diagOK || keys["data-dir"].Status != diagOK || keys["join-token"].Status != diagOK {
		t.Fatalf("a fresh master should pass its basic checks: %+v", keys)
	}
}

func TestSelfTestSections(t *testing.T) {
	result := newRoutesTower(t).selfTest(context.Background())
	known := map[string]bool{sectionData: true, sectionAgents: true, sectionOther: true}
	seen := map[string]bool{}
	for _, line := range result.Lines {
		if !known[line.Section] {
			t.Errorf("line %q has unknown section %q", line.Key, line.Section)
		}
		if line.Section != selfTestSection(line.Key) {
			t.Errorf("line %q section = %q, want %q", line.Key, line.Section, selfTestSection(line.Key))
		}
		seen[line.Section] = true
	}
	if len(seen) < 2 {
		t.Fatalf("the report should span more than one section, got %+v", seen)
	}
}

func TestTrend(t *testing.T) {
	now := time.Now()
	if got := trend([]runtimeSample{{At: now}}); got == "" {
		t.Fatal("a short series must still explain itself")
	}
	rising := []runtimeSample{
		{Goroutines: 10, HeapMiB: 5},
		{Goroutines: 12, HeapMiB: 6},
		{Goroutines: 20, HeapMiB: 9},
	}
	if got := trend(rising); got != "Trend: goroutines and heap are rising across the last three runs." {
		t.Fatalf("trend = %q", got)
	}
}
