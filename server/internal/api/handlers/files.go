package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/internal/logger"
)

// FilesHandler handles file-related requests
type FilesHandler struct {
	fileService *files.Service
}

// NewFilesHandler creates a new files handler
func NewFilesHandler(fileService *files.Service) *FilesHandler {
	return &FilesHandler{
		fileService: fileService,
	}
}

// List handles GET /api/files
func (h *FilesHandler) List(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	entries, err := h.fileService.List(path)
	if err != nil {
		h.handleFileError(w, err)
		return
	}

	Success(w, entries)
}

// Read handles GET /api/files/read
func (h *FilesHandler) Read(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		BadRequest(w, "path is required")
		return
	}

	content, err := h.fileService.Read(path)
	if err != nil {
		h.handleFileError(w, err)
		return
	}

	Success(w, map[string]string{"content": content, "path": path})
}

// WriteRequest represents a file write request
type WriteRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Write handles POST /api/files/write
func (h *FilesHandler) Write(w http.ResponseWriter, r *http.Request) {
	var req WriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Path == "" {
		BadRequest(w, "path is required")
		return
	}

	if err := h.fileService.Write(req.Path, req.Content); err != nil {
		h.handleFileError(w, err)
		return
	}

	SuccessWithMessage(w, nil)
}

// MkdirRequest represents a mkdir request
type MkdirRequest struct {
	Path string `json:"path"`
}

// Mkdir handles POST /api/files/mkdir
func (h *FilesHandler) Mkdir(w http.ResponseWriter, r *http.Request) {
	var req MkdirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Path == "" {
		BadRequest(w, "path is required")
		return
	}

	if err := h.fileService.Mkdir(req.Path); err != nil {
		h.handleFileError(w, err)
		return
	}

	SuccessWithMessage(w, nil)
}

// Delete handles DELETE /api/files
func (h *FilesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		BadRequest(w, "path is required")
		return
	}

	if err := h.fileService.Delete(path); err != nil {
		h.handleFileError(w, err)
		return
	}

	SuccessWithMessage(w, nil)
}

// Upload handles POST /api/files/upload
func (h *FilesHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Max 100MB
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		BadRequest(w, "failed to parse form")
		return
	}

	destPath := r.FormValue("path")
	if destPath == "" {
		BadRequest(w, "path is required")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	if err := h.fileService.Upload(destPath, handler.Filename, file); err != nil {
		h.handleFileError(w, err)
		return
	}

	logger.Info("File uploaded: %s/%s", destPath, handler.Filename)
	SuccessWithMessage(w, map[string]string{"filename": handler.Filename})
}

// Download handles GET /api/files/download
func (h *FilesHandler) Download(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		BadRequest(w, "path is required")
		return
	}

	filePath, info, err := h.fileService.GetFileInfo(path)
	if err != nil {
		h.handleFileError(w, err)
		return
	}

	if info.IsDir() {
		BadRequest(w, "cannot download a directory")
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+info.Name()+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))

	http.ServeFile(w, r, filePath)
}

// Search handles GET /api/search
func (h *FilesHandler) Search(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		BadRequest(w, "query (q) is required")
		return
	}

	searchContent := r.URL.Query().Get("content") == "true"

	limit := 100
	if n := r.URL.Query().Get("limit"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	results, err := h.fileService.Search(path, query, searchContent, limit)
	if err != nil {
		h.handleFileError(w, err)
		return
	}

	Success(w, map[string]interface{}{"results": results, "count": len(results)})
}

// handleFileError converts file service errors to HTTP responses
func (h *FilesHandler) handleFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, files.ErrPathNotAllowed):
		Forbidden(w, err.Error())
	case errors.Is(err, files.ErrDangerousPath):
		Forbidden(w, err.Error())
	case errors.Is(err, files.ErrFileTooLarge):
		BadRequest(w, err.Error())
	case errors.Is(err, files.ErrBinaryFile):
		BadRequest(w, err.Error())
	default:
		InternalError(w, err.Error())
	}
}
