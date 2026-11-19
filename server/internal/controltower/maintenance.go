package controltower

import (
	"time"

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

// pruneJobs removes finished jobs past the retention window.
func (t *Tower) pruneJobs() {
	removed, err := t.agentService.PruneJobs(agents.JobRetention)
	if err != nil {
		logger.Warn("Failed to prune old jobs: %v", err)
		return
	}
	if removed > 0 {
		logger.Info("Deleted %d jobs finished more than %s ago", removed, agents.JobRetention)
	}
}
