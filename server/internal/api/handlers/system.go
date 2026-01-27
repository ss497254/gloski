package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ss497254/gloski/internal/system"
)

// SystemHandler handles system-related requests
type SystemHandler struct {
	systemService *system.Service
	features      map[string]bool
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(systemService *system.Service) *SystemHandler {
	return &SystemHandler{
		systemService: systemService,
	}
}

// SetFeatures sets the available features map
func (h *SystemHandler) SetFeatures(features map[string]bool) {
	h.features = features
}

// GetStats handles GET /api/system/stats
func (h *SystemHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.systemService.GetStats()
	if err != nil {
		InternalError(w, "failed to get system stats", err.Error())
		return
	}
	Success(w, stats)
}

// GetInfo handles GET /api/system/info
func (h *SystemHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	info, err := h.systemService.GetServerInfo()
	if err != nil {
		InternalError(w, "failed to get server info", err.Error())
		return
	}

	// Override features if set from app
	if h.features != nil {
		info.Features = h.features
	}

	Success(w, info)
}

// GetProcesses handles GET /api/system/processes
func (h *SystemHandler) GetProcesses(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if n := r.URL.Query().Get("limit"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	processes, err := h.systemService.GetProcesses(limit)
	if err != nil {
		InternalError(w, "failed to get processes", err.Error())
		return
	}

	Success(w, map[string]interface{}{"processes": processes})
}

// GetStatsHistory handles GET /api/system/stats/history
// Query params:
//   - duration: time duration string (e.g., "5m", "1h"). Default: "5m"
func (h *SystemHandler) GetStatsHistory(w http.ResponseWriter, r *http.Request) {
	// Parse duration from query param, default to 5 minutes
	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = "5m"
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		BadRequest(w, "invalid duration format")
		return
	}

	// Cap at reasonable maximum (1 hour)
	if duration > time.Hour {
		duration = time.Hour
	}

	samples := h.systemService.GetStatsHistory(duration)

	Success(w, map[string]interface{}{
		"samples":  samples,
		"count":    len(samples),
		"duration": duration.String(),
	})
}
