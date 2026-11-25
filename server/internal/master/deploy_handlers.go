package master

import (
	"context"

	"github.com/ss497254/gloski/internal/actions"
	"github.com/ss497254/gloski/internal/agentdist"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/master/agentstream"
	"github.com/ss497254/gloski/internal/provisioning"
	"github.com/ss497254/gloski/internal/tailnet"
)

// deployHandlers owns the dependencies for deployment, remote agent lifecycle,
// diagnostics, tailnet peer discovery, and device action history/undo.
type deployHandlers struct {
	fleetSvc      *fleet.Registry
	installer     *provisioning.Installer
	lifecycle     *lifecycle
	actionStore   *actions.Store
	tailnetClient *tailnet.Client
	agentService  *agentstream.Service
	dist          *agentdist.Dist
	agentBinary   string
	agentGRPCURL  func() string
	ctx           context.Context
}
