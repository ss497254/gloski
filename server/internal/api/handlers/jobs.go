package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/jobs"
	"github.com/ss497254/gloski/internal/logger"
)

// JobsHandler handles job-related requests
type JobsHandler struct {
	jobService *jobs.Service
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(jobService *jobs.Service) *JobsHandler {
	return &JobsHandler{
		jobService: jobService,
	}
}

// List handles GET /api/jobs
func (h *JobsHandler) List(w http.ResponseWriter, r *http.Request) {
	jobList, err := h.jobService.List()
	if err != nil {
		InternalError(w, "failed to list jobs", err.Error())
		return
	}
	Success(w, map[string]interface{}{"jobs": jobList})
}

// StartJobRequest represents a job start request
type StartJobRequest struct {
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
}

// Start handles POST /api/jobs
func (h *JobsHandler) Start(w http.ResponseWriter, r *http.Request) {
	var req StartJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Command == "" {
		BadRequest(w, "command is required")
		return
	}

	id := uuid.New().String()
	job, err := h.jobService.Start(id, req.Command, req.Cwd)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	logger.Info("Job started: %s (command: %s)", id, req.Command)
	Success(w, job)
}

// Get handles GET /api/jobs/{id}
func (h *JobsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.jobService.Get(id)
	if err != nil {
		NotFound(w, "job not found")
		return
	}

	Success(w, job)
}

// GetLogs handles GET /api/jobs/{id}/logs
func (h *JobsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	output, err := h.jobService.GetLogs(id)
	if err != nil {
		NotFound(w, err.Error())
		return
	}

	Success(w, map[string]interface{}{"logs": output})
}

// Stop handles POST /api/jobs/{id}/stop
func (h *JobsHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.jobService.Stop(id); err != nil {
		BadRequest(w, err.Error())
		return
	}

	logger.Info("Job stopped: %s", id)
	SuccessWithMessage(w, nil)
}

// Delete handles DELETE /api/jobs/{id}
func (h *JobsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.jobService.Delete(id); err != nil {
		BadRequest(w, err.Error())
		return
	}

	logger.Info("Job deleted: %s", id)
	SuccessWithMessage(w, nil)
}
