package controltower

import (
	"context"
	"time"

	"github.com/ss497254/gloski/internal/agentdist"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/provisioning"
	"github.com/ss497254/gloski/internal/tailnet"
)

// fleetHandlers owns device queries, device deletion, live fleet event streams,
// and master node info.
type fleetHandlers struct {
	name          string
	version       string
	startedAt     time.Time
	port          int
	fleetSvc      *fleet.Registry
	fleetEvents   *fleet.EventHub
	agentService  *agents.Service
	installer     *provisioning.Installer
	dist          *agentdist.Dist
	tailnetClient *tailnet.Client
	hasLocalAgent func() bool
	deviceTarget  func(deviceID string) (provisioning.Target, error)
	ctx           context.Context
}
