package metrics

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/system"
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

func seed(t *testing.T, store *Store, device, metric string, ts time.Time, value float64) {
	t.Helper()
	utc := ts.UTC()
	_, err := store.db.Exec(`
		INSERT INTO metric_samples (device_id, metric, ts, day, value) VALUES (?, ?, ?, ?, ?)`,
		device, metric, utc, dayKey(utc), value)
	if err != nil {
		t.Fatal(err)
	}
}

func count(t *testing.T, store *Store) int {
	t.Helper()
	var n int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM metric_samples`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMigrationCreatesTableAndIndex(t *testing.T) {
	store := newTestStore(t)
	for _, name := range []string{"metric_samples", "idx_metric_samples_series"} {
		var found string
		err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE name = ?`, name).Scan(&found)
		if err != nil || found != name {
			t.Fatalf("%s not found: %v", name, err)
		}
	}
}

func TestRecordThrottlesBySampleInterval(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	samples := []Sample{{Metric: MetricCPU, Value: 42}}

	written, err := store.Record(ctx, "dev", now, samples)
	if err != nil || !written {
		t.Fatalf("first record = %v, %v; want true", written, err)
	}
	if got := count(t, store); got != 1 {
		t.Fatalf("rows after first record = %d, want 1", got)
	}

	written, err = store.Record(ctx, "dev", now.Add(SampleInterval/2), samples)
	if err != nil || written {
		t.Fatalf("early record = %v, %v; want false", written, err)
	}
	if got := count(t, store); got != 1 {
		t.Fatalf("rows after early record = %d, want 1", got)
	}

	written, err = store.Record(ctx, "dev", now.Add(SampleInterval+time.Second), samples)
	if err != nil || !written {
		t.Fatalf("later record = %v, %v; want true", written, err)
	}
	if got := count(t, store); got != 2 {
		t.Fatalf("rows after later record = %d, want 2", got)
	}

	// A different device is sampled independently.
	written, err = store.Record(ctx, "other", now, samples)
	if err != nil || !written {
		t.Fatalf("other device record = %v, %v; want true", written, err)
	}
}

