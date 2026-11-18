package controltower

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/httpapi"
)

var deviceWSUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // tailnet access is controlled by tailscale serve / ACLs
	},
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request, maxBytes int64) (T, bool) {
	var value T
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	if err := decoder.Decode(&value); err != nil {
		httpapi.BadRequest(w, "invalid request body")
		return value, false
	}
	return value, true
}

func (t *Tower) handleDeviceStats(w http.ResponseWriter, r *http.Request) {
	device, err := t.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	httpapi.Success(w, device.Stats)
}

func (t *Tower) handleDeviceStatsWS(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if _, err := t.fleetSvc.Get(deviceID); err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	connection, err := deviceWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		device, err := t.fleetSvc.Get(deviceID)
		if err != nil {
			return
		}
		if device.Stats != nil {
			raw, err := json.Marshal(device.Stats)
			if err != nil {
				return
			}
			if err := connection.WriteMessage(websocket.TextMessage, raw); err != nil {
				return
			}
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (t *Tower) handleDeviceProcesses(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	raw, err := t.agentService.Processes(r.PathValue("id"), limit, r.URL.Query().Get("sort"))
	if err != nil {
		writeAgentError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

func writeAgentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, agents.ErrNoAgent):
		httpapi.Error(w, http.StatusBadGateway, err.Error())
	case errors.Is(err, fleet.ErrDeviceNotFound):
		httpapi.NotFound(w, "device not found")
	default:
		httpapi.InternalError(w, "agent request failed", err.Error())
	}
}
