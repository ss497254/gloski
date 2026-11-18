// Package agentgrpc implements the master side of the single bidirectional
// gRPC stream used by every agent, including the master's local agent.
package agentgrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/devices"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/system"
	"github.com/ss497254/gloski/proto/agentv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrNoAgent = errors.New("agent is not connected")

// Server implements the Agent service and keeps master-owned state.
type Server struct {
	agentv1.UnimplementedAgentServer

	devices  *devices.Service
	jobs     *JobStore
	version  string
	interval int

	mu       sync.RWMutex
	sessions map[string]*session
}

type session struct {
	deviceID string
	token    string
	stream   agentv1.Agent_ConnectServer
	replaced bool

	sendMu  sync.Mutex
	mu      sync.Mutex
	pending map[string]chan *agentv1.AgentMessage
	terms   map[string]chan []byte
}

// New creates an agent gRPC service backed by the master database.
func New(db *database.Database, devicesSvc *devices.Service, version string, interval int) (*Server, error) {
	jobs, err := NewJobStore(db)
	if err != nil {
		return nil, err
	}
	if interval <= 0 {
		interval = 10
	}
	return &Server{
		devices:  devicesSvc,
		jobs:     jobs,
		version:  version,
		interval: interval,
		sessions: make(map[string]*session),
	}, nil
}

// Connect handles one agent's bidirectional stream for its whole lifetime.
func (s *Server) Connect(stream agentv1.Agent_ConnectServer) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	registration := first.GetRegister()
	if registration == nil {
		return status.Error(codes.InvalidArgument, "first message must be a registration")
	}

	stats := decodeStats(registration.GetStatsJson())
	registered, err := s.devices.Register(devices.RegisterRequest{
		JoinToken: registration.GetJoinToken(),
		Name:      registration.GetName(),
		URL:       "grpc://agent",
		Hostname:  registration.GetHostname(),
		Platform:  registration.GetPlatform(),
		Version:   registration.GetVersion(),
		DeployID:  registration.GetDeployId(),
		Master:    registration.GetMaster(),
		Stats:     stats,
	}, "")
	if err != nil {
		return status.Error(codes.Unauthenticated, "invalid join token")
	}
	device, err := s.devices.Get(registered.DeviceID)
	if err != nil {
		return status.Error(codes.Internal, "failed to load device")
	}

	sess := &session{
		deviceID: device.ID,
		token:    registered.DeviceToken,
		stream:   stream,
		pending:  make(map[string]chan *agentv1.AgentMessage),
		terms:    make(map[string]chan []byte),
	}
	s.mu.Lock()
	if previous := s.sessions[device.ID]; previous != nil {
		previous.replaced = true
	}
	s.sessions[device.ID] = sess
	s.mu.Unlock()
	defer s.disconnect(sess)

	response := &agentv1.MasterMessage{
		RequestId: first.RequestId,
		Payload: &agentv1.MasterMessage_RegisterResponse{
			RegisterResponse: &agentv1.RegisterResponse{
				DeviceId:          device.ID,
				DeviceToken:       registered.DeviceToken,
				HeartbeatInterval: int32(s.interval),
				MasterVersion:     s.version,
			},
		},
	}
	if err := sess.send(response); err != nil {
		return err
	}

	logger.Info("Agent connected over gRPC: %s (%s)", device.Name, device.ID)
	for {
		message, err := stream.Recv()
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				logger.Debug("Agent stream %s closed: %v", device.ID, err)
			}
			return nil
		}
		s.handle(sess, message)
	}
}

