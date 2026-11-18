package controltower

import (
	"errors"
	"net/http"

	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/logger"
)

func (t *Tower) handleTowerInfo(w http.ResponseWriter, r *http.Request) {
	list, err := t.fleetSvc.List()
	if err != nil {
		handlers.InternalError(w, "failed to list devices", err.Error())
		return
	}
	online := 0
	for i := range list {
		if list[i].Status == fleet.StatusOnline {
			online++
		}
	}
	handlers.SuccessWithMessage(w, map[string]interface{}{
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
		handlers.InternalError(w, "failed to list devices", err.Error())
		return
	}
	if list == nil {
		list = []fleet.Device{}
	}
	handlers.SuccessWithMessage(w, map[string]interface{}{"devices": list})
}

func (t *Tower) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	dev, err := t.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, fleet.ErrDeviceNotFound) {
			handlers.NotFound(w, "device not found")
			return
		}
		handlers.InternalError(w, "failed to get device", err.Error())
		return
	}
	handlers.SuccessWithMessage(w, dev)
}

func (t *Tower) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if device, err := t.fleetSvc.Get(deviceID); err == nil && device.DeployID != "" {
		if err := t.fleetSvc.DeleteDeployment(device.DeployID); err != nil {
			logger.Warn("Failed to delete deployment for device %s: %v", deviceID, err)
		}
	}
	if err := t.fleetSvc.Delete(deviceID); err != nil {
		if errors.Is(err, fleet.ErrDeviceNotFound) {
			handlers.NotFound(w, "device not found")
			return
		}
		handlers.InternalError(w, "failed to remove device", err.Error())
		return
	}
	handlers.SuccessWithMessage(w, map[string]bool{"removed": true})
}

func (t *Tower) handleFleetWS(w http.ResponseWriter, r *http.Request) {
	t.fleetEvents.ServeWS(w, r, t.fleetSvc.List)
}
