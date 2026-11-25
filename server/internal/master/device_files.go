package master

import (
	"context"
	"net/http"

	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/proto/agentv1"
)

type writeFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type pathRequest struct {
	Path string `json:"path"`
}

type renameRequest struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

func fileListRequest(path string) *agentv1.FileRequest {
	return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_LIST, Path: path}
}

func fileReadRequest(path string) *agentv1.FileRequest {
	return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_READ, Path: path}
}

func fileWriteRequest(path string, content []byte) *agentv1.FileRequest {
	return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_WRITE, Path: path, Content: content}
}

func fileDeleteRequest(path string) *agentv1.FileRequest {
	return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_DELETE, Path: path}
}

func fileMkdirRequest(path string) *agentv1.FileRequest {
	return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_MKDIR, Path: path}
}

func fileRenameRequest(oldPath, newPath string) *agentv1.FileRequest {
	return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_RENAME, Path: oldPath, NewPath: newPath}
}

func fileStatRequest(path string) *agentv1.FileRequest {
	return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_STAT, Path: path}
}

func (d *deviceHandlers) performFile(ctx context.Context, w http.ResponseWriter, deviceID string, request *agentv1.FileRequest) *agentv1.FileResult {
	result, err := d.agentService.File(ctx, deviceID, request)
	if err != nil {
		writeAgentError(w, err)
		return nil
	}
	if result.GetCode() != 0 {
		httpapi.Error(w, http.StatusBadRequest, result.GetMessage())
		return nil
	}
	return result
}

func (d *deviceHandlers) handleListFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "~"
	}
	result := d.performFile(r.Context(), w, r.PathValue("id"), fileListRequest(path))
	if result == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(result.GetEntriesJson()))
}

func (d *deviceHandlers) handleReadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	result := d.performFile(r.Context(), w, r.PathValue("id"), fileReadRequest(path))
	if result == nil {
		return
	}
	httpapi.Success(w, map[string]interface{}{"content": string(result.GetContent()), "path": path})
}

func (d *deviceHandlers) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if d.performFile(r.Context(), w, r.PathValue("id"), fileDeleteRequest(path)) != nil {
		httpapi.SuccessWithMessage(w, nil)
	}
}

func (d *deviceHandlers) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[writeFileRequest](w, r, 32<<20)
	if !ok {
		return
	}
	if d.performFile(r.Context(), w, r.PathValue("id"), fileWriteRequest(request.Path, []byte(request.Content))) != nil {
		httpapi.SuccessWithMessage(w, nil)
	}
}

func (d *deviceHandlers) handleMkdir(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[pathRequest](w, r, 1<<20)
	if !ok {
		return
	}
	if d.performFile(r.Context(), w, r.PathValue("id"), fileMkdirRequest(request.Path)) != nil {
		httpapi.SuccessWithMessage(w, nil)
	}
}

func (d *deviceHandlers) handleRename(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[renameRequest](w, r, 1<<20)
	if !ok {
		return
	}
	if d.performFile(r.Context(), w, r.PathValue("id"), fileRenameRequest(request.OldPath, request.NewPath)) != nil {
		httpapi.SuccessWithMessage(w, nil)
	}
}
