package agentstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/jobs"
	"github.com/ss497254/gloski/proto/agentv1"
)

// RunJob stores a pending job, marks it running and sends it to the device. A
// send failure is recorded as a failed job.
func (svc *Service) RunJob(deviceID, command, cwd string) (*jobs.Job, error) {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return nil, err
	}
	job, err := svc.jobs.Create(deviceID, command, cwd)
	if err != nil {
		return nil, err
	}
	if err := svc.jobs.MarkRunning(job.ID); err != nil {
		return nil, err
	}
	message := &agentv1.MasterMessage{
		RequestId: job.ID,
		Payload: &agentv1.MasterMessage_RunJob{
			RunJob: &agentv1.RunJob{Command: command, Cwd: cwd},
		},
	}
	if err := conn.send(message); err != nil {
		_ = svc.jobs.Finish(job.ID, -1, err.Error(), false)
		return nil, err
	}
	return job, nil
}

// CancelJob asks the agent to stop a running job. If the device is offline
// the job is recorded as stopped straight away, and the agent is told to
// stop it when it reconnects.
func (svc *Service) CancelJob(deviceID, jobID, reason string) error {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return svc.jobs.Finish(jobID, -1, "", true)
	}
	svc.stopMu.Lock()
	svc.stopping[jobID] = struct{}{}
	svc.stopMu.Unlock()
	err = conn.send(&agentv1.MasterMessage{
		RequestId: jobID,
		Payload: &agentv1.MasterMessage_Cancel{
			Cancel: &agentv1.CancelRequest{Reason: reason},
		},
	})
	if err != nil {
		svc.stopMu.Lock()
		delete(svc.stopping, jobID)
		svc.stopMu.Unlock()
	}
	return err
}

// ListJobs returns a device's jobs from the master database.
func (svc *Service) ListJobs(deviceID string) ([]jobs.Job, error) { return svc.jobs.List(deviceID) }

// GetJob returns one job.
func (svc *Service) GetJob(id string) (*jobs.Job, error) { return svc.jobs.Get(id) }

// JobLogs returns a job's streamed output.
func (svc *Service) JobLogs(id string) ([]string, error) { return svc.jobs.Logs(id) }

// RecentJobs returns the latest jobs across all fleet.
func (svc *Service) RecentJobs(limit int) ([]jobs.Job, error) { return svc.jobs.Recent(limit) }

// ProblemJobs returns failed or abandoned jobs across all fleet.
func (svc *Service) ProblemJobs(limit int) ([]jobs.Job, error) { return svc.jobs.ProblemJobs(limit) }

// ProblemCountsByDevice counts unfinished jobs for every device.
func (svc *Service) ProblemCountsByDevice(ctx context.Context) (map[string]int, error) {
	return svc.jobs.ProblemCountsByDevice(ctx)
}

// DeleteJob removes a job and its logs.
func (svc *Service) DeleteJob(id string) error { return svc.jobs.Delete(id) }

// PruneJobs deletes terminal jobs finished longer ago than olderThan.
func (svc *Service) PruneJobs(olderThan time.Duration) (int64, error) {
	return svc.jobs.PruneFinished(time.Now().UTC().Add(-olderThan))
}

// StartTransfer asks an agent to move file contents over HTTP and reports the
// outcome when it finishes. Only the request and the result travel on the
// stream; the bytes use the agent HTTP port, so a big transfer cannot delay
// terminals or commands.
func (svc *Service) StartTransfer(deviceID string, request *agentv1.FileTransfer) (<-chan error, error) {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return nil, err
	}
	id := request.GetTransferId()
	reply := make(chan *agentv1.AgentMessage, 1)
	conn.mu.Lock()
	conn.pending[id] = reply
	conn.mu.Unlock()

	done := make(chan error, 1)
	if err := conn.send(&agentv1.MasterMessage{
		RequestId: id,
		Payload:   &agentv1.MasterMessage_FileTransfer{FileTransfer: request},
	}); err != nil {
		conn.mu.Lock()
		delete(conn.pending, id)
		conn.mu.Unlock()
		return nil, err
	}

	go func() {
		defer func() {
			conn.mu.Lock()
			delete(conn.pending, id)
			conn.mu.Unlock()
			close(done)
		}()
		select {
		case message := <-reply:
			if message == nil {
				done <- ErrNoAgent
				return
			}
			result := message.GetFileResult()
			switch {
			case result == nil:
				done <- errors.New("agent returned no file transfer result")
			case result.GetCode() != 0:
				done <- errors.New(result.GetMessage())
			}
		case <-conn.closed:
			done <- ErrNoAgent
		}
	}()
	return done, nil
}

// File performs a file operation on an agent and waits for its result.
func (svc *Service) File(ctx context.Context, deviceID string, request *agentv1.FileRequest) (*agentv1.FileResult, error) {
	reply, err := svc.request(ctx, deviceID, &agentv1.MasterMessage{
		Payload: &agentv1.MasterMessage_FileRequest{FileRequest: request},
	}, fileRequestTimeout)
	if err != nil {
		return nil, err
	}
	return reply.GetFileResult(), nil
}

