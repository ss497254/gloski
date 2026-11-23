package controltower

import (
	"context"
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/checks"
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
