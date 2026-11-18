package controltower

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

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
		httpapi.Error(w, http.StatusBadRequest, result.GetMessage())
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
	httpapi.Success(w, map[string]interface{}{"content": string(result.GetContent()), "path": path})
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
		httpapi.SuccessWithMessage(w, nil)
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
		httpapi.SuccessWithMessage(w, nil)
	}
}

func (t *Tower) handleMkdir(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[pathRequest](w, r, 1<<20)
	if !ok {
		return
	}
	if t.performFile(w, r.PathValue("id"), fileRequest(request.Path, "mkdir")) != nil {
		httpapi.SuccessWithMessage(w, nil)
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
		httpapi.SuccessWithMessage(w, nil)
	}
}

func (t *Tower) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpapi.BadRequest(w, "invalid upload")
		return
	}
	destination := r.FormValue("path")
	if destination == "" {
		destination = "~"
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpapi.BadRequest(w, "file is required")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 32<<20))
	if err != nil {
		httpapi.InternalError(w, "failed to read upload", err.Error())
		return
	}
	// Browsers send a bare name, but the field is client-controlled: keep only
	// the final element so it cannot climb out of the destination.
	name := filepath.Base(filepath.FromSlash(header.Filename))
	if name == "." || name == ".." || name == string(filepath.Separator) {
		httpapi.BadRequest(w, "invalid file name")
		return
	}
	path := strings.TrimRight(destination, "/") + "/" + name
	grpcRequest := fileRequest(path, "write")
	grpcRequest.Content = content
	if t.performFile(w, r.PathValue("id"), grpcRequest) != nil {
		httpapi.Success(w, map[string]interface{}{"filename": name})
	}
}
