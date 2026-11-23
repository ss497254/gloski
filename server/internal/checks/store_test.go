package checks

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/database"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "master.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewStore(db)
}

func sample(checkID, deviceID string, ranAt time.Time, status Status) Result {
	return Result{
		CheckID:    checkID,
		DeviceID:   deviceID,
		Status:     status,
		Summary:    "System drive 94% used",
		Details:    []Detail{{Key: "free", Value: "12 GB"}},
		RanAt:      ranAt,
		DurationMS: 250,
		Interval:   time.Minute,
	}
}

func TestStoreLatestHistoryAndPrune(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	now := time.Now()
	old := sample("disk", "dev1", now.Add(-3*time.Minute), StatusWarning)
	fresh := sample("disk", "dev1", now.Add(-10*time.Second), StatusCritical)
	fresh.Summary = "System drive 99% used"
	for _, result := range []Result{old, fresh} {
		if err := store.Save(ctx, result); err != nil {
			t.Fatal(err)
		}
	}

	latest, found, err := store.Latest(ctx, "disk", "dev1")
	if err != nil || !found {
		t.Fatalf("Latest() = %+v, %v, %v", latest, found, err)
	}
	if latest.Summary != fresh.Summary || latest.Status != StatusCritical {
		t.Fatalf("Latest() = %+v, want the new result", latest)
	}
	if latest.Stale {
		t.Fatal("a 10-second-old result with a 1-minute interval is not stale")
	}
	if len(latest.Details) != 1 || latest.Details[0].Value != "12 GB" {
		t.Fatalf("details did not round-trip: %+v", latest.Details)
	}
	if latest.DurationMS != 250 || latest.Interval != time.Minute {
		t.Fatalf("durations did not round-trip: %+v", latest)
	}

	history, err := store.History(ctx, "disk", "dev1", 10)
	if err != nil || len(history) != 2 {
		t.Fatalf("History() = %+v, %v; want two results", history, err)
	}
	if history[0].Summary != fresh.Summary {
		t.Fatalf("History()[0] = %+v, want newest first", history[0])
	}
	if history[0].Stale || !history[1].Stale {
		t.Fatalf("staleness was not recomputed on read: fresh=%v old=%v", history[0].Stale, history[1].Stale)
	}

	removed, err := store.Prune(ctx, 1)
	if err != nil || removed != 1 {
		t.Fatalf("Prune() = %d, %v; want 1", removed, err)
	}
	history, err = store.History(ctx, "disk", "dev1", 10)
	if err != nil || len(history) != 1 || history[0].Summary != fresh.Summary {
		t.Fatalf("history after prune = %+v, %v", history, err)
	}
}

func TestStoreLatestForDevice(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	now := time.Now()
	oldDisk := sample("disk", "dev1", now.Add(-3*time.Minute), StatusOK)
	freshDisk := sample("disk", "dev1", now.Add(-10*time.Second), StatusCritical)
	freshDisk.Summary = "fresh disk"
	oldUpdates := sample("updates", "dev1", now.Add(-4*time.Minute), StatusOK)
	freshUpdates := sample("updates", "dev1", now.Add(-3*time.Minute), StatusSkipped)
	oldOther := sample("disk", "dev2", now.Add(-4*time.Minute), StatusOK)
	freshOther := sample("disk", "dev2", now.Add(-3*time.Minute), StatusUnknown)
	for _, result := range []Result{oldDisk, freshDisk, oldUpdates, freshUpdates, oldOther, freshOther} {
		if err := store.Save(ctx, result); err != nil {
			t.Fatal(err)
		}
	}

	latest, err := store.LatestForDevice(ctx, "dev1")
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 2 {
		t.Fatalf("LatestForDevice() = %+v, want two checks", latest)
	}
	if latest[0].CheckID != "disk" || latest[1].CheckID != "updates" {
		t.Fatalf("LatestForDevice() = %+v, want check IDs in order", latest)
	}
	if latest[0].Summary != "fresh disk" || latest[0].Status != StatusCritical {
		t.Fatalf("LatestForDevice() did not pick the newest run: %+v", latest[0])
	}
	if latest[1].Status != StatusSkipped || !latest[1].Stale {
		t.Fatalf("latest updates result = %+v, want skipped and stale", latest[1])
	}

	// Prune works per pair, not by a global row count.
	removed, err := store.Prune(ctx, 1)
	if err != nil || removed != 3 {
		t.Fatalf("Prune() = %d, %v; want one old run for each of three pairs", removed, err)
	}
	for _, pair := range [][2]string{{"disk", "dev1"}, {"updates", "dev1"}, {"disk", "dev2"}} {
		history, err := store.History(ctx, pair[0], pair[1], 10)
		if err != nil || len(history) != 1 {
			t.Fatalf("History(%v) = %+v, %v; want one row", pair, history, err)
		}
	}

	missing, found, err := store.Latest(ctx, "updates", "dev2")
	if err != nil || found || missing.CheckID != "" {
		t.Fatalf("Latest() for a missing pair = %+v, %v, %v", missing, found, err)
	}
}
