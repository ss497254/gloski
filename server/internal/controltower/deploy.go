package controltower

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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
		httpapi.BadRequest(w, fmt.Sprintf("gloski-agent binary not found at %q", t.installer.Binary()))
		return
	}

	var request deployRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		httpapi.BadRequest(w, "invalid deployment request")
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

	go t.runDeployment(deployment, target)
	httpapi.SuccessWithMessage(w, deployment)
}

func (t *Tower) runDeployment(deployment *fleet.Deployment, target provisioning.Target) {
	timeout := time.Duration(t.cfg.Master.DeployTimeout) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := t.fleetSvc.SetDeploymentStatus(deployment.ID, fleet.DeployUploading, ""); err != nil {
		logger.Warn("Failed to update deployment %s: %v", deployment.ID, err)
	}

	pid, err := t.installer.Deploy(ctx, target)
	if err != nil {
		logger.Error("Deployment %s failed: %v", deployment.ID, err)
		if statusErr := t.fleetSvc.SetDeploymentStatus(deployment.ID, fleet.DeployFailed, err.Error()); statusErr != nil {
			logger.Warn("Failed to record deployment failure: %v", statusErr)
		}
		return
	}

	if err := t.fleetSvc.SetDeploymentRunning(deployment.ID, pid); err != nil {
		logger.Warn("Failed to mark deployment %s as running: %v", deployment.ID, err)
	}
	logger.Info("Agent %s deployed to %s (pid %s)", deployment.Name, deployment.Host, pid)
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
	target, err := t.deviceTarget(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	pid, err := t.installer.Start(ctx, target)
	if err != nil {
		_ = t.fleetSvc.SetDeploymentStatus(target.DeployID, fleet.DeployFailed, err.Error())
		httpapi.InternalError(w, "failed to start agent", err.Error())
		return
	}
	if err := t.fleetSvc.SetDeploymentRunning(target.DeployID, pid); err != nil {
		logger.Warn("Failed to update deployment %s: %v", target.DeployID, err)
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{"pid": pid, "status": fleet.DeployRunning})
}

func (t *Tower) handleStopDevice(w http.ResponseWriter, r *http.Request) {
	device, err := t.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	target, err := t.deviceTarget(device.ID)
	if err != nil {
		httpapi.BadRequest(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	stopped, err := t.installer.Stop(ctx, target)
	if err != nil {
		_ = t.fleetSvc.SetDeploymentStatus(target.DeployID, fleet.DeployFailed, err.Error())
		httpapi.InternalError(w, "failed to stop agent", err.Error())
		return
	}
	if err := t.fleetSvc.SetDeploymentStopped(target.DeployID); err != nil {
		logger.Warn("Failed to update deployment %s: %v", target.DeployID, err)
	}
	if err := t.fleetSvc.MarkOffline(device.ID); err != nil {
		logger.Warn("Failed to mark device %s offline: %v", device.ID, err)
	}
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
		Name:       device.Name,
		Host:       deployment.Host,
		User:       deployment.User,
		SSHPort:    deployment.SSHPort,
		AgentPort:  deployment.AgentPort,
		MasterURL:  deployment.MasterURL,
		JoinToken:  t.fleetSvc.JoinToken(),
		DeployID:   deployment.ID,
		RemoteBase: provisioning.DefaultRemoteBase(),
	}, nil
}