// Processes asks an agent for its current process table.
func (svc *Service) Processes(ctx context.Context, deviceID string, limit int, sortBy string) ([]byte, error) {
	reply, err := svc.request(ctx, deviceID, &agentv1.MasterMessage{
		Payload: &agentv1.MasterMessage_ProcessRequest{
			ProcessRequest: &agentv1.ProcessRequest{Limit: int32(limit), Sort: sortBy},
		},
	}, processRequestTimeout)
	if err != nil {
		return nil, err
	}
	result := reply.GetProcessResult()
	if result == nil {
		return nil, errors.New("agent returned no process result")
	}
	return []byte(result.GetJson()), nil
}

// Check asks an agent to run one native probe and waits for its result.
func (svc *Service) Check(ctx context.Context, deviceID string, definition checks.Definition) (checks.Result, error) {
	accepted := make([]int32, 0, len(definition.Accepted))
	for _, code := range definition.Accepted {
		accepted = append(accepted, int32(code))
	}
	reply, err := svc.request(ctx, deviceID, &agentv1.MasterMessage{
		Payload: &agentv1.MasterMessage_CheckRequest{
			CheckRequest: &agentv1.CheckRequest{
				CheckId:    definition.ID(),
				Kind:       string(definition.Kind),
				Target:     definition.Target,
				Accepted:   accepted,
				TimeoutMs:  int32(definition.Timeout.Milliseconds()),
				SkipVerify: definition.SkipVerify,
			},
		},
	}, checkRequestTimeout)
	if err != nil {
		return checks.Result{}, err
	}
	payload := reply.GetCheckResult()
	if payload == nil {
		return checks.Result{}, errors.New("agent returned no check result")
	}
	details := make([]checks.Detail, 0, len(payload.GetDetails()))
	for _, detail := range payload.GetDetails() {
		details = append(details, checks.Detail{Key: detail.GetKey(), Value: detail.GetValue()})
	}
	return checks.Result{
		CheckID:    definition.ID(),
		DeviceID:   deviceID,
		Status:     checks.Status(payload.GetStatus()),
		Summary:    payload.GetSummary(),
		Details:    details,
		RanAt:      time.Now().UTC(),
		DurationMS: payload.GetDurationMs(),
		Interval:   definition.Interval,
	}, nil
}

// ShutdownMode says what an agent should do after it stops.
type ShutdownMode int

const (
	// ShutdownExit ends the agent process.
	ShutdownExit ShutdownMode = iota
	// ShutdownUninstall removes the agent from the device.
	ShutdownUninstall
)

// Shutdown asks an agent to stop itself.
func (svc *Service) Shutdown(deviceID string, mode ShutdownMode) error {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return err
	}
	return conn.send(&agentv1.MasterMessage{
		Payload: &agentv1.MasterMessage_Shutdown{Shutdown: &agentv1.Shutdown{Uninstall: mode == ShutdownUninstall}},
	})
}

// Connected reports whether an agent stream is live.
func (svc *Service) Connected(deviceID string) bool {
	_, err := svc.connection(deviceID)
	return err == nil
}

// ErrRequestTimeout means the connected agent did not answer before the
// command deadline expired.
var ErrRequestTimeout = errors.New("agent did not respond in time")

func (svc *Service) request(ctx context.Context, deviceID string, message *agentv1.MasterMessage, timeout time.Duration) (*agentv1.AgentMessage, error) {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	message.RequestId = id
	channel := make(chan *agentv1.AgentMessage, 1)
	conn.mu.Lock()
	conn.pending[id] = channel
	conn.mu.Unlock()
	defer func() {
		conn.mu.Lock()
		delete(conn.pending, id)
		conn.mu.Unlock()
	}()

	if err := conn.send(message); err != nil {
		return nil, err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case reply := <-channel:
		if reply == nil {
			return nil, ErrNoAgent
		}
		return reply, nil
	case <-timer.C:
		return nil, fmt.Errorf("%w: %s", ErrRequestTimeout, deviceID)
	}
}

// Disconnect hangs up a device's stream, if it has one.
func (svc *Service) Disconnect(deviceID string) {
	if conn, err := svc.connection(deviceID); err == nil {
		conn.close()
	}
}

// Close hangs up every agent stream so the gRPC server can stop. Agents
// treat this as a dropped connection and reconnect when the master returns.
func (svc *Service) Close() {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	for _, conn := range svc.connections {
		// Not a real disconnect: leave device and job state for the restart.
		conn.replaced.Store(true)
		conn.close()
	}
}

// ForgetDevice removes a device's jobs and logs.
func (svc *Service) ForgetDevice(deviceID string) error { return svc.jobs.DeleteDevice(deviceID) }

// Build reports the architecture and binary hash of a connected agent.
func (svc *Service) Build(deviceID string) (arch, hash string, connected bool) {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return "", "", false
	}
	return conn.arch, conn.buildHash, true
}

// noteRejection remembers why an agent was turned away, keyed by the name it
// gave, so "offline" can be explained.
func (svc *Service) noteRejection(name, reason string) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if svc.rejections == nil {
		svc.rejections = make(map[string]string)
	}
	svc.rejections[name] = reason
}

// Rejection returns why an agent using this name was last turned away, or "".
// A successful connection clears it.
func (svc *Service) Rejection(name string) string {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	return svc.rejections[name]
}

// ActiveJobs counts jobs on a device that have not finished.
func (svc *Service) ActiveJobs(deviceID string) int {
	return svc.jobs.Active(deviceID)
}