func TestRecordThrottlesEachMetricIndependently(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := store.Record(ctx, "dev", now, []Sample{{Metric: MetricCPU, Value: 1}}); err != nil {
		t.Fatal(err)
	}
	// A metric that first appears inside the interval must still be written.
	written, err := store.Record(ctx, "dev", now.Add(time.Minute), []Sample{
		{Metric: MetricCPU, Value: 2},
		{Metric: MetricMemory, Value: 50},
		{Metric: MetricSwap, Value: 10},
	})
	if err != nil || !written {
		t.Fatalf("second record = %v, %v; want true", written, err)
	}
	for metric, want := range map[string]int{MetricCPU: 1, MetricMemory: 1, MetricSwap: 1} {
		var n int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM metric_samples WHERE metric = ?`, metric).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != want {
			t.Fatalf("rows for %s = %d, want %d", metric, n, want)
		}
	}
}

func TestHistoryIsChronologicalAndFiltered(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	seed(t, store, "dev", MetricCPU, now.Add(-3*time.Hour), 10)
	seed(t, store, "dev", MetricCPU, now.Add(-1*time.Hour), 30)
	seed(t, store, "dev", MetricLoad, now.Add(-2*time.Hour), 2)
	seed(t, store, "other", MetricCPU, now.Add(-1*time.Hour), 99)

	got, err := store.History(ctx, "dev", MetricCPU, 6*time.Hour, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("history length = %d, want 2", len(got))
	}
	if got[0].Value != 10 || got[1].Value != 30 {
		t.Fatalf("history values = %v, %v; want chronological [10 30]", got[0].Value, got[1].Value)
	}
	if got[0].TS.After(got[1].TS) {
		t.Fatal("history is not chronological")
	}

	all, err := store.History(ctx, "dev", "", 6*time.Hour, 100)
	if err != nil || len(all) != 3 {
		t.Fatalf("all metrics = %d rows, %v; want 3", len(all), err)
	}
	recent, err := store.History(ctx, "dev", MetricCPU, 2*time.Hour, 100)
	if err != nil || len(recent) != 1 {
		t.Fatalf("narrow range = %d rows, %v; want 1", len(recent), err)
	}
}

func TestHistoryReturnsNewestWhenTruncated(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		seed(t, store, "dev", MetricCPU, now.Add(-time.Duration(9-i)*time.Minute), float64(i))
	}

	got, err := store.History(ctx, "dev", MetricCPU, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("history length = %d, want 3", len(got))
	}
	if got[0].Value != 7 || got[2].Value != 9 {
		t.Fatalf("history should hold the newest three, got %v, %v, %v", got[0].Value, got[1].Value, got[2].Value)
	}
}

func TestThinKeepsRecentFullAndOnePerOldDay(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Recent samples stay at full resolution.
	for i := 0; i < 3; i++ {
		seed(t, store, "dev", MetricCPU, now.Add(-time.Duration(i)*time.Hour), float64(i))
	}
	// Older samples collapse to the earliest of each day.
	seed(t, store, "dev", MetricCPU, now.AddDate(0, 0, -10).Add(3*time.Hour), 1)
	seed(t, store, "dev", MetricCPU, now.AddDate(0, 0, -10).Add(2*time.Hour), 2)
	seed(t, store, "dev", MetricCPU, now.AddDate(0, 0, -10).Add(time.Hour), 3)
	seed(t, store, "dev", MetricLoad, now.AddDate(0, 0, -20).Add(5*time.Hour), 4)
	// Past the cap: gone.
	seed(t, store, "dev", MetricCPU, now.AddDate(0, 0, -400), 5)

	if got := count(t, store); got != 8 {
		t.Fatalf("seeded rows = %d, want 8", got)
	}
	removed, err := store.Thin(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 3 {
		t.Fatalf("thinned rows = %d, want 3", removed)
	}
	if got := count(t, store); got != 5 {
		t.Fatalf("rows after thinning = %d, want 5", got)
	}

	got, err := store.History(ctx, "dev", MetricCPU, 365*24*time.Hour, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range got {
		if sample.TS.Before(now.AddDate(0, 0, -365)) {
			t.Fatalf("expired sample survived: %v", sample.TS)
		}
	}
}

func TestSamplesFromStats(t *testing.T) {
	stats := &system.Stats{
		CPU:     system.CPUStats{UsagePercent: 25},
		Memory:  system.MemoryStats{UsedPercent: 60},
		Swap:    system.SwapStats{Total: 1, UsedPercent: 10},
		LoadAvg: system.LoadAvg{Load1: 1.5},
		Network: system.NetStats{RxRate: 100, TxRate: 200},
		Disks:   []system.DiskStats{{MountPoint: "/", Total: 100, UsedPercent: 40}},
	}

	samples := SamplesFromStats(stats)
	want := map[string]float64{
		MetricCPU: 25, MetricMemory: 60, MetricSwap: 10, MetricLoad: 1.5,
		MetricNetworkRX: 100, MetricNetworkTX: 200, DiskMetric("/"): 40,
	}
	if len(samples) != len(want) {
		t.Fatalf("got %d samples, want %d: %+v", len(samples), len(want), samples)
	}
	for _, sample := range samples {
		value, ok := want[sample.Metric]
		if !ok || value != sample.Value {
			t.Fatalf("unexpected sample: %+v", sample)
		}
	}
	mount, ok := DiskMountPoint(DiskMetric("/home"))
	if !ok || mount != "/home" {
		t.Fatalf("DiskMountPoint roundtrip = %q, %v", mount, ok)
	}
	if _, ok := DiskMountPoint(MetricCPU); ok {
		t.Fatal("a non-disk metric was treated as a disk")
	}
	if SamplesFromStats(nil) != nil {
		t.Fatal("nil stats should produce no samples")
	}
}