func (s *Server) handle(sess *session, message *agentv1.AgentMessage) {
	switch payload := message.Payload.(type) {
	case *agentv1.AgentMessage_Heartbeat:
		stats := decodeStats(payload.Heartbeat.GetStatsJson())
		if _, err := s.devices.Heartbeat(sess.token, devices.HeartbeatRequest{Stats: stats}); err != nil {
			logger.Warn("Heartbeat from %s rejected: %v", sess.deviceID, err)
		}
	case *agentv1.AgentMessage_JobOutput:
		if err := s.jobs.AppendLog(message.RequestId, payload.JobOutput.GetChunk(), payload.JobOutput.GetStderr()); err != nil {
			logger.Warn("Failed to store job output for %s: %v", message.RequestId, err)
		}
	case *agentv1.AgentMessage_JobResult:
		result := payload.JobResult
		if err := s.jobs.Finish(message.RequestId, int(result.GetExitCode()), result.GetError()); err != nil {
			logger.Warn("Failed to finish job %s: %v", message.RequestId, err)
		}
	case *agentv1.AgentMessage_FileResult:
		sess.respond(message.RequestId, message)
	case *agentv1.AgentMessage_ProcessResult:
		sess.respond(message.RequestId, message)
	case *agentv1.AgentMessage_TerminalOutput:
		sess.terminalOutput(message.RequestId, payload.TerminalOutput)
	}
}

func (s *Server) disconnect(sess *session) {
	s.mu.Lock()
	if s.sessions[sess.deviceID] == sess {
		delete(s.sessions, sess.deviceID)
	}
	s.mu.Unlock()

	sess.mu.Lock()
	for id, channel := range sess.pending {
		close(channel)
		delete(sess.pending, id)
	}
	for id, channel := range sess.terms {
		close(channel)
		delete(sess.terms, id)
	}
	sess.mu.Unlock()

	if sess.replaced {
		logger.Info("Agent session replaced: %s", sess.deviceID)
		return
	}
	if err := s.devices.MarkOffline(sess.deviceID); err != nil {
		logger.Warn("Failed to mark device %s offline: %v", sess.deviceID, err)
	}
	if err := s.jobs.AbortDevice(sess.deviceID, "agent disconnected"); err != nil {
		logger.Warn("Failed to abort jobs for %s: %v", sess.deviceID, err)
	}
	logger.Info("Agent disconnected: %s", sess.deviceID)
}

func (s *Server) session(deviceID string) (*session, error) {
	s.mu.RLock()
	sess := s.sessions[deviceID]
	s.mu.RUnlock()
	if sess == nil {
		return nil, ErrNoAgent
	}
	return sess, nil
}

// RunJob starts a master-owned job on an agent and returns its ID.
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
	}, 20*time.Second)
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
	}, 10*time.Second)
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
	// The resize protocol from the browser uses a leading 0x01 byte.
	if len(data) > 0 && data[0] == 0x01 {
		return sess.send(&agentv1.MasterMessage{
			RequestId: id,
			Payload: &agentv1.MasterMessage_TerminalInput{
				TerminalInput: &agentv1.TerminalInput{Data: data},
			},
		})
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

func (sess *session) send(message *agentv1.MasterMessage) error {
	sess.sendMu.Lock()
	defer sess.sendMu.Unlock()
	return sess.stream.Send(message)
}

func (sess *session) respond(id string, message *agentv1.AgentMessage) {
	sess.mu.Lock()
	channel := sess.pending[id]
	sess.mu.Unlock()
	if channel == nil {
		return
	}
	select {
	case channel <- message:
	default:
	}
}

func (sess *session) terminalOutput(id string, output *agentv1.TerminalOutput) {
	sess.mu.Lock()
	channel := sess.terms[id]
	if output.GetClosed() {
		delete(sess.terms, id)
	}
	sess.mu.Unlock()
	if channel == nil {
		return
	}
	if output.GetClosed() {
		close(channel)
		return
	}
	select {
	case channel <- output.GetData():
	default:
	}
}

func decodeStats(raw string) *system.Stats {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var stats system.Stats
	if err := json.Unmarshal([]byte(raw), &stats); err != nil {
		logger.Warn("Failed to decode agent stats: %v", err)
		return nil
	}
	return &stats
}
