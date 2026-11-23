package controltower

import (
	"context"
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/metrics"
)

func TestPruneJobsRecordsMaintenanceState(t *testing.T) {
	tower := newRoutesTower(t)
	last, err := tower.maintenanceState()
	if !last.IsZero() || err != nil {
		t.Fatalf("initial maintenance state = %v, %v; want unset", last, err)
	}

	ctx := context.Background()
	now := time.Now()
	for i := 0; i < checks.HistoryLimit+1; i++ {
		result := checks.Result{
			CheckID:  "disk",
			DeviceID: "dev",
			Status:   checks.StatusOK,
			Summary:  "ok",
			RanAt:    now.Add(-time.Duration(i) * time.Minute),
		}
		if err := tower.checkStore.Save(ctx, result); err != nil {
			t.Fatal(err)
		}
	}

	tower.pruneJobs()
	last, err = tower.maintenanceState()
	if err != nil || last.IsZero() || time.Since(last) > time.Minute {
		t.Fatalf("maintenance state after a successful run = %v, %v", last, err)
	}
	history, err := tower.checkStore.History(ctx, "disk", "dev", 100)
	if err != nil || len(history) != checks.HistoryLimit {
		t.Fatalf("history after prune = %d rows, %v; want %d", len(history), err, checks.HistoryLimit)
	}

	// A failed run must be recorded too, so the self-test does not claim
	// maintenance is fine just because it ran once.
	if err := tower.db.Close(); err != nil {
		t.Fatal(err)
	}
	tower.pruneJobs()
	_, err = tower.maintenanceState()
	if err == nil {
		t.Fatal("a failed maintenance run was not recorded")
	}
}

func TestPruneJobsThinsMetricSamples(t *testing.T) {
	tower := newRoutesTower(t)
	now := time.Now().UTC()
	insert := func(ts time.Time) {
		t.Helper()
		utc := ts.UTC()
		_, err := tower.db.DB().Exec(`
			INSERT INTO metric_samples (device_id, metric, ts, day, value)
			VALUES ('dev', ?, ?, ?, 1)`, metrics.MetricCPU, utc, utc.Format("2006-01-02"))
		if err != nil {
			t.Fatal(err)
		}
	}
	count := func() int {
		t.Helper()
		var n int
		if err := tower.db.DB().QueryRow(`SELECT COUNT(*) FROM metric_samples`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	for i := 0; i < 3; i++ {
		insert(now.Add(-time.Duration(i) * time.Hour))
	}
	for i := 0; i < 3; i++ {
		insert(now.AddDate(0, 0, -10).Add(-time.Duration(i) * time.Hour))
	}
	insert(now.AddDate(0, 0, -400))
	if got := count(); got != 7 {
		t.Fatalf("seeded samples = %d, want 7", got)
	}

	tower.pruneJobs()
	if _, err := tower.maintenanceState(); err != nil {
		t.Fatalf("maintenance after thinning failed: %v", err)
	}
	// Three recent samples, one for the old day, none expired.
	if got := count(); got != 4 {
		t.Fatalf("samples after prune = %d, want 4", got)
	}
}
