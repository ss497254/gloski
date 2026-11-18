// SSH deployment and device lifecycle endpoints. Remote agents are installed to /tmp and connect back over gRPC.

package master

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/deploy"
	"github.com/ss497254/gloski/internal/devices"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/tailscale"
)

type deployRequest struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	User      string `json:"user"`
	SSHPort   int    `json:"ssh_port"`
	AgentPort int    `json:"agent_port"`
}

func (s *Server) handleTailnetPeers(w http.ResponseWriter, r *http.Request) {
	if !s.tailscale.Available() {
		handlers.SuccessWithMessage(w, map[string]interface{}{
			"available": false,
			"error":     "tailscaled API is not reachable from the master",
			"peers":     []tailscale.Node{},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status, err := s.tailscale.Status(ctx)
	if err != nil {
		handlers.SuccessWithMessage(w, map[string]interface{}{
			"available": false,
			"error":     err.Error(),
			"peers":     []tailscale.Node{},
		})
		return
	}

	handlers.SuccessWithMessage(w, map[string]interface{}{
		"available":     true,
		"backend_state": status.BackendState,
		"self":          status.Self,
		"peers":         status.Peers,
	})
}

func (s *Server) handleDeployDevice(w http.ResponseWriter, r *http.Request) {
	if !s.deployer.Available() {
		handlers.BadRequest(w, fmt.Sprintf("gloski-agent binary not found at %q", s.deployer.Binary()))
		return
	}

	var request deployRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		handlers.BadRequest(w, "invalid deployment request")
		return
	}

	// Agents speak gRPC. Derive the endpoint the master advertises instead of
	// using the browser origin, which points at the HTTP dashboard.
	masterURL := s.agentGRPCURL()

	target := deploy.Target{
		Name:      strings.TrimSpace(request.Name),
		Host:      strings.TrimSpace(request.Host),
		User:      strings.TrimSpace(request.User),
		SSHPort:   request.SSHPort,
		AgentPort: request.AgentPort,
		MasterURL: masterURL,
		JoinToken: s.devices.JoinToken(),
		DeployID:  uuid.NewString(),
	}
	if err := target.Validate(); err != nil {
		handlers.BadRequest(w, err.Error())
		return
	}

	now := time.Now().UTC()
	deployment := &devices.Deployment{
		ID:        target.DeployID,
		Name:      target.Name,
		Host:      target.Host,
		User:      target.User,
		SSHPort:   target.SSHPort,
		AgentPort: target.AgentPort,
		MasterURL: target.MasterURL,
		Status:    devices.DeployPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.devices.CreateDeployment(deployment); err != nil {
		handlers.InternalError(w, "failed to record deployment", err.Error())
		return
	}

	go s.runDeployment(deployment, target)
	handlers.SuccessWithMessage(w, deployment)
}

func (s *Server) runDeployment(deployment *devices.Deployment, target deploy.Target) {
	timeout := time.Duration(s.cfg.Master.DeployTimeout) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.devices.SetDeploymentStatus(deployment.ID, devices.DeployUploading, ""); err != nil {
		logger.Warn("Failed to update deployment %s: %v", deployment.ID, err)
	}

	pid, err := s.deployer.Deploy(ctx, target)
	if err != nil {
		logger.Error("Deployment %s failed: %v", deployment.ID, err)
		if statusErr := s.devices.SetDeploymentStatus(deployment.ID, devices.DeployFailed, err.Error()); statusErr != nil {
			logger.Warn("Failed to record deployment failure: %v", statusErr)
		}
		return
	}

	if err := s.devices.SetDeploymentRunning(deployment.ID, pid); err != nil {
		logger.Warn("Failed to mark deployment %s as running: %v", deployment.ID, err)
	}
	logger.Info("Agent %s deployed to %s (pid %s)", deployment.Name, deployment.Host, pid)
}

func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	list, err := s.devices.ListDeployments()
	if err != nil {
		handlers.InternalError(w, "failed to list deployments", err.Error())
		return
	}
	if list == nil {
		list = []devices.Deployment{}
	}
	handlers.SuccessWithMessage(w, map[string]interface{}{"deployments": list})
}

func (s *Server) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	deployment, err := s.devices.GetDeployment(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, devices.ErrDeploymentNotFound) {
			handlers.NotFound(w, "deployment not found")
			return
		}
		handlers.InternalError(w, "failed to get deployment", err.Error())
		return
	}
	handlers.SuccessWithMessage(w, deployment)
}

func (s *Server) handleStartDevice(w http.ResponseWriter, r *http.Request) {
	target, err := s.deviceTarget(r.PathValue("id"))
	if err != nil {
		handlers.NotFound(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	pid, err := s.deployer.Start(ctx, target)
	if err != nil {
		_ = s.devices.SetDeploymentStatus(target.DeployID, devices.DeployFailed, err.Error())
		handlers.InternalError(w, "failed to start agent", err.Error())
		return
	}
	if err := s.devices.SetDeploymentRunning(target.DeployID, pid); err != nil {
		logger.Warn("Failed to update deployment %s: %v", target.DeployID, err)
	}
	handlers.SuccessWithMessage(w, map[string]interface{}{"pid": pid, "status": devices.DeployRunning})
}

func (s *Server) handleStopDevice(w http.ResponseWriter, r *http.Request) {
	device, err := s.devices.Get(r.PathValue("id"))
	if err != nil {
		handlers.NotFound(w, "device not found")
		return
	}
	target, err := s.deviceTarget(device.ID)
	if err != nil {
		handlers.BadRequest(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	stopped, err := s.deployer.Stop(ctx, target)
	if err != nil {
		_ = s.devices.SetDeploymentStatus(target.DeployID, devices.DeployFailed, err.Error())
		handlers.InternalError(w, "failed to stop agent", err.Error())
		return
	}
	if err := s.devices.SetDeploymentStopped(target.DeployID); err != nil {
		logger.Warn("Failed to update deployment %s: %v", target.DeployID, err)
	}
	if err := s.devices.MarkOffline(device.ID); err != nil {
		logger.Warn("Failed to mark device %s offline: %v", device.ID, err)
	}
	handlers.SuccessWithMessage(w, map[string]interface{}{"stopped": stopped, "status": devices.DeployStopped})
}

func (s *Server) deviceTarget(deviceID string) (deploy.Target, error) {
	device, err := s.devices.Get(deviceID)
	if err != nil {
		return deploy.Target{}, errors.New("device not found")
	}
	if device.DeployID == "" || device.Deploy == nil {
		return deploy.Target{}, errors.New("device was not deployed by this tower")
	}

	deployment := device.Deploy
	return deploy.Target{
		Name:       device.Name,
		Host:       deployment.Host,
		User:       deployment.User,
		SSHPort:    deployment.SSHPort,
		AgentPort:  deployment.AgentPort,
		MasterURL:  deployment.MasterURL,
		JoinToken:  s.devices.JoinToken(),
		DeployID:   deployment.ID,
		RemoteBase: deploy.DefaultRemoteBase(),
	}, nil
}
