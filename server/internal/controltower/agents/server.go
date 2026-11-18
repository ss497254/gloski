// Package agents implements the master side of the bidirectional gRPC stream
// used by every device, including the master's own agent.
package agents

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/fleet"
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

// Service owns agent connections and master-side job state.
type Service struct {
	agentv1.UnimplementedAgentServer

	fleetSvc *fleet.Registry
	jobs     *JobStore
	version  string
	interval int

	mu          sync.RWMutex
	connections map[string]*connection

	// stopping holds jobs the user cancelled, so their result is recorded as
	// stopped rather than failed.
	stopMu   sync.Mutex
	stopping map[string]struct{}
}

func New(db *database.Database, fleetSvc *fleet.Registry, version string, interval int) (*Service, error) {
	jobs, err := NewJobStore(db)
	if err != nil {
		return nil, err
	}
	if interval <= 0 {
		interval = defaultHeartbeatInterval
	}
	return &Service{
		fleetSvc:    fleetSvc,
		jobs:        jobs,
		version:     version,
		interval:    interval,
		connections: make(map[string]*connection),
		stopping:    make(map[string]struct{}),
	}, nil
}

// Connect handles one agent stream. A reconnecting agent registers again and
// gets a new token; the old connection is hung up and marked replaced so its
// teardown does not mark the device offline. Jobs are not tied to a stream:
// registration reconciles what the agent still holds with what the master
// is waiting on.
func (svc *Service) Connect(stream agentv1.Agent_ConnectServer) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	registration := first.GetRegister()
	if registration == nil {
		return status.Error(codes.InvalidArgument, "first message must be a registration")
	}

	// Devices are keyed by name. A second machine using a name that is live
	// right now would otherwise take over the first one's identity and jobs,
	// and the two would then evict each other forever.
	if id, machine, found := svc.fleetSvc.Identity(registration.GetName()); found && svc.Connected(id) &&
		machine != "" && registration.GetMachineId() != "" && machine != registration.GetMachineId() {
		logger.Warn("Rejected agent %q: a different machine is already connected under that name", registration.GetName())
		return status.Error(codes.AlreadyExists, "another machine is already connected with this name")
	}

	stats := decodeStats(registration.GetStatsJson())
	registered, err := svc.fleetSvc.Register(fleet.RegisterRequest{
		JoinToken: registration.GetJoinToken(),
		Name:      registration.GetName(),
		URL:       agentURLLabel,
		Hostname:  registration.GetHostname(),
		Platform:  registration.GetPlatform(),
		Version:   registration.GetVersion(),
		DeployID:  registration.GetDeployId(),
		Master:    registration.GetMaster(),
		MachineID: registration.GetMachineId(),
		Stats:     stats,
	})
	if err != nil {
		return status.Error(codes.Unauthenticated, "invalid join token")
	}
	device, err := svc.fleetSvc.Get(registered.DeviceID)
	if err != nil {
		return status.Error(codes.Internal, "failed to load device")
	}

	conn := &connection{
		deviceID: device.ID,
		token:    registered.DeviceToken,
		stream:   stream,
		closed:   make(chan struct{}),
		pending:  make(map[string]chan *agentv1.AgentMessage),
		terms:    make(map[string]*TerminalOutput),
	}
	svc.mu.Lock()
	if previous := svc.connections[device.ID]; previous != nil {
		// Hang up the old stream; its teardown must not undo the new connection.
		previous.replaced.Store(true)
		previous.close()
	}
	svc.connections[device.ID] = conn
	svc.mu.Unlock()
	defer svc.disconnect(conn)

	resumes := svc.reconcileJobs(device.ID, registration.GetJobs())

	response := &agentv1.MasterMessage{
		RequestId: first.RequestId,
		Payload: &agentv1.MasterMessage_RegisterResponse{
			RegisterResponse: &agentv1.RegisterResponse{
				DeviceId:          device.ID,
				DeviceToken:       registered.DeviceToken,
				HeartbeatInterval: int32(svc.interval),
				MasterVersion:     svc.version,
				Jobs:              resumes,
			},
		},
	}
	if err := conn.send(response); err != nil {
		return err
	}

	logger.Info("Agent connected over gRPC: %s (%s)", device.Name, device.ID)
	svc.fleetSvc.Changed()

	// Recv blocks, so it runs on its own goroutine. Returning from this
	// handler is the only way to end the stream from the master side, and
	// doing so also unblocks that goroutine.
	incoming := make(chan *agentv1.AgentMessage)
	receiveErr := make(chan error, 1)
	go func() {
		for {
			message, err := stream.Recv()
			if err != nil {
				receiveErr <- err
				return
			}
			select {
			case incoming <- message:
			case <-conn.closed:
				return
			}
		}
	}()
	for {
		select {
		case <-conn.closed:
			return nil
		case err := <-receiveErr:
			if !errors.Is(err, context.Canceled) {
				logger.Debug("Agent stream %s closed: %v", device.ID, err)
			}
			return nil
		case message := <-incoming:
			svc.handle(conn, message)
		}
	}
}

