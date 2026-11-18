package master

import (
	"errors"
	"net/http"

	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/devices"
	"github.com/ss497254/gloski/internal/logger"
)

func (s *Server) handleTowerInfo(w http.ResponseWriter, r *http.Request) {
	list, err := s.devices.List()
	if err != nil {
		handlers.InternalError(w, "failed to list devices", err.Error())
		return
	}
	online := 0
	for i := range list {
		if list[i].Status == devices.StatusOnline {
			online++
		}
	}
	handlers.SuccessWithMessage(w, map[string]interface{}{
		"mode":                "master",
		"name":                s.name,
		"version":             s.version,
		"started_at":          s.startedAt,
		"devices":             len(list),
		"online":              online,
		"heartbeat_interval":  s.devices.HeartbeatInterval(),
		"port":                s.cfg.Port,
		"deploy_available":    s.deployer.Available(),
		"agent_binary":        s.deployer.Binary(),
		"tailscale_available": s.tailscale.Available(),
		"tailscale_socket":    s.tailscale.Socket(),
	})
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	list, err := s.devices.List()
	if err != nil {
		handlers.InternalError(w, "failed to list devices", err.Error())
		return
	}
	if list == nil {
		list = []devices.Device{}
	}
	handlers.SuccessWithMessage(w, map[string]interface{}{"devices": list})
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	dev, err := s.devices.Get(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, devices.ErrDeviceNotFound) {
			handlers.NotFound(w, "device not found")
			return
		}
		handlers.InternalError(w, "failed to get device", err.Error())
		return
	}
	handlers.SuccessWithMessage(w, dev)
}

func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if device, err := s.devices.Get(deviceID); err == nil && device.DeployID != "" {
		if err := s.devices.DeleteDeployment(device.DeployID); err != nil {
			logger.Warn("Failed to delete deployment for device %s: %v", deviceID, err)
		}
	}
	if err := s.devices.Delete(deviceID); err != nil {
		if errors.Is(err, devices.ErrDeviceNotFound) {
			handlers.NotFound(w, "device not found")
			return
		}
		handlers.InternalError(w, "failed to remove device", err.Error())
		return
	}
	handlers.SuccessWithMessage(w, map[string]bool{"removed": true})
}

func (s *Server) handleFleetWS(w http.ResponseWriter, r *http.Request) {
	s.hub.ServeWS(w, r, s.devices.List)
}
