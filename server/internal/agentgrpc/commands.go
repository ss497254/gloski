// Commands the master sends to an agent over its Connect stream.

package agentgrpc

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/proto/agentv1"
)

// RunJob records a new job and sends it to the target agent over its stream.
func (s *Server) RunJob(deviceID, command, cwd string) (*Job, error) {
	sess, err := s.session(deviceID)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.Create(deviceID, command, cwd)
	if err != nil {
		return nil, err
	}
	if err := s.jobs.MarkRunning(job.ID); err != nil {
		return nil, err
	}
	message := &agentv1.MasterMessage{
		RequestId: job.ID,
		Payload: &agentv1.MasterMessage_RunJob{
			RunJob: &agentv1.RunJob{Command: command, Cwd: cwd},
		},
	}
	if err := sess.send(message); err != nil {
		_ = s.jobs.Finish(job.ID, -1, err.Error())
		return nil, err
	}
	return job, nil
}

// CancelJob asks the agent to stop a running job.
func (s *Server) CancelJob(deviceID, jobID, reason string) error {
	sess, err := s.session(deviceID)
	if err != nil {
		return err
	}
	return sess.send(&agentv1.MasterMessage{
		RequestId: jobID,
		Payload: &agentv1.MasterMessage_Cancel{
			Cancel: &agentv1.CancelRequest{Reason: reason},
		},
	})
}

// ListJobs returns a device's jobs from the master database.
func (s *Server) ListJobs(deviceID string) ([]Job, error) { return s.jobs.List(deviceID) }

// GetJob returns one job.
func (s *Server) GetJob(id string) (*Job, error) { return s.jobs.Get(id) }

// JobLogs returns a job's streamed output.
func (s *Server) JobLogs(id string) ([]string, error) { return s.jobs.Logs(id) }

// RecentJobs returns the latest jobs across all devices.
func (s *Server) RecentJobs(limit int) ([]Job, error) { return s.jobs.Recent(limit) }

// ProblemJobs returns failed or abandoned jobs across all devices.
func (s *Server) ProblemJobs(limit int) ([]Job, error) { return s.jobs.ProblemJobs(limit) }

// DeleteJob removes a job and its logs.
func (s *Server) DeleteJob(id string) error { return s.jobs.Delete(id) }

// File performs a file operation on an agent and waits for its result.
func (s *Server) File(deviceID string, request *agentv1.FileRequest) (*agentv1.FileResult, error) {
	reply, err := s.request(deviceID, &agentv1.MasterMessage{
		Payload: &agentv1.MasterMessage_FileRequest{FileRequest: request},
	}, fileRequestTimeout)
	if err != nil {
		return nil, err
	}
	return reply.GetFileResult(), nil
}

// Processes asks an agent for its current process table.
func (s *Server) Processes(deviceID string, limit int, sortBy string) ([]byte, error) {
	reply, err := s.request(deviceID, &agentv1.MasterMessage{
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

// OpenTerminal starts a PTY session on an agent and returns its output channel.
func (s *Server) OpenTerminal(deviceID, cwd string, cols, rows uint32) (<-chan []byte, string, error) {
	sess, err := s.session(deviceID)
	if err != nil {
		return nil, "", err
	}
	id := uuid.NewString()
	channel := make(chan []byte, 256)
	sess.mu.Lock()
	sess.terms[id] = channel
	sess.mu.Unlock()

	message := &agentv1.MasterMessage{
		RequestId: id,
		Payload: &agentv1.MasterMessage_TerminalOpen{
			TerminalOpen: &agentv1.TerminalOpen{Cwd: cwd, Cols: cols, Rows: rows},
		},
	}
	if err := sess.send(message); err != nil {
		sess.mu.Lock()
		delete(sess.terms, id)
		sess.mu.Unlock()
		return nil, "", err
	}
	return channel, id, nil
}

// TerminalInput forwards keystrokes to an agent PTY.
func (s *Server) TerminalInput(deviceID, id string, data []byte) error {
	sess, err := s.session(deviceID)
	if err != nil {
		return err
	}
	return sess.send(&agentv1.MasterMessage{
		RequestId: id,
		Payload: &agentv1.MasterMessage_TerminalInput{
			TerminalInput: &agentv1.TerminalInput{Data: data},
		},
	})
}

// CloseTerminal closes an agent PTY session.
func (s *Server) CloseTerminal(deviceID, id string) error {
	sess, err := s.session(deviceID)
	if err != nil {
		return err
	}
	return sess.send(&agentv1.MasterMessage{
		RequestId: id,
		Payload:   &agentv1.MasterMessage_TerminalClose{TerminalClose: &agentv1.TerminalClose{}},
	})
}

// Shutdown asks an agent to stop itself.
func (s *Server) Shutdown(deviceID string) error {
	sess, err := s.session(deviceID)
	if err != nil {
		return err
	}
	return sess.send(&agentv1.MasterMessage{
		Payload: &agentv1.MasterMessage_Shutdown{Shutdown: &agentv1.Shutdown{}},
	})
}

// Connected reports whether an agent stream is live.
func (s *Server) Connected(deviceID string) bool {
	_, err := s.session(deviceID)
	return err == nil
}

func (s *Server) request(deviceID string, message *agentv1.MasterMessage, timeout time.Duration) (*agentv1.AgentMessage, error) {
	sess, err := s.session(deviceID)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	message.RequestId = id
	channel := make(chan *agentv1.AgentMessage, 1)
	sess.mu.Lock()
	sess.pending[id] = channel
	sess.mu.Unlock()
	defer func() {
		sess.mu.Lock()
		delete(sess.pending, id)
		sess.mu.Unlock()
	}()

	if err := sess.send(message); err != nil {
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
