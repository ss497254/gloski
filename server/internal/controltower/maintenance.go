package controltower

import (
	"context"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/logger"
)

const (
	maintenanceDelay = time.Minute
	maintenanceEvery = 24 * time.Hour
)

// maintenanceLoop keeps the master's database small. It runs shortly after
// start, so a master that is restarted often still cleans up, then daily.
func (t *Tower) maintenanceLoop() {
	timer := time.NewTimer(maintenanceDelay)
	defer timer.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-timer.C:
			t.pruneJobs()
			timer.Reset(maintenanceEvery)
		}
	}
}

// pruneJobs removes finished jobs past the retention window, thins check
// history and metric samples. The result is recorded for the self-test, which
// must not repeat the work or report a guess about the last run.
func (t *Tower) pruneJobs() {
	err := func() error {
		removed, err := t.agentService.PruneJobs(agents.JobRetention)
		if err != nil {
			return err
		}
		if removed > 0 {
			logger.Info("Deleted %d jobs finished more than %s ago", removed, agents.JobRetention)
		}
		pruned, err := t.checkStore.Prune(context.Background(), checks.HistoryLimit)
		if err != nil {
			return err
		}
		if pruned > 0 {
			logger.Info("Deleted %d old check results", pruned)
		}
		thinned, err := t.metricStore.Thin(context.Background(), time.Now())
		if err != nil {
			return err
		}
		if thinned > 0 {
			logger.Info("Deleted %d old metric samples", thinned)
		}
		return nil
	}()

	t.maintenanceMu.Lock()
	t.lastMaintenance = time.Now().UTC()
	t.lastMaintenanceErr = err
	t.maintenanceMu.Unlock()

	if err != nil {
		logger.Warn("Maintenance failed: %v", err)
	}
}

func (t *Tower) maintenanceState() (time.Time, error) {
	t.maintenanceMu.Lock()
	defer t.maintenanceMu.Unlock()
	return t.lastMaintenance, t.lastMaintenanceErr
}
