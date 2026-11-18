package controltower

import (
	"io"
	"net/http"
	"strings"

	"github.com/ss497254/gloski/internal/api/handlers"
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

func fileRequest(path, operation string) *agentv1.FileRequest {
	switch operation {
	case "list":
		return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_LIST, Path: path}
	case "read":
		return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_READ, Path: path}
	case "write":
		return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_WRITE, Path: path}
	case "delete":
		return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_DELETE, Path: path}
	case "mkdir":
		return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_MKDIR, Path: path}
	case "rename":
		return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_RENAME, Path: path}
	default:
		return &agentv1.FileRequest{Op: agentv1.FileOp_FILE_OP_UNSPECIFIED, Path: path}
	}
}

func (t *Tower) performFile(w http.ResponseWriter, deviceID string, request *agentv1.FileRequest) *agentv1.FileResult {
	result, err := t.agentService.File(deviceID, request)
	if err != nil {
		writeAgentError(w, err)
		return nil
	}
	if result.GetCode() != 0 {
		handlers.Error(w, http.StatusBadRequest, result.GetMessage())
		return nil
	}
	return result
}

func (t *Tower) handleListFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "~"
	}
	result := t.performFile(w, r.PathValue("id"), fileRequest(path, "list"))
	if result == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(result.GetEntriesJson()))
}

func (t *Tower) handleReadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	result := t.performFile(w, r.PathValue("id"), fileRequest(path, "read"))
	if result == nil {
		return
	}
	handlers.Success(w, map[string]interface{}{"content": string(result.GetContent()), "path": path})
}

func (t *Tower) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	result := t.performFile(w, r.PathValue("id"), fileRequest(path, "read"))
	if result == nil {
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(result.GetContent())
}

func (t *Tower) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if t.performFile(w, r.PathValue("id"), fileRequest(path, "delete")) != nil {
		handlers.SuccessWithMessage(w, nil)
	}
}

func (t *Tower) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[writeFileRequest](w, r, 32<<20)
	if !ok {
		return
	}
	grpcRequest := fileRequest(request.Path, "write")
	grpcRequest.Content = []byte(request.Content)
	if t.performFile(w, r.PathValue("id"), grpcRequest) != nil {
		handlers.SuccessWithMessage(w, nil)
	}
}

func (t *Tower) handleMkdir(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[pathRequest](w, r, 1<<20)
	if !ok {
		return
	}
	if t.performFile(w, r.PathValue("id"), fileRequest(request.Path, "mkdir")) != nil {
		handlers.SuccessWithMessage(w, nil)
	}
}

func (t *Tower) handleRename(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[renameRequest](w, r, 1<<20)
	if !ok {
		return
	}
	grpcRequest := fileRequest(request.OldPath, "rename")
	grpcRequest.NewPath = request.NewPath
	if t.performFile(w, r.PathValue("id"), grpcRequest) != nil {
		handlers.SuccessWithMessage(w, nil)
	}
}

func (t *Tower) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handlers.BadRequest(w, "invalid upload")
		return
	}
	destination := r.FormValue("path")
	if destination == "" {
		destination = "~"
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		handlers.BadRequest(w, "file is required")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 32<<20))
	if err != nil {
		handlers.InternalError(w, "failed to read upload", err.Error())
		return
	}
	path := strings.TrimRight(destination, "/") + "/" + header.Filename
	grpcRequest := fileRequest(path, "write")
	grpcRequest.Content = content
	if t.performFile(w, r.PathValue("id"), grpcRequest) != nil {
		handlers.Success(w, map[string]interface{}{"filename": header.Filename})
	}
}
