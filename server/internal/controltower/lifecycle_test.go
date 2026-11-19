package controltower

import (
	"testing"
	"time"
)

func TestLifecycleDecide(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("a healthy agent is left alone and clears history", func(t *testing.T) {
		state := &deviceState{failures: 3, nextTry: start.Add(time.Hour), offlineSince: start}
		if reason := state.decide(start, true, false, true); reason != "" {
			t.Fatalf("decide() = %q", reason)
		}
		if state.failures != 0 || !state.nextTry.IsZero() || !state.offlineSince.IsZero() {
			t.Fatalf("state not reset: %+v", state)
		}
	})

	t.Run("a disconnected agent gets a grace period first", func(t *testing.T) {
		state := &deviceState{}
		if reason := state.decide(start, false, false, true); reason != "" {
			t.Fatal("acted the moment the agent dropped")
		}
		if reason := state.decide(start.Add(lifecycleGrace-time.Second), false, false, true); reason != "" {
			t.Fatal("acted inside the grace period")
		}
		if reason := state.decide(start.Add(lifecycleGrace), false, false, true); reason == "" {
			t.Fatal("did not act after the grace period")
		}
		if reason := state.decide(start.Add(2*lifecycleGrace), false, false, true); reason != "" {
			t.Fatal("started a second install while one was running")
		}
	})

	t.Run("backoff is respected", func(t *testing.T) {
		state := &deviceState{offlineSince: start, nextTry: start.Add(10 * time.Minute)}
		if reason := state.decide(start.Add(5*time.Minute), false, false, true); reason != "" {
			t.Fatal("acted before the next allowed attempt")
		}
		if reason := state.decide(start.Add(10*time.Minute), false, false, true); reason == "" {
			t.Fatal("did not act once the backoff ended")
		}
	})

	t.Run("an outdated agent is replaced only when idle", func(t *testing.T) {
		state := &deviceState{}
		if reason := state.decide(start, true, true, false); reason != "" {
			t.Fatal("replaced an agent that has running jobs")
		}
		if reason := state.decide(start, true, true, true); reason == "" {
			t.Fatal("did not replace an idle outdated agent")
		}
	})
}
