package worker

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"sync"

	"github.com/ss497254/gloski/proto/agentv1"
)

func (w *Runtime) runJob(id string, request *agentv1.RunJob) {
	ctx, cancel := context.WithCancel(context.Background())
	w.mu.Lock()
	w.jobs[id] = cancel
	w.mu.Unlock()
	defer func() {
		cancel()
		w.mu.Lock()
		delete(w.jobs, id)
		w.mu.Unlock()
	}()

	shell := w.cfg.Shell
	if shell == "" {
		shell = "/bin/sh"
	}
	command := exec.CommandContext(ctx, shell, "-c", request.GetCommand())
	if cwd := strings.TrimSpace(request.GetCwd()); cwd != "" {
		command.Dir = cwd
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		w.sendResult(id, -1, err.Error())
		return
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		w.sendResult(id, -1, err.Error())
		return
	}
	if err := command.Start(); err != nil {
		w.sendResult(id, -1, err.Error())
		return
	}

	// Both pipes must be drained while the command runs. Waiting on the
	// process first can deadlock if the child fills one pipe buffer.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		w.streamOutput(id, stdout, false)
	}()
	go func() {
		defer wg.Done()
		w.streamOutput(id, stderr, true)
	}()

	err = command.Wait()
	wg.Wait()
	code := 0
	message := ""
	if err != nil {
		message = err.Error()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		} else {
			code = -1
		}
	}
	w.sendResult(id, code, message)
}

func (w *Runtime) streamOutput(id string, reader interface{ Read([]byte) (int, error) }, stderr bool) {
	buffer := make([]byte, 16*1024)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			sendErr := w.send(&agentv1.AgentMessage{
				RequestId: id,
				Payload: &agentv1.AgentMessage_JobOutput{
					JobOutput: &agentv1.JobOutput{Chunk: chunk, Stderr: stderr},
				},
			})
			if sendErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (w *Runtime) sendResult(id string, code int, message string) {
	_ = w.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload: &agentv1.AgentMessage_JobResult{
			JobResult: &agentv1.JobResult{ExitCode: int32(code), Error: message},
		},
	})
}

const fileErrorCode = 1

func fileError(err error) *agentv1.FileResult {
	return &agentv1.FileResult{Code: fileErrorCode, Message: err.Error()}
}

func (w *Runtime) handleFile(id string, request *agentv1.FileRequest) {
	result := &agentv1.FileResult{}
	switch request.GetOp() {
	case agentv1.FileOp_FILE_OP_LIST:
		listing, err := w.files.List(request.GetPath())
		if err != nil {
			result = fileError(err)
			break
		}
		raw, err := json.Marshal(listing)
		if err != nil {
			result = fileError(err)
			break
		}
		result.EntriesJson = string(raw)
	case agentv1.FileOp_FILE_OP_READ:
		content, err := w.files.Read(request.GetPath())
		if err != nil {
			result = fileError(err)
			break
		}
		result.Content = []byte(content)
	case agentv1.FileOp_FILE_OP_WRITE:
		if err := w.files.Write(request.GetPath(), string(request.GetContent())); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_DELETE:
		if err := w.files.Delete(request.GetPath()); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_MKDIR:
		if err := w.files.Mkdir(request.GetPath()); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_RENAME:
		if err := w.files.Rename(request.GetPath(), request.GetNewPath()); err != nil {
			result = fileError(err)
		}
	default:
		result.Code, result.Message = 1, "unsupported file operation"
	}
	_ = w.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload:   &agentv1.AgentMessage_FileResult{FileResult: result},
	})
}

func (w *Runtime) handleProcesses(id string, request *agentv1.ProcessRequest) {
	limit := int(request.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	processes, err := w.stats.GetProcesses(limit, request.GetSort())
	result := &agentv1.ProcessResult{}
	if err != nil {
		result.Json = jsonLine("error", err.Error())
	} else {
		raw, marshalErr := json.Marshal(processes)
		if marshalErr != nil {
			result.Json = jsonLine("error", marshalErr.Error())
		} else {
			result.Json = string(raw)
		}
	}
	_ = w.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload:   &agentv1.AgentMessage_ProcessResult{ProcessResult: result},
	})
}

func jsonLine(key, value string) string {
	raw, _ := json.Marshal(map[string]string{key: value})
	return string(raw)
}
