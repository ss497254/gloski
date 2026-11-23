package checks

import (
	"testing"
	"time"
)

func TestParseOutputStatusFromExit(t *testing.T) {
	cases := []struct {
		code int
		want Status
	}{
		{0, StatusOK},
		{1, StatusWarning},
		{2, StatusCritical},
		{3, StatusUnknown},
		{127, StatusUnknown},
	}
	for _, tc := range cases {
		result := ParseOutput("disk", "dev", "System drive 94% used\nfree=12 GB", tc.code,
			time.Now(), 250*time.Millisecond, time.Minute)
		if result.Status != tc.want {
			t.Fatalf("exit %d: status = %q, want %q", tc.code, result.Status, tc.want)
		}
		if result.Summary != "System drive 94% used" {
			t.Fatalf("exit %d: summary = %q", tc.code, result.Summary)
		}
		if len(result.Details) != 1 || result.Details[0].Key != "free" || result.Details[0].Value != "12 GB" {
			t.Fatalf("exit %d: details = %+v", tc.code, result.Details)
		}
		if result.DurationMS != 250 {
			t.Fatalf("exit %d: duration = %d ms, want 250", tc.code, result.DurationMS)
		}
	}
}

func TestParseOutputSkipWinsOverExitCode(t *testing.T) {
	result := ParseOutput("battery", "dev", "GLOSKI-SKIP: no battery on this machine\nignored=value",
		2, time.Now(), time.Second, time.Minute)
	if result.Status != StatusSkipped {
		t.Fatalf("status = %q, want skipped", result.Status)
	}
	if result.Summary != "no battery on this machine" {
		t.Fatalf("summary = %q", result.Summary)
	}
	if len(result.Details) != 0 {
		t.Fatalf("skipped result has details: %+v", result.Details)
	}
}

func TestParseOutputDetailsKeepOrderAndPlainLines(t *testing.T) {
	output := "One summary line\nused: 94%\nfree=12 GB\nraw note\n\n"
	result := ParseOutput("disk", "dev", output, 1, time.Now(), 0, 0)
	want := []Detail{
		{Key: "used", Value: "94%"},
		{Key: "free", Value: "12 GB"},
		{Value: "raw note"},
	}
	if len(result.Details) != len(want) {
		t.Fatalf("details = %+v, want %+v", result.Details, want)
	}
	for i := range want {
		if result.Details[i] != want[i] {
			t.Fatalf("details[%d] = %+v, want %+v", i, result.Details[i], want[i])
		}
	}
}

func TestParseOutputWithNoOutputIsUnknown(t *testing.T) {
	result := ParseOutput("ping", "dev", "  \n", 0, time.Now(), 0, 0)
	if result.Status != StatusUnknown || result.Summary == "" {
		t.Fatalf("empty result = %+v, want unknown with a summary", result)
	}
}

func TestStale(t *testing.T) {
	now := time.Now()
	interval := time.Minute
	makeResult := func(ranAt time.Time) Result {
		return Result{RanAt: ranAt, Interval: interval}
	}
	if makeResult(now.Add(-3*time.Minute)).StaleAt(now) != true {
		t.Fatal("a result older than twice the interval must be stale")
	}
	if makeResult(now.Add(-30 * time.Second)).StaleAt(now) {
		t.Fatal("a fresh result must not be stale")
	}
	if (Result{RanAt: now.Add(-time.Hour)}).StaleAt(now) {
		t.Fatal("a result with no known schedule cannot be called stale")
	}
}
