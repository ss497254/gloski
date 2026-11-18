package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"sync"

	"github.com/ss497254/gloski/proto/agentv1"
)

func (a *agentRuntime) runJob(id string, request *agentv1.RunJob) {
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.jobs[id] = cancel
	a.mu.Unlock()
	defer func() {
		cancel()
		a.mu.Lock()
		delete(a.jobs, id)
		a.mu.Unlock()
	}()

	shell := a.cfg.Shell
	if shell == "" {
		shell = "/bin/sh"
	}
	command := exec.CommandContext(ctx, shell, "-c", request.GetCommand())
	if cwd := strings.TrimSpace(request.GetCwd()); cwd != "" {
		command.Dir = cwd
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		a.sendResult(id, -1, err.Error())
		return
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		a.sendResult(id, -1, err.Error())
		return
	}
	if err := command.Start(); err != nil {
		a.sendResult(id, -1, err.Error())
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		a.streamOutput(id, stdout, false)
	}()
	go func() {
		defer wg.Done()
		a.streamOutput(id, stderr, true)
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
	a.sendResult(id, code, message)
}

func (a *agentRuntime) streamOutput(id string, reader interface{ Read([]byte) (int, error) }, stderr bool) {
	buffer := make([]byte, 16*1024)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			sendErr := a.send(&agentv1.AgentMessage{
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

func (a *agentRuntime) sendResult(id string, code int, message string) {
	_ = a.send(&agentv1.AgentMessage{
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

func (a *agentRuntime) handleFile(id string, request *agentv1.FileRequest) {
	result := &agentv1.FileResult{}
	switch request.GetOp() {
	case agentv1.FileOp_FILE_OP_LIST:
		listing, err := a.files.List(request.GetPath())
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
		content, err := a.files.Read(request.GetPath())
		if err != nil {
			result = fileError(err)
			break
		}
		result.Content = []byte(content)
	case agentv1.FileOp_FILE_OP_WRITE:
		if err := a.files.Write(request.GetPath(), string(request.GetContent())); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_DELETE:
		if err := a.files.Delete(request.GetPath()); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_MKDIR:
		if err := a.files.Mkdir(request.GetPath()); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_RENAME:
		if err := a.files.Rename(request.GetPath(), request.GetNewPath()); err != nil {
			result = fileError(err)
		}
	default:
		result.Code, result.Message = 1, "unsupported file operation"
	}
	_ = a.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload:   &agentv1.AgentMessage_FileResult{FileResult: result},
	})
}

func (a *agentRuntime) handleProcesses(id string, request *agentv1.ProcessRequest) {
	limit := int(request.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	processes, err := a.stats.GetProcesses(limit, request.GetSort())
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
	_ = a.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload:   &agentv1.AgentMessage_ProcessResult{ProcessResult: result},
	})
}

func jsonLine(key, value string) string {
	raw, _ := json.Marshal(map[string]string{key: value})
	return string(raw)
}