func (svc *Service) handle(conn *connection, message *agentv1.AgentMessage) {
	switch payload := message.Payload.(type) {
	case *agentv1.AgentMessage_Heartbeat:
		stats := decodeStats(payload.Heartbeat.GetStatsJson())
		if _, err := svc.fleetSvc.Heartbeat(conn.token, fleet.HeartbeatRequest{Stats: stats}); err != nil {
			logger.Warn("Heartbeat from %s rejected: %v", conn.deviceID, err)
		}
	case *agentv1.AgentMessage_JobOutput:
		if err := svc.jobs.AppendLog(message.RequestId, payload.JobOutput.GetSeq(), payload.JobOutput.GetChunk(), payload.JobOutput.GetStderr()); err != nil {
			logger.Warn("Failed to store job output for %s: %v", message.RequestId, err)
		}
	case *agentv1.AgentMessage_JobResult:
		result := payload.JobResult
		svc.stopMu.Lock()
		_, stopped := svc.stopping[message.RequestId]
		delete(svc.stopping, message.RequestId)
		svc.stopMu.Unlock()
		if err := svc.jobs.Finish(message.RequestId, int(result.GetExitCode()), result.GetError(), stopped); err != nil {
			logger.Warn("Failed to finish job %s: %v", message.RequestId, err)
			break // no ack: the agent keeps the result and offers it again
		}
		// Tell the agent the result is stored so it can drop the job.
		_ = conn.send(&agentv1.MasterMessage{
			RequestId: message.RequestId,
			Payload:   &agentv1.MasterMessage_JobAck{JobAck: &agentv1.JobAck{}},
		})
	case *agentv1.AgentMessage_FileResult:
		conn.respond(message.RequestId, message)
	case *agentv1.AgentMessage_ProcessResult:
		conn.respond(message.RequestId, message)
	case *agentv1.AgentMessage_TerminalOutput:
		conn.terminalOutput(message.RequestId, payload.TerminalOutput)
	}
}

func (svc *Service) disconnect(conn *connection) {
	svc.mu.Lock()
	if svc.connections[conn.deviceID] == conn {
		delete(svc.connections, conn.deviceID)
	}
	svc.mu.Unlock()

	conn.mu.Lock()
	for id, channel := range conn.pending {
		close(channel)
		delete(conn.pending, id)
	}
	for id, terminal := range conn.terms {
		terminal.close()
		delete(conn.terms, id)
	}
	conn.mu.Unlock()

	if conn.replaced.Load() {
		logger.Info("Agent connection replaced: %s", conn.deviceID)
		return
	}
	// Running jobs are left alone: the agent keeps them going and reports
	// back when it reconnects.
	svc.fleetSvc.Changed()
	logger.Info("Agent disconnected: %s", conn.deviceID)
}

func (svc *Service) connection(deviceID string) (*connection, error) {
	svc.mu.RLock()
	conn := svc.connections[deviceID]
	svc.mu.RUnlock()
	if conn == nil {
		return nil, ErrNoAgent
	}
	return conn, nil
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

// reconcileJobs settles job state when an agent (re)connects. For each job
// the agent holds, the master says where to resume or to forget it. Jobs the
// master is waiting on that the agent does not hold are abandoned: the agent
// process that ran them is gone.
func (svc *Service) reconcileJobs(deviceID string, states []*agentv1.JobState) []*agentv1.JobResume {
	held := make(map[string]bool, len(states))
	resumes := make([]*agentv1.JobResume, 0, len(states))
	for _, state := range states {
		acked, active := svc.jobs.Resume(deviceID, state.GetId())
		if active {
			held[state.GetId()] = true
		}
		resumes = append(resumes, &agentv1.JobResume{Id: state.GetId(), AckedSeq: acked, Forget: !active})
	}
	if err := svc.jobs.AbandonMissing(deviceID, held, "agent restarted"); err != nil {
		logger.Warn("Failed to reconcile jobs for %s: %v", deviceID, err)
	}
	return resumes
}
