package handlers

import (
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()

// HealthHandler handles health check requests
type HealthHandler struct {
	version  string
	features map[string]bool
}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		version: "dev",
	}
}

// SetVersion sets the server version
func (h *HealthHandler) SetVersion(version string) {
	h.version = version
}

// SetFeatures sets the available features
func (h *HealthHandler) SetFeatures(features map[string]bool) {
	h.features = features
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string          `json:"status"`
	Version   string          `json:"version"`
	Uptime    int64           `json:"uptime"`
	Timestamp int64           `json:"timestamp"`
	Go        GoInfo          `json:"go"`
	Features  map[string]bool `json:"features,omitempty"`
}

// GoInfo contains Go runtime information
type GoInfo struct {
	Version    string `json:"version"`
	NumCPU     int    `json:"num_cpu"`
	Goroutines int    `json:"goroutines"`
}

// Check handles GET /api/health
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Version:   h.version,
		Uptime:    int64(time.Since(startTime).Seconds()),
		Timestamp: time.Now().Unix(),
		Go: GoInfo{
			Version:    runtime.Version(),
			NumCPU:     runtime.NumCPU(),
			Goroutines: runtime.NumGoroutine(),
		},
		Features: h.features,
	}

	Success(w, response)
}

// Ready handles GET /api/health/ready (Kubernetes readiness probe)
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// For now, always ready if we can serve the request
	Success(w, map[string]string{"status": "ready"})
}

// Live handles GET /api/health/live (Kubernetes liveness probe)
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	Success(w, map[string]string{"status": "live"})
}
