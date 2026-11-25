package master

import (
	"time"

	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/logger"
)

func (t *Tower) fleetLoop() {
	defer close(t.done)

	interval := time.Duration(t.fleetSvc.HeartbeatInterval()) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			list, err := t.fleetSvc.List()
			if err != nil {
				logger.Warn("Failed to refresh fleet status: %v", err)
				continue
			}
			t.fleetEvents.Broadcast(fleet.Event{Type: "devices", Devices: list})
		}
	}
}

// routes registers browser endpoints. Agent traffic never arrives here.
