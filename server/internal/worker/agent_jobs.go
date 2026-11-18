package worker

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/ss497254/gloski/proto/agentv1"
)

// jobStopGrace is how long a cancelled job gets between SIGTERM and SIGKILL,
// and how long output pipes may stay open after the shell itself has exited.
const jobStopGrace = 5 * time.Second

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
	// The shell gets its own process group so cancelling reaches everything it
	// started, not just the shell.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		return syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
	}
	// exec copies output through these writers and Wait returns only after
	// the copy is done, so no trailing output is lost. WaitDelay bounds that
	// wait when a background child keeps the pipes open.
	command.WaitDelay = jobStopGrace
	command.Stdout = &jobOutput{runtime: w, id: id}
	command.Stderr = &jobOutput{runtime: w, id: id, stderr: true}

	if err := command.Start(); err != nil {
		w.sendResult(id, -1, err.Error())
		return
	}
	pid := command.Process.Pid

	err := command.Wait()
	if ctx.Err() != nil {
		// Cancelled: make sure nothing that ignored SIGTERM is left behind.
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}

	code := 0
	message := ""
	switch {
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
		// ErrWaitDelay means the shell exited but a child still held the
		// output pipe; the shell's own exit status is what counts.
		if command.ProcessState != nil {
			code = command.ProcessState.ExitCode()
		}
		if code != 0 {
			message = command.ProcessState.String()
		}
	default:
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

// jobOutput forwards one output stream of a job to the master.
type jobOutput struct {
	runtime *Runtime
	id      string
	stderr  bool
}

func (o *jobOutput) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	chunk := make([]byte, len(data))
	copy(chunk, data)
	err := o.runtime.send(&agentv1.AgentMessage{
		RequestId: o.id,
		Payload: &agentv1.AgentMessage_JobOutput{
			JobOutput: &agentv1.JobOutput{Chunk: chunk, Stderr: o.stderr},
		},
	})
	if err != nil {
		return 0, err
	}
	return len(data), nil
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
