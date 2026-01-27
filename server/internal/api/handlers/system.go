package handlers

import (
	"database/sql"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/ss497254/gloski/internal/system"
)

var startTime = time.Now()

// SystemHandler handles system-related requests
type SystemHandler struct {
	systemService *system.Service
	features      map[string]bool
	version       string
	db            *sql.DB
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(systemService *system.Service) *SystemHandler {
	return &SystemHandler{
		systemService: systemService,
		version:       "dev",
	}
}

// SetFeatures sets the available features map
func (h *SystemHandler) SetFeatures(features map[string]bool) {
	h.features = features
}

// SetVersion sets the server version
func (h *SystemHandler) SetVersion(version string) {
	h.version = version
}

// SetDB sets the database connection for status checks
func (h *SystemHandler) SetDB(db *sql.DB) {
	h.db = db
}

// ─────────────────────────────────────────────────────────────────────────────
// Server Status (authenticated - detailed info)
// ─────────────────────────────────────────────────────────────────────────────

// ServerStatusResponse represents detailed server status (for authenticated users)
type ServerStatusResponse struct {
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	Uptime    int64                  `json:"uptime"`
	Timestamp int64                  `json:"timestamp"`
	Go        GoInfo                 `json:"go"`
	Features  map[string]bool        `json:"features,omitempty"`
	Checks    map[string]CheckResult `json:"checks"`
}

// GoInfo contains Go runtime information
type GoInfo struct {
	Version    string `json:"version"`
	NumCPU     int    `json:"num_cpu"`
	Goroutines int    `json:"goroutines"`
	MemoryMB   uint64 `json:"memory_mb"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Status handles GET /api/system/status (authenticated - detailed server status)
func (h *SystemHandler) Status(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]CheckResult)
	allHealthy := true

	// Check database connectivity
	if h.db != nil {
		if err := h.db.PingContext(r.Context()); err != nil {
			checks["database"] = CheckResult{Status: "unhealthy", Message: err.Error()}
			allHealthy = false
		} else {
			checks["database"] = CheckResult{Status: "healthy"}
		}
	} else {
		checks["database"] = CheckResult{Status: "unavailable", Message: "no database configured"}
	}

	// Memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	allocMB := memStats.Alloc / 1024 / 1024
	if allocMB > 500 {
		checks["memory"] = CheckResult{Status: "warning", Message: "high memory usage"}
	} else {
		checks["memory"] = CheckResult{Status: "healthy"}
	}

	// Goroutine count
	goroutines := runtime.NumGoroutine()
	if goroutines > 1000 {
		checks["goroutines"] = CheckResult{Status: "warning", Message: "high goroutine count"}
	} else {
		checks["goroutines"] = CheckResult{Status: "healthy"}
	}

	status := "healthy"
	if !allHealthy {
		status = "degraded"
	}

	response := ServerStatusResponse{
		Status:    status,
		Version:   h.version,
		Uptime:    int64(time.Since(startTime).Seconds()),
		Timestamp: time.Now().Unix(),
		Go: GoInfo{
			Version:    runtime.Version(),
			NumCPU:     runtime.NumCPU(),
			Goroutines: goroutines,
			MemoryMB:   allocMB,
		},
		Features: h.features,
		Checks:   checks,
	}

	Success(w, response)
}

// ─────────────────────────────────────────────────────────────────────────────
// System Stats & Info
// ─────────────────────────────────────────────────────────────────────────────

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
