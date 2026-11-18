package agents

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/proto/agentv1"
)

func (svc *Service) RunJob(deviceID, command, cwd string) (*Job, error) {
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
func (svc *Service) ListJobs(deviceID string) ([]Job, error) { return svc.jobs.List(deviceID) }

// GetJob returns one job.
func (svc *Service) GetJob(id string) (*Job, error) { return svc.jobs.Get(id) }

// JobLogs returns a job's streamed output.
func (svc *Service) JobLogs(id string) ([]string, error) { return svc.jobs.Logs(id) }

// RecentJobs returns the latest jobs across all fleet.
func (svc *Service) RecentJobs(limit int) ([]Job, error) { return svc.jobs.Recent(limit) }

// ProblemJobs returns failed or abandoned jobs across all fleet.
func (svc *Service) ProblemJobs(limit int) ([]Job, error) { return svc.jobs.ProblemJobs(limit) }

// DeleteJob removes a job and its logs.
func (svc *Service) DeleteJob(id string) error { return svc.jobs.Delete(id) }

// File performs a file operation on an agent and waits for its result.
func (svc *Service) File(deviceID string, request *agentv1.FileRequest) (*agentv1.FileResult, error) {
	reply, err := svc.request(deviceID, &agentv1.MasterMessage{
		Payload: &agentv1.MasterMessage_FileRequest{FileRequest: request},
	}, fileRequestTimeout)
	if err != nil {
		return nil, err
	}
	return reply.GetFileResult(), nil
}

// Processes asks an agent for its current process table.
func (svc *Service) Processes(deviceID string, limit int, sortBy string) ([]byte, error) {
	reply, err := svc.request(deviceID, &agentv1.MasterMessage{
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

// Shutdown asks an agent to stop itself.
func (svc *Service) Shutdown(deviceID string) error {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return err
	}
	return conn.send(&agentv1.MasterMessage{
		Payload: &agentv1.MasterMessage_Shutdown{Shutdown: &agentv1.Shutdown{}},
	})
}

// Connected reports whether an agent stream is live.
func (svc *Service) Connected(deviceID string) bool {
	_, err := svc.connection(deviceID)
	return err == nil
}

func (svc *Service) request(deviceID string, message *agentv1.MasterMessage, timeout time.Duration) (*agentv1.AgentMessage, error) {
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
	case reply := <-channel:
		if reply == nil {
			return nil, ErrNoAgent
		}
		return reply, nil
	case <-timer.C:
		return nil, fmt.Errorf("agent %s did not respond in time", deviceID)
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
