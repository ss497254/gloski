package controltower

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ss497254/gloski/internal/actions"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/jobs"
)

var deviceWSUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // origin is enforced for every route by httpapi.SameOrigin
	},
}

// decodeJSON reads a JSON request body. It insists on a JSON content type so a
// plain HTML form on another site cannot be used to submit commands.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request, maxBytes int64) (T, bool) {
	var value T
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/json" {
		httpapi.Error(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return value, false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	if err := decoder.Decode(&value); err != nil {
		httpapi.BadRequest(w, "invalid request body")
		return value, false
	}
	return value, true
}

func (d *deviceHandlers) handleDeviceStats(w http.ResponseWriter, r *http.Request) {
	d.touchStats()
	device, err := d.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	httpapi.Success(w, device.Stats)
}

func (d *deviceHandlers) handleDeviceStatsWS(w http.ResponseWriter, r *http.Request) {
	d.touchStats()
	deviceID := r.PathValue("id")
	if _, err := d.fleetSvc.Get(deviceID); err != nil {
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
		d.touchStats()
		device, err := d.fleetSvc.Get(deviceID)
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

func (d *deviceHandlers) handleDeviceProcesses(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	raw, err := d.agentService.Processes(r.Context(), r.PathValue("id"), limit, r.URL.Query().Get("sort"))
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
	case errors.Is(err, agents.ErrRequestTimeout), errors.Is(err, context.DeadlineExceeded):
		httpapi.Error(w, http.StatusGatewayTimeout, err.Error())
	case errors.Is(err, fleet.ErrDeviceNotFound),
		errors.Is(err, fleet.ErrDeploymentNotFound),
		errors.Is(err, jobs.ErrJobNotFound),
		errors.Is(err, actions.ErrActionNotFound):
		httpapi.NotFound(w, err.Error())
	case errors.Is(err, errStopUnreachable), errors.Is(err, errNotDeployed):
		httpapi.BadRequest(w, err.Error())
	default:
		httpapi.InternalError(w, "agent request failed", err.Error())
	}
}
