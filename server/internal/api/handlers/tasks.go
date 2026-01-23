package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/tasks"
)

// TasksHandler handles task-related requests
type TasksHandler struct {
	taskService *tasks.Service
}

// NewTasksHandler creates a new tasks handler
func NewTasksHandler(taskService *tasks.Service) *TasksHandler {
	return &TasksHandler{
		taskService: taskService,
	}
}

// List handles GET /api/tasks
func (h *TasksHandler) List(w http.ResponseWriter, r *http.Request) {
	taskList := h.taskService.List()
	Success(w, map[string]interface{}{"tasks": taskList})
}

// StartRequest represents a task start request
type StartRequest struct {
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
}

// Start handles POST /api/tasks
func (h *TasksHandler) Start(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Command == "" {
		BadRequest(w, "command is required")
		return
	}

	id := uuid.New().String()[:8]
	task, err := h.taskService.Start(id, req.Command, req.Cwd)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	logger.Info("Task started: %s (command: %s)", id, req.Command)
	Success(w, task)
}

// Get handles GET /api/tasks/{id}
func (h *TasksHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, ok := h.taskService.Get(id)
	if !ok {
		NotFound(w, "task not found")
		return
	}

	Success(w, task)
}

// GetLogs handles GET /api/tasks/{id}/logs
func (h *TasksHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	output, err := h.taskService.GetOutput(id)
	if err != nil {
		NotFound(w, err.Error())
		return
	}

	Success(w, map[string]interface{}{"logs": output})
}

// Stop handles DELETE /api/tasks/{id}
func (h *TasksHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.taskService.Stop(id); err != nil {
		BadRequest(w, err.Error())
		return
	}

	logger.Info("Task stopped: %s", id)
	SuccessWithMessage(w, nil)
}

// ListSystemd handles GET /api/systemd
func (h *TasksHandler) ListSystemd(w http.ResponseWriter, r *http.Request) {
	userMode := r.URL.Query().Get("user") == "true"
	units, err := h.taskService.ListSystemdUnits(userMode)
	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]interface{}{"units": units})
}

// SystemdAction handles POST /api/systemd/{unit}/{action}
func (h *TasksHandler) SystemdAction(w http.ResponseWriter, r *http.Request) {
	unit := r.PathValue("unit")
	action := r.PathValue("action")
	userMode := r.URL.Query().Get("user") == "true"

	validActions := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
		"enable":  true,
		"disable": true,
	}

	if !validActions[action] {
		BadRequest(w, "invalid action")
		return
	}

	if err := h.taskService.SystemdAction(unit, action, userMode); err != nil {
		InternalError(w, err.Error())
		return
	}

	logger.Info("Systemd action: %s %s", action, unit)
	SuccessWithMessage(w, nil)
}

// SystemdLogs handles GET /api/systemd/{unit}/logs
func (h *TasksHandler) SystemdLogs(w http.ResponseWriter, r *http.Request) {
	unit := r.PathValue("unit")
	userMode := r.URL.Query().Get("user") == "true"

	lines := 100
	if n := r.URL.Query().Get("lines"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 && parsed <= 1000 {
			lines = parsed
		}
	}

	logs, err := h.taskService.SystemdLogs(unit, userMode, lines)
	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]string{"logs": logs})
}
