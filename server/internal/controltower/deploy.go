package controltower

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/provisioning"
	"github.com/ss497254/gloski/internal/tailnet"
)

type deployRequest struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	User      string `json:"user"`
	SSHPort   int    `json:"ssh_port"`
	AgentPort int    `json:"agent_port"`
}

func (t *Tower) handleTailnetPeers(w http.ResponseWriter, r *http.Request) {
	if !t.tailnetClient.Available() {
		httpapi.SuccessWithMessage(w, map[string]interface{}{
			"available": false,
			"error":     "tailscaled API is not reachable from the master",
			"peers":     []tailnet.Node{},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status, err := t.tailnetClient.Status(ctx)
	if err != nil {
		httpapi.SuccessWithMessage(w, map[string]interface{}{
			"available": false,
			"error":     err.Error(),
			"peers":     []tailnet.Node{},
		})
		return
	}

	httpapi.SuccessWithMessage(w, map[string]interface{}{
		"available":     true,
		"backend_state": status.BackendState,
		"self":          status.Self,
		"peers":         status.Peers,
	})
}

func (t *Tower) handleDeployDevice(w http.ResponseWriter, r *http.Request) {
	if !t.installer.Available() {
		httpapi.BadRequest(w, "the master has no agent builds to install; run 'make build' on the master")
		return
	}

	request, ok := decodeJSON[deployRequest](w, r, 1<<20)
	if !ok {
		return
	}

	// Agents speak gRPC. Derive the endpoint the master advertises instead of
	// using the browser origin, which points at the HTTP dashboard.
	masterURL := t.agentGRPCURL()

	target := provisioning.Target{
		Name:      strings.TrimSpace(request.Name),
		Host:      strings.TrimSpace(request.Host),
		User:      strings.TrimSpace(request.User),
		SSHPort:   request.SSHPort,
		AgentPort: request.AgentPort,
		MasterURL: masterURL,
		JoinToken: t.fleetSvc.JoinToken(),
		DeployID:  uuid.NewString(),
	}
	if err := target.Validate(); err != nil {
		httpapi.BadRequest(w, err.Error())
		return
	}

	now := time.Now().UTC()
	deployment := &fleet.Deployment{
		ID:        target.DeployID,
		Name:      target.Name,
		Host:      target.Host,
		User:      target.User,
		SSHPort:   target.SSHPort,
		AgentPort: target.AgentPort,
		MasterURL: target.MasterURL,
		Status:    fleet.DeployPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := t.fleetSvc.CreateDeployment(deployment); err != nil {
		httpapi.InternalError(w, "failed to record deployment", err.Error())
		return
	}

	go func() { _, _ = t.lifecycle.Install(deployment, false) }()
	httpapi.SuccessWithMessage(w, deployment)
}

func (t *Tower) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	list, err := t.fleetSvc.ListDeployments()
	if err != nil {
		httpapi.InternalError(w, "failed to list deployments", err.Error())
		return
	}
	if list == nil {
		list = []fleet.Deployment{}
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{"deployments": list})
}

func (t *Tower) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	deployment, err := t.fleetSvc.GetDeployment(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, fleet.ErrDeploymentNotFound) {
			httpapi.NotFound(w, "deployment not found")
			return
		}
		httpapi.InternalError(w, "failed to get deployment", err.Error())
		return
	}
	httpapi.SuccessWithMessage(w, deployment)
}

func (t *Tower) handleStartDevice(w http.ResponseWriter, r *http.Request) {
	t.installNow(w, r, false)
}

// handleTrustHost accepts a device's new SSH host key, then installs again.
func (t *Tower) handleTrustHost(w http.ResponseWriter, r *http.Request) {
	t.installNow(w, r, true)
}

// installNow runs the install script on a device at a person's request. It
// starts a stopped agent, restarts a running one, and replaces a broken one.
func (t *Tower) installNow(w http.ResponseWriter, r *http.Request, trustHost bool) {
	device, err := t.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	if device.Deploy == nil {
		httpapi.BadRequest(w, "the master has no SSH details for this device")
		return
	}
	if trustHost {
		if err := t.installer.ForgetHost(device.Deploy.Host, device.Deploy.SSHPort); err != nil {
			httpapi.InternalError(w, "failed to forget the old host key", err.Error())
			return
		}
	}
	result, err := t.lifecycle.Install(device.Deploy, true)
	if err != nil {
		httpapi.InternalError(w, "failed to install the agent", err.Error())
		return
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{"pid": result.PID, "status": fleet.DeployRunning})
}

func (t *Tower) handleStopDevice(w http.ResponseWriter, r *http.Request) {
	device, err := t.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	target, targetErr := t.deviceTarget(device.ID)

	// A connected agent is simply told to exit. SSH is the way in when it is
	// not connected. Either way the deployment is marked stopped, which is
	// what keeps the lifecycle loop from starting it again.
	stopped := false
	if err := t.agentService.Shutdown(device.ID, agents.ShutdownExit); err == nil {
		stopped = true
	} else if targetErr != nil {
		httpapi.BadRequest(w, "the device is offline and was not installed over SSH, so the master cannot reach it")
		return
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		stopped, err = t.installer.Stop(ctx, target)
		if err != nil {
			_ = t.fleetSvc.SetDeploymentStatus(target.DeployID, fleet.DeployFailed, err.Error())
			httpapi.InternalError(w, "failed to stop agent", err.Error())
			return
		}
	}
	if targetErr == nil {
		if err := t.fleetSvc.SetDeploymentStopped(target.DeployID); err != nil {
			logger.Warn("Failed to update deployment %s: %v", target.DeployID, err)
		}
	}
	// The agent is gone; drop its stream now instead of waiting for TCP to notice.
	t.agentService.Disconnect(device.ID)
	httpapi.SuccessWithMessage(w, map[string]interface{}{"stopped": stopped, "status": fleet.DeployStopped})
}

func (t *Tower) deviceTarget(deviceID string) (provisioning.Target, error) {
	device, err := t.fleetSvc.Get(deviceID)
	if err != nil {
		return provisioning.Target{}, errors.New("device not found")
	}
	if device.DeployID == "" || device.Deploy == nil {
		return provisioning.Target{}, errors.New("device was not deployed by this tower")
	}

	deployment := device.Deploy
	return provisioning.Target{
		Name:      device.Name,
		Host:      deployment.Host,
		User:      deployment.User,
		SSHPort:   deployment.SSHPort,
		AgentPort: deployment.AgentPort,
		MasterURL: deployment.MasterURL,
		JoinToken: t.fleetSvc.JoinToken(),
		DeployID:  deployment.ID,
	}, nil
}
