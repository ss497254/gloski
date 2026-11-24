package controltower

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/database"
)

func TestProbeRunnerStoresOneResultPerInterval(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "master.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := checks.NewStore(db)
	definition := checks.Definition{Kind: checks.ProbeTCP, Target: "127.0.0.1:1", Interval: time.Hour}

	calls := 0
	runner := newProbeRunner([]checks.Definition{definition}, func(_ context.Context, deviceID string, def checks.Definition) (checks.Result, error) {
		calls++
		return checks.Result{
			CheckID: def.ID(), DeviceID: deviceID, Status: checks.StatusCritical,
			Summary: "down", RanAt: time.Now(), Interval: def.Interval,
		}, nil
	})
	ctx := context.Background()
	runner.run(ctx, "dev", store)
	runner.run(ctx, "dev", store) // same interval: must not run again
	if calls != 1 {
		t.Fatalf("probe ran %d times, want once per interval", calls)
	}
	results, err := store.LatestForAllDevices(ctx)
	if err != nil || len(results) != 1 {
		t.Fatalf("stored results = %d, %v; want 1", len(results), err)
	}
	if results[0].Status != checks.StatusCritical {
		t.Fatalf("stored status = %s, want critical", results[0].Status)
	}
}

func TestProbesActive(t *testing.T) {
	tower := &Tower{}
	if tower.probesActive() {
		t.Fatal("probes must be idle before anyone opens the dashboard")
	}
	tower.touchStatsAccess()
	if !tower.probesActive() {
		t.Fatal("probes must run while the dashboard is open")
	}
	tower.statsMu.Lock()
	tower.lastStatsAccess = time.Now().Add(-probeIdleAfter - time.Minute)
	tower.statsMu.Unlock()
	if tower.probesActive() {
		t.Fatal("probes must stop once the dashboard is idle")
	}
}

func TestProbeDefinitionsSkipBadEntries(t *testing.T) {
	definitions := probeDefinitions([]config.ProbeDefinition{
		{Kind: "tcp", Target: "host:80", Interval: "1m"},
		{Kind: "bogus", Target: "host:80"},
		{Kind: "dns", Target: ""},
		{Kind: "http", Target: "http://host", Interval: "soon"},
	})
	if len(definitions) != 1 {
		t.Fatalf("definitions = %d, want 1: %+v", len(definitions), definitions)
	}
	if definitions[0].Kind != checks.ProbeTCP || definitions[0].Interval != time.Minute {
		t.Fatalf("unexpected definition: %+v", definitions[0])
	}
}

func TestProbeDefinitionsApplyDefaults(t *testing.T) {
	definitions := probeDefinitions([]config.ProbeDefinition{{Kind: "tcp", Target: "host:80", Timeout: "30s"}})
	if len(definitions) != 1 {
		t.Fatalf("definitions = %d, want 1", len(definitions))
	}
	if definitions[0].Interval != checks.DefaultProbeInterval {
		t.Fatalf("interval = %s, want %s", definitions[0].Interval, checks.DefaultProbeInterval)
	}
	if definitions[0].Timeout != maxProbeTimeout {
		t.Fatalf("timeout = %s, want %s", definitions[0].Timeout, maxProbeTimeout)
	}
}
