package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ss497254/gloski/internal/docker"
)

// DockerHandler handles Docker-related requests.
type DockerHandler struct {
	service *docker.Service
}

// NewDockerHandler creates a new Docker handler.
func NewDockerHandler(service *docker.Service) *DockerHandler {
	return &DockerHandler{service: service}
}

// Available returns whether Docker is available.
func (h *DockerHandler) Available() bool {
	return h.service != nil
}

// ListContainers handles GET /api/docker/containers
func (h *DockerHandler) ListContainers(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Docker not available")
		return
	}

	all := r.URL.Query().Get("all") == "true"

	containers, err := h.service.ListContainers(all)
	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]interface{}{"containers": containers})
}

// ListImages handles GET /api/docker/images
func (h *DockerHandler) ListImages(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Docker not available")
		return
	}

	images, err := h.service.ListImages()
	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]interface{}{"images": images})
}

// ContainerAction handles POST /api/docker/containers/{id}/{action}
func (h *DockerHandler) ContainerAction(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Docker not available")
		return
	}

	containerID := r.PathValue("id")
	action := r.PathValue("action")

	if containerID == "" || action == "" {
		BadRequest(w, "container ID and action required")
		return
	}

	if err := h.service.ContainerAction(containerID, action); err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "ok", "action": action})
}

// ContainerLogs handles GET /api/docker/containers/{id}/logs
func (h *DockerHandler) ContainerLogs(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Docker not available")
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		BadRequest(w, "container ID required")
		return
	}

	tail := 100
	if n := r.URL.Query().Get("tail"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 {
			tail = parsed
		}
	}

	logs, err := h.service.ContainerLogs(containerID, tail, false)
	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]string{"logs": logs})
}

// ContainerInspect handles GET /api/docker/containers/{id}/inspect
func (h *DockerHandler) ContainerInspect(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Docker not available")
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		BadRequest(w, "container ID required")
		return
	}

	info, err := h.service.ContainerInspect(containerID)
	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, info)
}

// ContainerStats handles GET /api/docker/containers/{id}/stats
func (h *DockerHandler) ContainerStats(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Docker not available")
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		BadRequest(w, "container ID required")
		return
	}

	stats, err := h.service.ContainerStats(containerID)
	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, stats)
}

// PullImage handles POST /api/docker/images/pull
func (h *DockerHandler) PullImage(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Docker not available")
		return
	}

	var req struct {
		Image string `json:"image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Image == "" {
		BadRequest(w, "image name required")
		return
	}

	if err := h.service.PullImage(req.Image); err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "ok", "image": req.Image})
}

// RemoveImage handles DELETE /api/docker/images/{id}
func (h *DockerHandler) RemoveImage(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Docker not available")
		return
	}

	imageID := r.PathValue("id")
	if imageID == "" {
		BadRequest(w, "image ID required")
		return
	}

	force := r.URL.Query().Get("force") == "true"

	if err := h.service.RemoveImage(imageID, force); err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "ok"})
}

// Info handles GET /api/docker/info
func (h *DockerHandler) Info(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Docker not available")
		return
	}

	info, err := h.service.Info()
	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, info)
}
