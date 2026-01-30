package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/internal/logger"
)

// FilesHandler handles file-related requests
type FilesHandler struct {
	fileService *files.Service
	db          *sql.DB
}

// NewFilesHandler creates a new files handler
func NewFilesHandler(fileService *files.Service) *FilesHandler {
	return &FilesHandler{
		fileService: fileService,
	}
}

// SetDB sets the database for pinned folders functionality
func (h *FilesHandler) SetDB(db *sql.DB) {
	h.db = db
}

// List handles GET /api/files
func (h *FilesHandler) List(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "~"
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

	// Normalize path to tilde notation
	normalizedPath, err := h.fileService.NormalizePath(path)
	if err != nil {
		normalizedPath = path
	}

	Success(w, map[string]string{"content": content, "path": normalizedPath})
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

// RenameRequest represents a rename/move request
type RenameRequest struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
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

// Rename handles POST /api/files/rename
func (h *FilesHandler) Rename(w http.ResponseWriter, r *http.Request) {
	var req RenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.OldPath == "" {
		BadRequest(w, "old_path is required")
		return
	}

	if req.NewPath == "" {
		BadRequest(w, "new_path is required")
		return
	}

	if err := h.fileService.Rename(req.OldPath, req.NewPath); err != nil {
		h.handleFileError(w, err)
		return
	}

	logger.Info("File renamed: %s -> %s", req.OldPath, req.NewPath)
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
		path = "~"
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

	// Use request context for cancellation
	results, err := h.fileService.SearchWithContext(r.Context(), path, query, searchContent, limit)
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
	case os.IsPermission(err):
		Forbidden(w, "permission denied: check server user has write access to this directory")
	case os.IsNotExist(err):
		BadRequest(w, "path does not exist")
	default:
		InternalError(w, "file operation failed", err.Error())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Pinned Folders
// ─────────────────────────────────────────────────────────────────────────────

// PinnedFolder represents a pinned folder.
type PinnedFolder struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// ListPinned handles GET /api/files/pinned - list all pinned folders.
func (h *FilesHandler) ListPinned(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		InternalError(w, "database not configured", "")
		return
	}

	rows, err := h.db.Query(`
		SELECT id, path, name, created_at 
		FROM pinned_folders 
		ORDER BY created_at ASC
	`)
	if err != nil {
		InternalError(w, "failed to query pinned folders", err.Error())
		return
	}
	defer rows.Close()

	folders := []PinnedFolder{}
	for rows.Next() {
		var f PinnedFolder
		if err := rows.Scan(&f.ID, &f.Path, &f.Name, &f.CreatedAt); err != nil {
			InternalError(w, "failed to scan pinned folder", err.Error())
			return
		}
		folders = append(folders, f)
	}

	homeDir, _ := os.UserHomeDir()

	Success(w, map[string]interface{}{
		"folders":  folders,
		"home_dir": homeDir,
	})
}

// CreatePinned handles POST /api/files/pinned - pin a folder.
func (h *FilesHandler) CreatePinned(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		InternalError(w, "database not configured", "")
		return
	}

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Path == "" {
		BadRequest(w, "path is required")
		return
	}

	if req.Name == "" {
		BadRequest(w, "name is required")
		return
	}

	// Check if already pinned
	var existingID string
	err := h.db.QueryRow("SELECT id FROM pinned_folders WHERE path = ?", req.Path).Scan(&existingID)
	if err == nil {
		// Already pinned, return the existing one
		Success(w, map[string]interface{}{
			"id":   existingID,
			"path": req.Path,
			"name": req.Name,
		})
		return
	}

	id := uuid.New().String()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err = h.db.Exec(`
		INSERT INTO pinned_folders (id, path, name, created_at)
		VALUES (?, ?, ?, ?)
	`, id, req.Path, req.Name, createdAt)
	if err != nil {
		InternalError(w, "failed to create pinned folder", err.Error())
		return
	}

	Success(w, map[string]interface{}{
		"id":         id,
		"path":       req.Path,
		"name":       req.Name,
		"created_at": createdAt,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Chunked Upload Handlers
// ─────────────────────────────────────────────────────────────────────────────

// InitChunkedUploadRequest represents a request to start a chunked upload
type InitChunkedUploadRequest struct {
	Filename    string `json:"filename"`
	Destination string `json:"destination"`
	TotalSize   int64  `json:"total_size"`
	ChunkSize   int64  `json:"chunk_size"`
}

// InitChunkedUpload handles POST /api/files/upload/init
func (h *FilesHandler) InitChunkedUpload(w http.ResponseWriter, r *http.Request) {
	var req InitChunkedUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Filename == "" {
		BadRequest(w, "filename is required")
		return
	}

	if req.Destination == "" {
		BadRequest(w, "destination is required")
		return
	}

	if req.TotalSize <= 0 {
		BadRequest(w, "total_size must be positive")
		return
	}

	// Default chunk size to 5MB if not specified
	if req.ChunkSize <= 0 {
		req.ChunkSize = 5 * 1024 * 1024
	}

	info, err := h.fileService.InitChunkedUpload(req.Destination, req.Filename, req.TotalSize, req.ChunkSize)
	if err != nil {
		h.handleFileError(w, err)
		return
	}

	Success(w, info)
}

// UploadChunk handles POST /api/files/upload/chunk
func (h *FilesHandler) UploadChunk(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form - limit to chunk size + overhead (10MB max per chunk)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		BadRequest(w, "failed to parse form")
		return
	}

	uploadID := r.FormValue("upload_id")
	if uploadID == "" {
		BadRequest(w, "upload_id is required")
		return
	}

	destination := r.FormValue("destination")
	if destination == "" {
		BadRequest(w, "destination is required")
		return
	}

	filename := r.FormValue("filename")
	if filename == "" {
		BadRequest(w, "filename is required")
		return
	}

	chunkIndexStr := r.FormValue("chunk_index")
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		BadRequest(w, "invalid chunk_index")
		return
	}

	file, _, err := r.FormFile("chunk")
	if err != nil {
		BadRequest(w, "chunk file is required")
		return
	}
	defer file.Close()

	if err := h.fileService.UploadChunk(destination, filename, uploadID, chunkIndex, file); err != nil {
		h.handleFileError(w, err)
		return
	}

	Success(w, map[string]interface{}{
		"chunk_index": chunkIndex,
		"status":      "uploaded",
	})
}

// CompleteChunkedUploadRequest represents a request to complete a chunked upload
type CompleteChunkedUploadRequest struct {
	UploadID    string `json:"upload_id"`
	Destination string `json:"destination"`
	Filename    string `json:"filename"`
	TotalChunks int    `json:"total_chunks"`
}

// CompleteChunkedUpload handles POST /api/files/upload/complete
func (h *FilesHandler) CompleteChunkedUpload(w http.ResponseWriter, r *http.Request) {
	var req CompleteChunkedUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.UploadID == "" {
		BadRequest(w, "upload_id is required")
		return
	}

	if req.Destination == "" {
		BadRequest(w, "destination is required")
		return
	}

	if req.Filename == "" {
		BadRequest(w, "filename is required")
		return
	}

	if req.TotalChunks <= 0 {
		BadRequest(w, "total_chunks must be positive")
		return
	}

	if err := h.fileService.CompleteChunkedUpload(req.Destination, req.Filename, req.UploadID, req.TotalChunks); err != nil {
		h.handleFileError(w, err)
		return
	}

	logger.Info("Chunked upload completed: %s/%s", req.Destination, req.Filename)
	Success(w, map[string]interface{}{
		"status":   "completed",
		"filename": req.Filename,
	})
}

// AbortChunkedUploadRequest represents a request to abort a chunked upload
type AbortChunkedUploadRequest struct {
	UploadID    string `json:"upload_id"`
	Destination string `json:"destination"`
}

// AbortChunkedUpload handles POST /api/files/upload/abort
func (h *FilesHandler) AbortChunkedUpload(w http.ResponseWriter, r *http.Request) {
	var req AbortChunkedUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.UploadID == "" {
		BadRequest(w, "upload_id is required")
		return
	}

	if req.Destination == "" {
		BadRequest(w, "destination is required")
		return
	}

	if err := h.fileService.AbortChunkedUpload(req.Destination, req.UploadID); err != nil {
		h.handleFileError(w, err)
		return
	}

	logger.Info("Chunked upload aborted: %s", req.UploadID)
	SuccessWithMessage(w, nil)
}

// DeletePinned handles DELETE /api/files/pinned/{id} - unpin a folder.
func (h *FilesHandler) DeletePinned(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		InternalError(w, "database not configured", "")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		BadRequest(w, "id is required")
		return
	}

	result, err := h.db.Exec("DELETE FROM pinned_folders WHERE id = ?", id)
	if err != nil {
		InternalError(w, "failed to delete pinned folder", err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		NotFound(w, "pinned folder not found")
		return
	}

	SuccessWithMessage(w, nil)
}
