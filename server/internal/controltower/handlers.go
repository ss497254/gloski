package controltower

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/logger"
)

func (f *fleetHandlers) handleTowerInfo(w http.ResponseWriter, r *http.Request) {
	list, err := f.fleetSvc.List()
	if err != nil {
		httpapi.InternalError(w, "failed to list devices", err.Error())
		return
	}
	online := 0
	for i := range list {
		if list[i].Status == fleet.StatusOnline {
			online++
		}
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{
		"mode":                "master",
		"name":                f.name,
		"version":             f.version,
		"started_at":          f.startedAt,
		"devices":             len(list),
		"online":              online,
		"heartbeat_interval":  f.fleetSvc.HeartbeatInterval(),
		"port":                f.port,
		"deploy_available":    f.installer.Available(),
		"agent_builds":        f.dist.Builds(),
		"tailscale_available": f.tailnetClient.Available(),
		"tailscale_socket":    f.tailnetClient.Socket(),
	})
}

func (f *fleetHandlers) handleListDevices(w http.ResponseWriter, r *http.Request) {
	list, err := f.fleetSvc.List()
	if err != nil {
		httpapi.InternalError(w, "failed to list devices", err.Error())
		return
	}
	if list == nil {
		list = []fleet.Device{}
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{"devices": list})
}

func (f *fleetHandlers) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	dev, err := f.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, fleet.ErrDeviceNotFound) {
			httpapi.NotFound(w, "device not found")
			return
		}
		httpapi.InternalError(w, "failed to get device", err.Error())
		return
	}
	httpapi.SuccessWithMessage(w, dev)
}

func (f *fleetHandlers) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if device, err := f.fleetSvc.Get(deviceID); err == nil {
		if device.Master && f.hasLocalAgent != nil && f.hasLocalAgent() {
			httpapi.BadRequest(w, "the master device cannot be removed while its local agent is enabled")
			return
		}
		// Stop the agent before forgetting it. A removed device whose agent
		// kept running would simply enrol again on its next reconnect.
		if err := f.agentService.Shutdown(deviceID, agents.ShutdownUninstall); err == nil {
			f.agentService.Disconnect(deviceID)
		} else if target, err := f.deviceTarget(deviceID); err == nil {
			// Offline, but installed over SSH: clean it up from this side.
			go func() {
				ctx, cancel := context.WithTimeout(f.ctx, 60*time.Second)
				defer cancel()
				if _, err := f.installer.Remove(ctx, target); err != nil {
					logger.Warn("Could not uninstall the agent on %s over SSH: %v", target.Host, err)
				}
			}()
		}
		if device.DeployID != "" {
			if err := f.fleetSvc.DeleteDeployment(device.DeployID); err != nil {
				logger.Warn("Failed to delete deployment for device %s: %v", deviceID, err)
			}
		}
		if err := f.agentService.ForgetDevice(deviceID); err != nil {
			logger.Warn("Failed to delete jobs for device %s: %v", deviceID, err)
		}
	}
	if err := f.fleetSvc.Delete(deviceID); err != nil {
		if errors.Is(err, fleet.ErrDeviceNotFound) {
			httpapi.NotFound(w, "device not found")
			return
		}
		httpapi.InternalError(w, "failed to remove device", err.Error())
		return
	}
	httpapi.SuccessWithMessage(w, map[string]bool{"removed": true})
}

func (f *fleetHandlers) handleFleetWS(w http.ResponseWriter, r *http.Request) {
	f.fleetEvents.ServeWS(w, r, f.fleetSvc.List)
}
