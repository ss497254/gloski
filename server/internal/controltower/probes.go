package controltower

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
)

// probeIdleAfter is how long the dashboard may be unused before scheduled
// probes stop. It only gates native probes; heartbeats and metric sampling
// continue.
const (
	probeIdleAfter = 3 * time.Minute
	// A probe's own timeout must fit inside the master's request deadline.
	maxProbeTimeout = 9 * time.Second
)

func (t *Tower) touchStatsAccess() {
	t.statsMu.Lock()
	t.lastStatsAccess = time.Now()
	t.statsMu.Unlock()
}

func (t *Tower) probesActive() bool {
	t.statsMu.Lock()
	defer t.statsMu.Unlock()
	return !t.lastStatsAccess.IsZero() && time.Since(t.lastStatsAccess) <= probeIdleAfter
}

// probeRunner schedules native probes from the heartbeat hook. It never opens
// a second channel and never writes anything but the existing check_results
// table.
type probeRunner struct {
	mu       sync.Mutex
	defs     []checks.Definition
	due      map[string]time.Time
	agg      map[string]*checks.Aggregator
	runCheck func(deviceID string, definition checks.Definition) (checks.Result, error)
}

func newProbeRunner(defs []checks.Definition, runCheck func(string, checks.Definition) (checks.Result, error)) *probeRunner {
	return &probeRunner{
		defs:     defs,
		due:      make(map[string]time.Time),
		agg:      make(map[string]*checks.Aggregator),
		runCheck: runCheck,
	}
}

func probeKey(deviceID, checkID string) string {
	return deviceID + "\x00" + checkID
}

// run sends the probes that are due for one device, aggregates each result and
// saves one bounded result per interval.
func (p *probeRunner) run(ctx context.Context, deviceID string, store *checks.Store) {
	now := time.Now()
	var due []checks.Definition
	p.mu.Lock()
	for _, definition := range p.defs {
		key := probeKey(deviceID, definition.ID())
		if now.Before(p.due[key]) {
			continue
		}
		p.due[key] = now.Add(definition.Interval)
		due = append(due, definition)
	}
	p.mu.Unlock()

	for _, definition := range due {
		result, err := p.runCheck(deviceID, definition)
		if err != nil {
			logger.Warn("Probe %s on %s failed: %v", definition.ID(), deviceID, err)
			continue
		}
		key := probeKey(deviceID, definition.ID())
		p.mu.Lock()
		aggregator := p.agg[key]
		if aggregator == nil {
			aggregator = &checks.Aggregator{}
			p.agg[key] = aggregator
		}
		aggregated := aggregator.Add(result, time.Now())
		p.mu.Unlock()
		if err := store.Save(ctx, aggregated); err != nil {
			logger.Warn("Could not save probe %s for %s: %v", definition.ID(), deviceID, err)
		}
	}
}

// probeDefinitions converts configured probes, skipping bad entries with a
// plain reason rather than guessing what the operator meant.
func probeDefinitions(configured []config.ProbeDefinition) []checks.Definition {
	definitions := make([]checks.Definition, 0, len(configured))
	for _, item := range configured {
		kind := checks.ProbeKind(strings.ToLower(strings.TrimSpace(item.Kind)))
		switch kind {
		case checks.ProbeTCP, checks.ProbeHTTP, checks.ProbeDNS, checks.ProbeTLS:
		default:
			logger.Warn("Skipping probe with unknown kind %q", item.Kind)
			continue
		}
		if strings.TrimSpace(item.Target) == "" {
			logger.Warn("Skipping %s probe with no target", kind)
			continue
		}
		definition := checks.Definition{
			Kind:       kind,
			Target:     item.Target,
			Accepted:   item.Accepted,
			SkipVerify: item.SkipVerify,
		}
		definition.Interval = checks.DefaultProbeInterval
		if item.Interval != "" {
			interval, err := time.ParseDuration(item.Interval)
			if err != nil || interval <= 0 {
				logger.Warn("Skipping %s probe %s: bad interval %q", kind, item.Target, item.Interval)
				continue
			}
			definition.Interval = interval
		}
		definition.Timeout = checks.DefaultProbeTimeout
		if item.Timeout != "" {
			timeout, err := time.ParseDuration(item.Timeout)
			if err != nil || timeout <= 0 {
				logger.Warn("Skipping %s probe %s: bad timeout %q", kind, item.Target, item.Timeout)
				continue
			}
			if timeout > maxProbeTimeout {
				logger.Warn("Clamping %s probe %s timeout to %s", kind, item.Target, maxProbeTimeout)
				timeout = maxProbeTimeout
			}
			definition.Timeout = timeout
		}
		definitions = append(definitions, definition)
	}
	return definitions
}
