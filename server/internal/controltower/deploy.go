package controltower

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/actions"
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

func (d *deployHandlers) handleTailnetPeers(w http.ResponseWriter, r *http.Request) {
	if !d.tailnetClient.Available() {
		httpapi.SuccessWithMessage(w, map[string]interface{}{
			"available": false,
			"error":     "tailscaled API is not reachable from the master",
			"peers":     []tailnet.Node{},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status, err := d.tailnetClient.Status(ctx)
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

func (d *deployHandlers) handleDeployDevice(w http.ResponseWriter, r *http.Request) {
	if !d.installer.Available() {
		httpapi.BadRequest(w, "the master has no agent builds to install; run 'make build' on the master")
		return
	}

	request, ok := decodeJSON[deployRequest](w, r, 1<<20)
	if !ok {
		return
	}

	// Agents speak gRPC. Derive the endpoint the master advertises instead of
	// using the browser origin, which points at the HTTP dashboard.
	masterURL := d.agentGRPCURL()

	target := provisioning.Target{
		Name:      strings.TrimSpace(request.Name),
		Host:      strings.TrimSpace(request.Host),
		User:      strings.TrimSpace(request.User),
		SSHPort:   request.SSHPort,
		AgentPort: request.AgentPort,
		MasterURL: masterURL,
		JoinToken: d.fleetSvc.JoinToken(),
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
	if err := d.fleetSvc.CreateDeployment(deployment); err != nil {
		httpapi.InternalError(w, "failed to record deployment", err.Error())
		return
	}

	go func() { _, _ = d.lifecycle.Install(deployment, false) }()
	httpapi.SuccessWithMessage(w, deployment)
}

func (d *deployHandlers) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	list, err := d.fleetSvc.ListDeployments()
	if err != nil {
		httpapi.InternalError(w, "failed to list deployments", err.Error())
		return
	}
	if list == nil {
		list = []fleet.Deployment{}
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{"deployments": list})
}

func (d *deployHandlers) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	deployment, err := d.fleetSvc.GetDeployment(r.PathValue("id"))
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

var errStopUnreachable = errors.New("the device is offline and was not installed over SSH, so the master cannot reach it")

func (d *deployHandlers) handleStartDevice(w http.ResponseWriter, r *http.Request) {
	d.installNow(w, r, false)
}

// handleTrustHost accepts a device's new SSH host key, then installs again.
func (d *deployHandlers) handleTrustHost(w http.ResponseWriter, r *http.Request) {
	d.installNow(w, r, true)
}

// installNow runs the install script on a device at a person's request. It
// starts a stopped agent, restarts a running one, and replaces a broken one.
func (d *deployHandlers) installNow(w http.ResponseWriter, r *http.Request, trustHost bool) {
	device, err := d.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	if device.Deploy == nil {
		httpapi.BadRequest(w, "the master has no SSH details for this device")
		return
	}

	from := device.Deploy.Status
	kind := actions.KindAgentRestart
	if from == fleet.DeployStopped {
		kind = actions.KindAgentStart
	}
	if trustHost {
		kind = actions.KindTrustHost
	}
	result, err := d.installDevice(r.Context(), device, trustHost)
	d.recordAction(device.ID, actions.Action{
		Kind: kind, Target: "agent", From: from, To: fleet.DeployRunning, Undoable: !trustHost,
	}, err)
	if err != nil {
		httpapi.InternalError(w, "failed to install the agent", err.Error())
		return
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{"pid": result.PID, "status": fleet.DeployRunning})
}

// installDevice does the work for start, restart, trust-host and undo. The
// caller has already checked that the device has SSH details.
func (d *deployHandlers) installDevice(ctx context.Context, device *fleet.Device, trustHost bool) (provisioning.Result, error) {
	if device.Deploy == nil {
		return provisioning.Result{}, errors.New("the master has no SSH details for this device")
	}
	if trustHost {
		if err := d.installer.ForgetHost(device.Deploy.Host, device.Deploy.SSHPort); err != nil {
			return provisioning.Result{}, fmt.Errorf("failed to forget the old host key: %w", err)
		}
	}
	return d.lifecycle.Install(device.Deploy, true)
}

func (d *deployHandlers) handleStopDevice(w http.ResponseWriter, r *http.Request) {
	device, err := d.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	from := ""
	if device.Deploy != nil {
		from = device.Deploy.Status
	}
	stopped, err := d.stopDevice(r.Context(), device)
	d.recordAction(device.ID, actions.Action{
		Kind: actions.KindAgentStop, Target: "agent", From: from, To: fleet.DeployStopped, Undoable: from != "",
	}, err)
	if err != nil {
		if errors.Is(err, errStopUnreachable) {
			httpapi.BadRequest(w, err.Error())
		} else {
			httpapi.InternalError(w, "failed to stop agent", err.Error())
		}
		return
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{"stopped": stopped, "status": fleet.DeployStopped})
}

// stopDevice is shared by the stop handler and undo. A connected agent is told
// to exit; SSH is the way in when it is not. Marking the deployment stopped is
// what keeps the lifecycle loop from starting it again.
func (d *deployHandlers) stopDevice(ctx context.Context, device *fleet.Device) (bool, error) {
	target, targetErr := d.deviceTarget(device.ID)
	stopped := false
	if err := d.agentService.Shutdown(device.ID, agents.ShutdownExit); err == nil {
		stopped = true
	} else if targetErr != nil {
		return false, errStopUnreachable
	} else {
		stopCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		var err error
		stopped, err = d.installer.Stop(stopCtx, target)
		if err != nil {
			_ = d.fleetSvc.SetDeploymentStatus(target.DeployID, fleet.DeployFailed, err.Error())
			return false, fmt.Errorf("failed to stop agent: %w", err)
		}
	}
	if targetErr == nil {
		if err := d.fleetSvc.SetDeploymentStopped(target.DeployID); err != nil {
			logger.Warn("Failed to update deployment %s: %v", target.DeployID, err)
		}
	}
	// The agent is gone; drop its stream now instead of waiting for TCP to notice.
	d.agentService.Disconnect(device.ID)
	return stopped, nil
}

// recordAction never fails the user's action: the change already happened,
// and a full database must not turn a successful start into an error.
func (d *deployHandlers) recordAction(deviceID string, action actions.Action, err error) {
	action.DeviceID = deviceID
	action.Result = actions.ResultOK
	if err != nil {
		action.Result = actions.ResultFailed
		action.Error = err.Error()
	}
	if recordErr := d.actionStore.Record(d.ctx, action); recordErr != nil {
		logger.Warn("Failed to record action on %s: %v", deviceID, recordErr)
	}
}

func (d *deployHandlers) deviceTarget(deviceID string) (provisioning.Target, error) {
	device, err := d.fleetSvc.Get(deviceID)
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
		JoinToken: d.fleetSvc.JoinToken(),
		DeployID:  deployment.ID,
	}, nil
}

func (d *deployHandlers) deploymentTarget(deployment *fleet.Deployment) provisioning.Target {
	return provisioning.Target{
		Name:      deployment.Name,
		Host:      deployment.Host,
		User:      deployment.User,
		SSHPort:   deployment.SSHPort,
		AgentPort: deployment.AgentPort,
		MasterURL: d.agentGRPCURL(),
		JoinToken: d.fleetSvc.JoinToken(),
		DeployID:  deployment.ID,
	}
}

func (t *Tower) deviceTarget(deviceID string) (provisioning.Target, error) {
	return t.deployAPI.deviceTarget(deviceID)
}

func (t *Tower) installDevice(ctx context.Context, device *fleet.Device, trustHost bool) (provisioning.Result, error) {
	return t.deployAPI.installDevice(ctx, device, trustHost)
}

func (t *Tower) stopDevice(ctx context.Context, device *fleet.Device) (bool, error) {
	return t.deployAPI.stopDevice(ctx, device)
}

func (t *Tower) recordAction(deviceID string, action actions.Action, err error) {
	t.deployAPI.recordAction(deviceID, action, err)
}

func (t *Tower) diagnoseSSH(ctx context.Context, deployment *fleet.Deployment, result *diagnosis) {
	t.deployAPI.diagnoseSSH(ctx, deployment, result)
}

func (t *Tower) diagnoseDevice(ctx context.Context, device *fleet.Device) diagnosis {
	return t.deployAPI.diagnoseDevice(ctx, device)
}
