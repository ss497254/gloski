package controltower

import (
	"sync"

	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/metrics"
)

// deviceHandlers owns the dependencies shared by browser requests that operate
// on a device. The master wires one instance into the HTTP and transfer routes.
type deviceHandlers struct {
	agentService *agents.Service
	fleetSvc     *fleet.Registry
	metricStore  *metrics.Store
	touchStats   func()

	transfersMu sync.Mutex
	transfers   map[string]*transfer
}
