package controltower

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/provisioning"
)

// The master owns every agent's lifecycle. Agents do not restart, update or
// reinstall themselves; this loop does it for them over SSH. For each device
// that should be running it asks one question, "is its agent connected and on
// the master's build?", and if not it runs the install script, which is safe
// to repeat. That one action repairs a crash, a reboot, a wiped install
// directory, an old build and an agent the master refused as incompatible.

const (
	lifecycleInterval = 15 * time.Second
	// lifecycleGrace is how long a device may be disconnected before the
	// master steps in. Agents reconnect by themselves within seconds after a
	// network blip or a master restart, and must be left alone to do so.
	lifecycleGrace      = 45 * time.Second
	lifecycleMinBackoff = time.Minute
	lifecycleMaxBackoff = 15 * time.Minute
)

type lifecycle struct {
	tower *Tower

	mu     sync.Mutex
	states map[string]*deviceState // by deployment ID
}

type deviceState struct {
	offlineSince time.Time
	nextTry      time.Time
	failures     int
	busy         bool
}

func newLifecycle(tower *Tower) *lifecycle {
	return &lifecycle{tower: tower, states: make(map[string]*deviceState)}
}

func (l *lifecycle) loop() {
	ticker := time.NewTicker(lifecycleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-l.tower.stop:
			return
		case <-ticker.C:
			l.tick(time.Now())
		}
	}
}

func (l *lifecycle) tick(now time.Time) {
	t := l.tower
	deployments, err := t.fleetSvc.ListDeployments()
	if err != nil {
		logger.Warn("lifecycle: cannot list deployments: %v", err)
		return
	}

	seen := make(map[string]bool, len(deployments))
	for i := range deployments {
		deployment := deployments[i]
		seen[deployment.ID] = true
		if deployment.Status == fleet.DeployStopped {
			continue // stopped on purpose; leave it alone
		}
		if deployment.DeviceID == "" {
			// Never connected: a first install that failed. The person who
			// asked for it sees the error and decides; do not retry forever.
			continue
		}

		connected := deployment.DeviceID != "" && t.agentService.Connected(deployment.DeviceID)

		l.mu.Lock()
		state := l.states[deployment.ID]
		if state == nil {
			// A master that just started treats every device as freshly
			// disconnected, which gives agents the grace period to return.
			state = &deviceState{}
			l.states[deployment.ID] = state
		}
		outdated := connected && t.agentOutdated(deployment.DeviceID)
		idle := !outdated || t.agentService.ActiveJobs(deployment.DeviceID) == 0
		reason := state.decide(now, connected, outdated, idle)
		act := reason != ""
		l.mu.Unlock()

		if connected && !act && deployment.Status != fleet.DeployRunning {
			_ = t.fleetSvc.SetDeploymentRunning(deployment.ID, deployment.RemotePID)
		}
		if act {
			logger.Info("lifecycle: installing agent on %s (%s) because %s", deployment.Name, deployment.Host, reason)
			go func() { _, _ = l.install(&deployment, false, true) }()
		}
	}

	l.mu.Lock()
	for id := range l.states {
		if !seen[id] {
			delete(l.states, id)
		}
	}
	l.mu.Unlock()
}

// Install runs the install script for one deployment right now, as asked for
// by a person. force restarts the agent even if nothing changed.
func (l *lifecycle) Install(deployment *fleet.Deployment, force bool) (provisioning.Result, error) {
	l.mu.Lock()
	state := l.states[deployment.ID]
	if state == nil {
		state = &deviceState{}
		l.states[deployment.ID] = state
	}
	if state.busy {
		l.mu.Unlock()
		return provisioning.Result{}, fmt.Errorf("an install is already running on %s", deployment.Name)
	}
	state.busy = true
	l.mu.Unlock()
	return l.install(deployment, force, false)
}

