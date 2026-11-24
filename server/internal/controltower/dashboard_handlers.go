package controltower

import (
	"context"
	"sync"
	"time"

	"github.com/ss497254/gloski/internal/agentdist"
	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/fleet"
)

// dashboardHandlers owns fleet health aggregation, dashboard issue generation,
// and master node self-testing.
type dashboardHandlers struct {
	fleetSvc     *fleet.Registry
	agentService *agents.Service
	checkStore   func() *checks.Store
	startedAt    time.Time
	touchStats   func()
	dataDir      string
	masterDBPath string
	db           *database.Database
	dist         *agentdist.Dist
	maintenance  func() (time.Time, error)
	ctx          context.Context

	samplesMu      sync.Mutex
	runtimeSamples []runtimeSample
}
