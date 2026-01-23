package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ss497254/gloski/internal/cron"
)

// CronHandler handles cron job requests.
type CronHandler struct {
	service *cron.Service
}

// NewCronHandler creates a new cron handler.
func NewCronHandler(service *cron.Service) *CronHandler {
	return &CronHandler{service: service}
}

// Available returns whether cron is available.
func (h *CronHandler) Available() bool {
	return h.service != nil
}

// ListJobs handles GET /api/cron/jobs
func (h *CronHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Cron not available")
		return
	}

	scope := r.URL.Query().Get("scope")

	var jobs []cron.CronJob
	var err error

	switch scope {
	case "user":
		jobs, err = h.service.ListUserJobs()
	case "system":
		jobs, err = h.service.ListSystemJobs()
	default:
		jobs, err = h.service.ListAllJobs()
	}

	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// AddJob handles POST /api/cron/jobs
func (h *CronHandler) AddJob(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Cron not available")
		return
	}

	var req struct {
		Schedule string `json:"schedule"`
		Command  string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Schedule == "" || req.Command == "" {
		BadRequest(w, "schedule and command are required")
		return
	}

	// Validate schedule
	if _, err := cron.ParseSchedule(req.Schedule); err != nil {
		BadRequest(w, "invalid schedule format: "+err.Error())
		return
	}

	if err := h.service.AddJob(req.Schedule, req.Command); err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "ok"})
}

// RemoveJob handles DELETE /api/cron/jobs
func (h *CronHandler) RemoveJob(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Cron not available")
		return
	}

	var req struct {
		Schedule string `json:"schedule"`
		Command  string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Schedule == "" || req.Command == "" {
		BadRequest(w, "schedule and command are required")
		return
	}

	if err := h.service.RemoveJob(req.Schedule, req.Command); err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "ok"})
}