// install does the work; the caller has already marked the state busy.
func (l *lifecycle) install(deployment *fleet.Deployment, force, automatic bool) (provisioning.Result, error) {
	t := l.tower
	timeout := time.Duration(t.cfg.Master.DeployTimeout) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(t.ctx, timeout)
	defer cancel()

	result, err := t.installer.Install(ctx, t.deploymentTarget(deployment), force)

	message := ""
	switch {
	case err != nil:
		message = err.Error()
	case automatic && !result.Started:
		// The agent process is alive on the device yet has no connection.
		message = fmt.Sprintf("the agent is running on the device but has not connected to the master at %s; check that the device can reach that address", t.agentGRPCURL())
		if rejected := t.agentService.Rejection(deployment.Name); rejected != "" {
			message = "the master refused the agent: " + rejected
		}
	}

	now := time.Now()
	l.mu.Lock()
	state := l.states[deployment.ID]
	if state == nil {
		state = &deviceState{}
		l.states[deployment.ID] = state
	}
	state.busy = false
	// Back off whether or not the script succeeded: success only counts once
	// the agent connects, which resets all of this.
	state.failures++
	backoff := lifecycleMinBackoff << min(state.failures-1, 8)
	if backoff > lifecycleMaxBackoff {
		backoff = lifecycleMaxBackoff
	}
	state.nextTry = now.Add(backoff)
	state.offlineSince = now
	l.mu.Unlock()

	if message != "" {
		logger.Warn("lifecycle: %s (%s): %s", deployment.Name, deployment.Host, message)
		if statusErr := t.fleetSvc.SetDeploymentStatus(deployment.ID, fleet.DeployFailed, message); statusErr != nil {
			logger.Warn("lifecycle: cannot record failure for %s: %v", deployment.Name, statusErr)
		}
		if err == nil {
			err = fmt.Errorf("%s", message)
		}
		return result, err
	}
	if statusErr := t.fleetSvc.SetDeploymentRunning(deployment.ID, result.PID); statusErr != nil {
		logger.Warn("lifecycle: cannot record install for %s: %v", deployment.Name, statusErr)
	}
	logger.Info("lifecycle: agent on %s (%s) is installed and running (pid %s, %s)", deployment.Name, deployment.Host, result.PID, result.Arch)
	return result, nil
}

// agentOutdated reports whether a connected agent runs a different build from
// the one the master would install on that device.
func (t *Tower) agentOutdated(deviceID string) bool {
	arch, hash, connected := t.agentService.Build(deviceID)
	if !connected || hash == "" {
		return false
	}
	build := t.dist.Build(arch)
	return build != nil && build.SHA256 != hash
}

func (t *Tower) deploymentTarget(deployment *fleet.Deployment) provisioning.Target {
	return provisioning.Target{
		Name:      deployment.Name,
		Host:      deployment.Host,
		User:      deployment.User,
		SSHPort:   deployment.SSHPort,
		AgentPort: deployment.AgentPort,
		// Always the master's current address, which may have changed since
		// the device was first deployed.
		MasterURL: t.agentGRPCURL(),
		JoinToken: t.fleetSvc.JoinToken(),
		DeployID:  deployment.ID,
	}
}

// decide updates the state for one look at a device and returns why the
// install script should run now, or "" to leave the device alone. When it
// returns a reason it has marked the state busy.
func (state *deviceState) decide(now time.Time, connected, outdated, idle bool) string {
	reason := ""
	switch {
	case state.busy:
	case connected && !outdated:
		state.offlineSince, state.nextTry, state.failures = time.Time{}, time.Time{}, 0
	case connected:
		// Replacing the agent ends its jobs, so wait until it is idle.
		if idle && !now.Before(state.nextTry) {
			reason = "its agent is not the master's build"
		}
	case state.offlineSince.IsZero():
		state.offlineSince = now
	case now.Sub(state.offlineSince) >= lifecycleGrace && !now.Before(state.nextTry):
		reason = "its agent is not connected"
	}
	if reason != "" {
		state.busy = true
	}
	return reason
}
