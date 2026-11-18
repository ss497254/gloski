package master

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ss497254/gloski/internal/agentgrpc"
	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/devices"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/proto/agentv1"
)

var deviceWSUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // tailnet access is controlled by tailscale serve / ACLs
	},
}

func (s *Server) handleDeviceStats(w http.ResponseWriter, r *http.Request) {
	device, err := s.devices.Get(r.PathValue("id"))
	if err != nil {
		handlers.NotFound(w, "device not found")
		return
	}
	handlers.Success(w, device.Stats)
}

func (s *Server) handleDeviceStatsWS(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if _, err := s.devices.Get(deviceID); err != nil {
		handlers.NotFound(w, "device not found")
		return
	}
	connection, err := deviceWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		device, err := s.devices.Get(deviceID)
		if err != nil {
			return
		}
		if device.Stats != nil {
			raw, err := json.Marshal(device.Stats)
			if err != nil {
				return
			}
			if err := connection.WriteMessage(websocket.TextMessage, raw); err != nil {
				return
			}
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) handleDeviceProcesses(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	raw, err := s.agents.Processes(r.PathValue("id"), limit, r.URL.Query().Get("sort"))
	if err != nil {
		writeAgentError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.agents.ListJobs(r.PathValue("id"))
	if err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.Success(w, map[string]interface{}{"jobs": jobs})
}

func (s *Server) handleRunJob(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Command string `json:"command"`
		CWD     string `json:"cwd"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		handlers.BadRequest(w, "invalid job request")
		return
	}
	if strings.TrimSpace(request.Command) == "" {
		handlers.BadRequest(w, "command is required")
		return
	}
	job, err := s.agents.RunJob(r.PathValue("id"), request.Command, request.CWD)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.Success(w, job)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.agents.GetJob(r.PathValue("jobId"))
	if err != nil {
		handlers.NotFound(w, "job not found")
		return
	}
	handlers.Success(w, job)
}

func (s *Server) handleJobLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.agents.JobLogs(r.PathValue("jobId"))
	if err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.Success(w, map[string]interface{}{"logs": logs})
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	if err := s.agents.CancelJob(r.PathValue("id"), r.PathValue("jobId"), "cancelled by user"); err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.SuccessWithMessage(w, nil)
}

func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	if err := s.agents.DeleteJob(r.PathValue("jobId")); err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.SuccessWithMessage(w, nil)
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

func (s *Server) performFile(w http.ResponseWriter, deviceID string, request *agentv1.FileRequest) *agentv1.FileResult {
	result, err := s.agents.File(deviceID, request)
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

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "~"
	}
	result := s.performFile(w, r.PathValue("id"), fileRequest(path, "list"))
	if result == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(result.GetEntriesJson()))
}

func (s *Server) handleReadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	result := s.performFile(w, r.PathValue("id"), fileRequest(path, "read"))
	if result == nil {
		return
	}
	handlers.Success(w, map[string]interface{}{"content": string(result.GetContent()), "path": path})
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	result := s.performFile(w, r.PathValue("id"), fileRequest(path, "read"))
	if result == nil {
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(result.GetContent())
}

func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if s.performFile(w, r.PathValue("id"), fileRequest(path, "delete")) != nil {
		handlers.SuccessWithMessage(w, nil)
	}
}

func (s *Server) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20)).Decode(&request); err != nil {
		handlers.BadRequest(w, "invalid write request")
		return
	}
	grpcRequest := fileRequest(request.Path, "write")
	grpcRequest.Content = []byte(request.Content)
	if s.performFile(w, r.PathValue("id"), grpcRequest) != nil {
		handlers.SuccessWithMessage(w, nil)
	}
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		handlers.BadRequest(w, "invalid mkdir request")
		return
	}
	if s.performFile(w, r.PathValue("id"), fileRequest(request.Path, "mkdir")) != nil {
		handlers.SuccessWithMessage(w, nil)
	}
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	var request struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		handlers.BadRequest(w, "invalid rename request")
		return
	}
	grpcRequest := fileRequest(request.OldPath, "rename")
	grpcRequest.NewPath = request.NewPath
	if s.performFile(w, r.PathValue("id"), grpcRequest) != nil {
		handlers.SuccessWithMessage(w, nil)
	}
}

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
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
	if s.performFile(w, r.PathValue("id"), grpcRequest) != nil {
		handlers.Success(w, map[string]interface{}{"filename": header.Filename})
	}
}

func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	cwd := r.URL.Query().Get("cwd")
	connection, err := deviceWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	output, id, err := s.agents.OpenTerminal(deviceID, cwd, 80, 24)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	defer func() { _ = s.agents.CloseTerminal(deviceID, id) }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for data := range output {
			if err := connection.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		}
	}()

	connection.SetReadLimit(1 << 20)
	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			return
		}
		if err := s.agents.TerminalInput(deviceID, id, message); err != nil {
			logger.Warn("Terminal input failed for %s: %v", deviceID, err)
			return
		}
	}
}

func writeAgentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, agentgrpc.ErrNoAgent):
		handlers.Error(w, http.StatusBadGateway, err.Error())
	case errors.Is(err, devices.ErrDeviceNotFound):
		handlers.NotFound(w, "device not found")
	default:
		handlers.InternalError(w, "agent request failed", err.Error())
	}
}
