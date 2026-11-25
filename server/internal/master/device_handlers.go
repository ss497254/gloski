package master

import (
	"sync"

	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/master/agentstream"
	"github.com/ss497254/gloski/internal/metrics"
)

// deviceHandlers owns the dependencies shared by browser requests that operate
// on a device. The master wires one instance into the HTTP and transfer routes.
type deviceHandlers struct {
	agentService *agentstream.Service
	fleetSvc     *fleet.Registry
	metricStore  *metrics.Store
	touchStats   func()

	transfersMu sync.Mutex
	transfers   map[string]*transfer
}
