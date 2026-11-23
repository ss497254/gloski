package system

import "testing"

func TestCounterDelta(t *testing.T) {
	cases := []struct {
		name              string
		current, previous uint64
		want              uint64
		wantOK            bool
	}{
		{"increase", 100, 40, 60, true},
		{"unchanged", 100, 100, 0, true},
		{"rollback", 30, 40, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := counterDelta(tc.current, tc.previous)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("counterDelta(%d, %d) = (%d, %v), want (%d, %v)",
					tc.current, tc.previous, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestCPUStatsFromTimes(t *testing.T) {
	previous := CPUTimes{User: 100, System: 50, Idle: 800, IOWait: 50, Total: 1000}
	current := CPUTimes{User: 200, System: 100, Idle: 1600, IOWait: 100, Total: 2000}
	prevCores := []CPUTimes{{Idle: 80, Total: 100}, {Idle: 90, Total: 100}}
	cores := []CPUTimes{{Idle: 160, Total: 200}, {Idle: 180, Total: 200}}

	stats := cpuStatsFromTimes(current, previous, cores, prevCores)

	if stats.User != 10 || stats.System != 5 || stats.Idle != 80 || stats.IOWait != 5 {
		t.Fatalf("unexpected breakdown: %+v", stats)
	}
	if stats.UsagePercent != 20 {
		t.Fatalf("usage = %v, want 20", stats.UsagePercent)
	}
	if len(stats.PerCore) != 2 || stats.PerCore[0] != 20 || stats.PerCore[1] != 10 {
		t.Fatalf("per-core = %v, want [20 10]", stats.PerCore)
	}
}

func TestCPUStatsFromTimesRollback(t *testing.T) {
	previous := CPUTimes{User: 500, System: 200, Idle: 1200, IOWait: 100, Total: 2000}
	current := CPUTimes{User: 100, System: 50, Idle: 800, IOWait: 50, Total: 1000}

	stats := cpuStatsFromTimes(current, previous, nil, nil)

	if stats.UsagePercent != 0 || stats.User != 0 || stats.System != 0 || stats.Idle != 0 {
		t.Fatalf("rollback should produce zero values, got %+v", stats)
	}
}

func TestCPUStatsFromTimesPerCoreRollback(t *testing.T) {
	previous := CPUTimes{Total: 1000, Idle: 800}
	current := CPUTimes{Total: 2000, Idle: 1600}
	prevCores := []CPUTimes{{Total: 500, Idle: 400}, {Total: 500, Idle: 400}}
	cores := []CPUTimes{{Total: 200, Idle: 160}, {Total: 1000, Idle: 800}}

	stats := cpuStatsFromTimes(current, previous, cores, prevCores)

	if len(stats.PerCore) != 2 {
		t.Fatalf("per-core length = %d, want 2", len(stats.PerCore))
	}
	if stats.PerCore[0] != 0 {
		t.Fatalf("rolled-back core = %v, want 0", stats.PerCore[0])
	}
	if stats.PerCore[1] != 20 {
		t.Fatalf("healthy core = %v, want 20", stats.PerCore[1])
	}
}

func TestByteRateRollback(t *testing.T) {
	if got := byteRate(100, 200, 2); got != 0 {
		t.Fatalf("rollback rate = %v, want 0", got)
	}
	if got := byteRate(300, 100, 2); got != 100 {
		t.Fatalf("rate = %v, want 100", got)
	}
	if got := byteRate(300, 100, 0); got != 0 {
		t.Fatalf("zero elapsed rate = %v, want 0", got)
	}
}
