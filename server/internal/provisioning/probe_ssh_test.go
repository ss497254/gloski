package provisioning

import (
	"context"
	"testing"
	"time"
)

func TestProbeReportsAgentState(t *testing.T) {
	f := newSSHFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	state, err := f.installer.Probe(ctx, f.target)
	if err != nil {
		t.Fatalf("Probe() on a bare device error = %v", err)
	}
	if state.Dir || state.Binary || state.Runnable || state.Running || state.PID != "" {
		t.Fatalf("Probe() on a bare device = %+v, want nothing installed", state)
	}

	f.install(t, false)

	state, err = f.installer.Probe(ctx, f.target)
	if err != nil {
		t.Fatalf("Probe() after install error = %v", err)
	}
	if !state.Dir || !state.Binary || !state.Runnable || !state.Running || state.PID == "" {
		t.Fatalf("Probe() after install = %+v, want a running agent", state)
	}
}
