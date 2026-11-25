package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/proto/agentv1"
)

const (
	// jobStopGrace is how long a cancelled job gets between SIGTERM and
	// SIGKILL, and how long output pipes may stay open after the shell itself
	// has exited.
	jobStopGrace = 5 * time.Second
	// jobReplayBytes bounds the output each job keeps in memory for replay
	// after a reconnect. Older output is dropped first; the master marks the
	// gap in the log.
	jobReplayBytes = 4 << 20
)

type outputChunk struct {
	seq    uint64
	data   []byte
	stderr bool
}

// job is one command and the output the master may still need. It exists
// from dispatch until the master acknowledges the result, independent of
// whether a stream is connected.
type job struct {
	id     string
	cancel context.CancelFunc
	done   chan struct{} // closed once the process has exited

	// mu orders output: chunks are appended and sent under it, so the master
	// always receives them in sequence.
	mu      sync.Mutex
	chunks  []outputChunk // consecutive sequence numbers, oldest first
	size    int
	lastSeq uint64 // newest chunk produced
	sentSeq uint64 // newest chunk handed to the stream

	finished   bool
	exitCode   int32
	errMsg     string
	resultSent bool
}

// startJob runs a command. The job is registered before this returns so a
// Cancel that follows immediately on the stream always finds it.
func (w *Runtime) startJob(id string, request *agentv1.RunJob) {
	ctx, cancel := context.WithCancel(context.Background())
	j := &job{id: id, cancel: cancel, done: make(chan struct{})}

	w.mu.Lock()
	if _, exists := w.jobs[id]; exists {
		w.mu.Unlock()
		cancel()
		return // duplicate dispatch
	}
	w.jobs[id] = j
	w.mu.Unlock()

	go w.runJob(ctx, j, request)
}

func (w *Runtime) runJob(ctx context.Context, j *job, request *agentv1.RunJob) {
	defer close(j.done)
	defer j.cancel()

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
	command.Stdout = &jobOutput{runtime: w, job: j}
	command.Stderr = &jobOutput{runtime: w, job: j, stderr: true}

	if err := command.Start(); err != nil {
		w.finishJob(j, -1, err.Error())
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
	w.finishJob(j, code, message)
}

// jobOutput receives one output stream of a job.
type jobOutput struct {
	runtime *Runtime
	job     *job
	stderr  bool
}

// Write never fails: while the master is unreachable the output waits in the
// job's replay buffer instead of breaking the command's pipe.
func (o *jobOutput) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	chunk := make([]byte, len(data))
	copy(chunk, data)

	j := o.job
	j.mu.Lock()
	defer j.mu.Unlock()
	j.lastSeq++
	j.chunks = append(j.chunks, outputChunk{seq: j.lastSeq, data: chunk, stderr: o.stderr})
	j.size += len(chunk)
	for j.size > jobReplayBytes && len(j.chunks) > 1 {
		j.size -= len(j.chunks[0].data)
		j.chunks[0].data = nil
		j.chunks = j.chunks[1:]
	}
	o.runtime.flushLocked(j)
	return len(data), nil
}

func (w *Runtime) finishJob(j *job, code int, message string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.finished = true
	j.exitCode = int32(code)
	j.errMsg = message
	w.flushLocked(j)
}

// flushLocked sends, in order, every chunk the master has not been handed
// yet, then the result if the job is done. It stops quietly at the first
// failure; the next reconnect rewinds and replays. Callers hold j.mu.
func (w *Runtime) flushLocked(j *job) {
	if len(j.chunks) > 0 {
		// Sequence numbers are consecutive, so the next unsent chunk is at a
		// fixed offset. A negative offset means older output was dropped.
		start := 0
		if first := j.chunks[0].seq; j.sentSeq+1 > first {
			start = int(j.sentSeq + 1 - first)
		}
		for i := start; i < len(j.chunks); i++ {
			chunk := j.chunks[i]
			if err := w.send(&agentv1.AgentMessage{
				RequestId: j.id,
				Payload: &agentv1.AgentMessage_JobOutput{
					JobOutput: &agentv1.JobOutput{Chunk: chunk.data, Stderr: chunk.stderr, Seq: chunk.seq},
				},
			}); err != nil {
				return
			}
			j.sentSeq = chunk.seq
		}
	}
	if j.finished && !j.resultSent {
		if err := w.send(&agentv1.AgentMessage{
			RequestId: j.id,
			Payload: &agentv1.AgentMessage_JobResult{
				JobResult: &agentv1.JobResult{ExitCode: j.exitCode, Error: j.errMsg},
			},
		}); err == nil {
			j.resultSent = true
		}
	}
}

// jobStates reports every held job to the master during registration.
func (w *Runtime) jobStates() []*agentv1.JobState {
	w.mu.Lock()
	defer w.mu.Unlock()
	states := make([]*agentv1.JobState, 0, len(w.jobs))
	for _, j := range w.jobs {
		j.mu.Lock()
		states = append(states, &agentv1.JobState{
			Id:       j.id,
			LastSeq:  j.lastSeq,
			Finished: j.finished,
			ExitCode: j.exitCode,
			Error:    j.errMsg,
		})
		j.mu.Unlock()
	}
	return states
}

// resumeJobs applies the master's answer: rewind each job it still tracks to
// the last chunk it stored, and stop the ones it has given up on. A job the
// master did not mention is treated as forgotten, which is also how a master
// that predates job resumption behaves.
func (w *Runtime) resumeJobs(resumes []*agentv1.JobResume) {
	keep := make(map[string]uint64, len(resumes))
	for _, resume := range resumes {
		if !resume.GetForget() {
			keep[resume.GetId()] = resume.GetAckedSeq()
		}
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	for id, j := range w.jobs {
		acked, ok := keep[id]
		if !ok {
			logger.Info("Master no longer tracks job %s; dropping it", id)
			j.cancel()
			delete(w.jobs, id)
			continue
		}
		j.mu.Lock()
		j.sentSeq = acked
		j.resultSent = false
		j.mu.Unlock()
	}
}

// flushJobs replays buffered output for every job after a reconnect.
func (w *Runtime) flushJobs() {
	w.mu.Lock()
	jobs := make([]*job, 0, len(w.jobs))
	for _, j := range w.jobs {
		jobs = append(jobs, j)
	}
	w.mu.Unlock()
	for _, j := range jobs {
		j.mu.Lock()
		w.flushLocked(j)
		j.mu.Unlock()
	}
}

func (w *Runtime) cancelJob(id string) {
	w.mu.Lock()
	j := w.jobs[id]
	w.mu.Unlock()
	if j != nil {
		j.cancel()
	}
}

// forgetJob drops a finished job once the master has stored its result.
func (w *Runtime) forgetJob(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	j := w.jobs[id]
	if j == nil {
		return
	}
	j.mu.Lock()
	finished := j.finished
	j.mu.Unlock()
	if finished {
		delete(w.jobs, id)
	}
}

// stopJobs cancels every job and waits briefly for them to exit. It runs
// when the agent itself is stopping.
func (w *Runtime) stopJobs() {
	w.mu.Lock()
	jobs := make([]*job, 0, len(w.jobs))
	for _, j := range w.jobs {
		jobs = append(jobs, j)
	}
	w.mu.Unlock()

	for _, j := range jobs {
		j.cancel()
	}
	deadline := time.After(jobStopGrace + time.Second)
	for _, j := range jobs {
		select {
		case <-j.done:
		case <-deadline:
			return
		}
	}
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
