package master

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/devices"
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
		"mode":               "master",
		"name":               s.name,
		"version":            s.version,
		"started_at":         s.startedAt,
		"devices":            len(list),
		"online":             online,
		"heartbeat_interval": s.devices.HeartbeatInterval(),
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
	if err := s.devices.Delete(r.PathValue("id")); err != nil {
		if errors.Is(err, devices.ErrDeviceNotFound) {
			handlers.NotFound(w, "device not found")
			return
		}
		handlers.InternalError(w, "failed to remove device", err.Error())
		return
	}
	handlers.SuccessWithMessage(w, map[string]bool{"removed": true})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req devices.RegisterRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		handlers.BadRequest(w, "invalid registration request")
		return
	}

	remoteIP := remoteHost(r)
	resp, err := s.devices.Register(req, remoteIP)
	if err != nil {
		if errors.Is(err, devices.ErrInvalidJoinToken) {
			handlers.Error(w, http.StatusUnauthorized, "invalid join token")
			return
		}
		handlers.InternalError(w, "registration failed", err.Error())
		return
	}
	handlers.SuccessWithMessage(w, resp)
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		handlers.Unauthorized(w, "missing device token")
		return
	}

	var req devices.HeartbeatRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
		handlers.BadRequest(w, "invalid heartbeat request")
		return
	}

	dev, err := s.devices.Heartbeat(token, req)
	if err != nil {
		if errors.Is(err, devices.ErrDeviceNotFound) {
			handlers.Unauthorized(w, "unknown device token")
			return
		}
		handlers.InternalError(w, "heartbeat failed", err.Error())
		return
	}
	handlers.SuccessWithMessage(w, map[string]interface{}{
		"device_id":          dev.ID,
		"heartbeat_interval": s.devices.HeartbeatInterval(),
	})
}

func (s *Server) handleFleetWS(w http.ResponseWriter, r *http.Request) {
	s.hub.ServeWS(w, r, s.devices.List)
}

func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func bearerToken(r *http.Request) string {
	if token := strings.TrimSpace(r.Header.Get("X-API-Key")); token != "" {
		return token
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}
	return ""
}
