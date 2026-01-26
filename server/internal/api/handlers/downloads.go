package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/ss497254/gloski/internal/downloads"
)

type DownloadsHandler struct {
	downloadService *downloads.Service
}

func NewDownloadsHandler(downloadService *downloads.Service) *DownloadsHandler {
	return &DownloadsHandler{downloadService: downloadService}
}

// List returns all downloads
func (h *DownloadsHandler) List(w http.ResponseWriter, r *http.Request) {
	downloadList := h.downloadService.List()
	Success(w, map[string]interface{}{
		"downloads": downloadList,
	})
}

// Get returns a single download by ID
func (h *DownloadsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		BadRequest(w, "download ID is required")
		return
	}

	download, err := h.downloadService.Get(id)
	if err != nil {
		NotFound(w, "download not found")
		return
	}

	Success(w, download)
}

// Add creates a new download
func (h *DownloadsHandler) Add(w http.ResponseWriter, r *http.Request) {
	var req downloads.AddDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.URL == "" {
		BadRequest(w, "URL is required")
		return
	}

	if req.Destination == "" {
		BadRequest(w, "destination is required")
		return
	}

	download, err := h.downloadService.Add(req.URL, req.Destination, req.Filename)
	if err != nil {
		InternalError(w, err.Error())
		return
	}

	Success(w, download)
}

// Pause pauses a download
func (h *DownloadsHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		BadRequest(w, "download ID is required")
		return
	}

	if err := h.downloadService.Pause(id); err != nil {
		BadRequest(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "paused"})
}

// Resume resumes a paused download
func (h *DownloadsHandler) Resume(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		BadRequest(w, "download ID is required")
		return
	}

	if err := h.downloadService.Resume(id); err != nil {
		BadRequest(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "resumed"})
}

// Cancel cancels a download
func (h *DownloadsHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		BadRequest(w, "download ID is required")
		return
	}

	if err := h.downloadService.Cancel(id); err != nil {
		BadRequest(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "cancelled"})
}

// Retry retries a failed download
func (h *DownloadsHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		BadRequest(w, "download ID is required")
		return
	}

	if err := h.downloadService.Retry(id); err != nil {
		BadRequest(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "retrying"})
}

// Delete removes a download
func (h *DownloadsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		BadRequest(w, "download ID is required")
		return
	}

	// Check if file should be deleted
	deleteFile := r.URL.Query().Get("delete_file") == "true"

	if err := h.downloadService.Delete(id, deleteFile); err != nil {
		BadRequest(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "deleted"})
}

// DownloadFile serves the downloaded file (authenticated)
func (h *DownloadsHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		BadRequest(w, "download ID is required")
		return
	}

	download, err := h.downloadService.Get(id)
	if err != nil {
		NotFound(w, "download not found")
		return
	}

	if download.Status != downloads.StatusCompleted {
		BadRequest(w, "download is not completed")
		return
	}

	// Check if file exists
	if _, err := os.Stat(download.FilePath); os.IsNotExist(err) {
		NotFound(w, "file not found")
		return
	}

	// Set headers for download
	w.Header().Set("Content-Disposition", "attachment; filename=\""+download.Filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, download.FilePath)
}

// CreateShareLink creates a share link for a download
func (h *DownloadsHandler) CreateShareLink(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		BadRequest(w, "download ID is required")
		return
	}

	var req downloads.CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is okay, just use defaults
		req = downloads.CreateShareRequest{}
	}

	shareLink, err := h.downloadService.CreateShareLink(id, req.ExpiresIn)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	Success(w, shareLink)
}

// RevokeShareLink removes a share link
func (h *DownloadsHandler) RevokeShareLink(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	token := r.PathValue("token")

	if id == "" || token == "" {
		BadRequest(w, "download ID and token are required")
		return
	}

	if err := h.downloadService.RevokeShareLink(id, token); err != nil {
		BadRequest(w, err.Error())
		return
	}

	Success(w, map[string]string{"status": "revoked"})
}

// ShareHandler handles public share link downloads
type ShareHandler struct {
	downloadService *downloads.Service
}

func NewShareHandler(downloadService *downloads.Service) *ShareHandler {
	return &ShareHandler{downloadService: downloadService}
}

// Download serves a file via share link (public, no auth required)
func (h *ShareHandler) Download(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		BadRequest(w, "share token is required")
		return
	}

	download, err := h.downloadService.GetByShareToken(token)
	if err != nil {
		NotFound(w, "invalid or expired share link")
		return
	}

	// Check if file exists
	if _, err := os.Stat(download.FilePath); os.IsNotExist(err) {
		NotFound(w, "file not found")
		return
	}

	// Get file info for Content-Length
	fileInfo, err := os.Stat(download.FilePath)
	if err != nil {
		InternalError(w, "failed to get file info")
		return
	}

	// Set headers for download
	w.Header().Set("Content-Disposition", "attachment; filename=\""+download.Filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	http.ServeFile(w, r, download.FilePath)
}
