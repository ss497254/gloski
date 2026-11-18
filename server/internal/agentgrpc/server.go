// Package agentgrpc implements the master side of the single bidirectional
// gRPC stream used by every agent, including the master's local agent.
package agentgrpc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/devices"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/system"
	"github.com/ss497254/gloski/proto/agentv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultHeartbeatInterval = 10
	agentURLLabel            = "grpc://agent"
	fileRequestTimeout       = 20 * time.Second
	processRequestTimeout    = 10 * time.Second
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

// New creates the agent service backed by master-owned job storage.
func New(db *database.Database, devicesSvc *devices.Service, version string, interval int) (*Server, error) {
	jobs, err := NewJobStore(db)
	if err != nil {
		return nil, err
	}
	if interval <= 0 {
		interval = defaultHeartbeatInterval
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
		URL:       agentURLLabel,
		Hostname:  registration.GetHostname(),
		Platform:  registration.GetPlatform(),
		Version:   registration.GetVersion(),
		DeployID:  registration.GetDeployId(),
		Master:    registration.GetMaster(),
		Stats:     stats,
	})
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
