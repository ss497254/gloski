package controltower

import (
	"errors"
	"net/http"

	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/logger"
)

func (t *Tower) handleTowerInfo(w http.ResponseWriter, r *http.Request) {
	list, err := t.fleetSvc.List()
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
		"name":                t.name,
		"version":             t.version,
		"started_at":          t.startedAt,
		"devices":             len(list),
		"online":              online,
		"heartbeat_interval":  t.fleetSvc.HeartbeatInterval(),
		"port":                t.cfg.Port,
		"deploy_available":    t.installer.Available(),
		"agent_binary":        t.installer.Binary(),
		"tailscale_available": t.tailnetClient.Available(),
		"tailscale_socket":    t.tailnetClient.Socket(),
	})
}

func (t *Tower) handleListDevices(w http.ResponseWriter, r *http.Request) {
	list, err := t.fleetSvc.List()
	if err != nil {
		httpapi.InternalError(w, "failed to list devices", err.Error())
		return
	}
	if list == nil {
		list = []fleet.Device{}
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{"devices": list})
}

func (t *Tower) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	dev, err := t.fleetSvc.Get(r.PathValue("id"))
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

func (t *Tower) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if device, err := t.fleetSvc.Get(deviceID); err == nil {
		if device.Master && t.localAgentProcess != nil {
			httpapi.BadRequest(w, "the master device cannot be removed while its local agent is enabled")
			return
		}
		// Stop the agent before forgetting it. A removed device whose agent
		// kept running would simply enrol again on its next reconnect.
		if err := t.agentService.Shutdown(deviceID); err == nil {
			t.agentService.Disconnect(deviceID)
		}
		if device.DeployID != "" {
			if err := t.fleetSvc.DeleteDeployment(device.DeployID); err != nil {
				logger.Warn("Failed to delete deployment for device %s: %v", deviceID, err)
			}
		}
		if err := t.agentService.ForgetDevice(deviceID); err != nil {
			logger.Warn("Failed to delete jobs for device %s: %v", deviceID, err)
		}
	}
	if err := t.fleetSvc.Delete(deviceID); err != nil {
		if errors.Is(err, fleet.ErrDeviceNotFound) {
			httpapi.NotFound(w, "device not found")
			return
		}
		httpapi.InternalError(w, "failed to remove device", err.Error())
		return
	}
	httpapi.SuccessWithMessage(w, map[string]bool{"removed": true})
}

func (t *Tower) handleFleetWS(w http.ResponseWriter, r *http.Request) {
	t.fleetEvents.ServeWS(w, r, t.fleetSvc.List)
}
