package handlers

import (
	"net/http"
)

// HealthHandler handles health check requests (public, minimal info only)
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check handles GET /api/health (public - minimal info)
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	Success(w, map[string]string{"status": "ok"})
}

// Ready handles GET /api/health/ready (Kubernetes readiness probe - public, minimal)
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	Success(w, map[string]string{"status": "ready"})
}

// Live handles GET /api/health/live (Kubernetes liveness probe - public, minimal)
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	Success(w, map[string]string{"status": "live"})
}
