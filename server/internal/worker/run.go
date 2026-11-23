// Package worker implements gloski-agent, the device-side half of the gRPC
// stream: registration, heartbeats, jobs, files and terminal sessions.
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/system"
	"github.com/ss497254/gloski/proto/agentv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxReconnectBackoff = 30 * time.Second
	// A session that lasted this long was healthy, so the next reconnect
	// starts from the shortest delay again.
	stableSession = time.Minute
)

var errDisconnected = errors.New("not connected to master")

// Run starts a device worker. It keeps nothing on disk and reconnects to the
// master whenever the stream drops; jobs keep running across reconnects.
func Run(cfg *config.Config, version string) error {
	if cfg.Agent.MasterURL == "" {
		return fmt.Errorf("worker.master_url is required (set GLOSKI_AGENT_MASTER_URL)")
	}
	if cfg.Mode != config.ModeAgent {
		return fmt.Errorf("gloski-agent must run in agent mode (current mode: %s)", cfg.Mode)
	}
	if strings.TrimSpace(cfg.Agent.JoinToken) == "" {
		return fmt.Errorf("worker.join_token is required (set GLOSKI_AGENT_JOIN_TOKEN)")
	}

	name := strings.TrimSpace(cfg.Agent.Name)
	if name == "" {
		name, _ = os.Hostname()
	}
	if name == "" {
		name = "gloski-agent"
	}

	statsStore := system.NewStore(300)
	// The agent serves no WebSocket clients. A hub with no Run loop would only
	// fill its channel and warn on every sample, so the collector keeps the
	// latest stats in the store without broadcasting them.
	collector := system.NewCollector(statsStore, nil, 2*time.Second)
	collector.Start()
	defer collector.Stop()

	// One context ends the agent, whether from a signal or a Shutdown command
	// from the master. A context cannot be "used up" the way a signal read
	// from a channel can, so every loop below sees the stop.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, shutdown := context.WithCancel(ctx)
	defer shutdown()

	worker := &Runtime{
		cfg:       cfg,
		version:   version,
		name:      name,
		machineID: machineID(),
		buildHash: buildHash(),
		stats:     system.NewService(statsStore, nil),
		files:     files.NewService(cfg),
		shutdown:  shutdown,
		jobs:      make(map[string]*job),
		terms:     make(map[string]*PTYSession),
	}
	// Jobs outlive connections but not the agent process.
	defer worker.stopJobs()

	backoff := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		if err := worker.connectOnce(ctx); err != nil && ctx.Err() == nil {
			if status.Code(err) == codes.FailedPrecondition {
				// The master will not work with this build. Retrying cannot
				// help; exit and let the master install its own build.
				return fmt.Errorf("master refused this agent: %w", err)
			}
			logger.Warn("Agent session ended: %v", err)
		}
		if ctx.Err() != nil {
			break
		}
		if time.Since(started) >= stableSession {
			backoff = time.Second
		}

		logger.Info("Agent will reconnect in %s", backoff)
		select {
		case <-ctx.Done():
		case <-time.After(backoff):
		}
		if backoff < maxReconnectBackoff {
			backoff *= 2
		}
	}
	logger.Info("Agent stopped")
	return nil
}

// Runtime is the device-side executor. It lives for the whole agent process;
// the stream to the master comes and goes underneath it.
type Runtime struct {
	cfg       *config.Config
	version   string
	name      string
	machineID string
	buildHash string
	stats     *system.Service
	files     *files.Service
	deviceID  string
	shutdown  context.CancelFunc // stops the whole agent

	// sendMu serializes sends (gRPC streams do not allow concurrent SendMsg)
	// and guards stream, which is nil while disconnected.
	sendMu sync.Mutex
	stream agentv1.Agent_ConnectClient

	mu    sync.Mutex
	jobs  map[string]*job
	terms map[string]*PTYSession
}

func (w *Runtime) connectOnce(ctx context.Context) error {
	logger.Info("Agent connecting to master at %s", w.cfg.Agent.MasterURL)
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	conn, err := dialMaster(dialCtx, w.cfg.Agent.MasterURL)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to connect to master: %w", err)
	}
	defer conn.Close()

	// The stream gets its own context so leaving this function always ends it.
	streamCtx, endStream := context.WithCancel(ctx)
	defer endStream()
	stream, err := agentv1.NewAgentClient(conn).Connect(streamCtx)
	if err != nil {
		return fmt.Errorf("failed to open agent stream: %w", err)
	}

	hostname, _ := os.Hostname()
	stats, _ := w.stats.GetStats()
	statsJSON, _ := json.Marshal(stats)

	// Registration goes out directly: the stream is not attached yet, so no
	// job output can slip in ahead of it.
	requestID := strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := stream.Send(&agentv1.AgentMessage{
		RequestId: requestID,
		Payload: &agentv1.AgentMessage_Register{
			Register: &agentv1.RegisterRequest{
				JoinToken: w.cfg.Agent.JoinToken,
				Name:      w.name,
				Hostname:  hostname,
				Platform:  runtime.GOOS + "/" + runtime.GOARCH,
				Version:   w.version,
				DeployId:  w.cfg.Agent.DeployID,
				Master:    w.cfg.Agent.Master,
				StatsJson: string(statsJSON),
				Jobs:      w.jobStates(),
				MachineId: w.machineID,
				Arch:      runtime.GOARCH,
				BuildHash: w.buildHash,
				Protocol:  agentv1.ProtocolVersion,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	response, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	registration := response.GetRegisterResponse()
	if registration == nil {
		return errors.New("master did not return a registration response")
	}
	w.deviceID = registration.GetDeviceId()
	interval := int(registration.GetHeartbeatInterval())
	if interval <= 0 {
		interval = w.cfg.Agent.HeartbeatInterval
	}
	if interval <= 0 {
		interval = 10
	}
	logger.Info("Agent %s (%s) registered with master", w.name, w.deviceID)

	// Order matters: rewind each job to what the master has, then attach the
	// stream, then replay. Attaching first would let live output go out with
	// a stale position and be mistaken for a gap.
	w.resumeJobs(registration.GetJobs())
	w.attach(stream)
	defer w.detach()
	w.flushJobs()

	sessionDone := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(sessionDone) }) }
	go w.heartbeatLoop(sessionDone, interval, finish)
	go w.receiveLoop(stream, sessionDone, finish)

	select {
	case <-ctx.Done():
		return nil
	case <-sessionDone:
		return errors.New("master stream closed")
	}
}

func (w *Runtime) attach(stream agentv1.Agent_ConnectClient) {
	w.sendMu.Lock()
	w.stream = stream
	w.sendMu.Unlock()
}

// detach drops the stream and everything that cannot outlive it. Terminals
// are interactive and end with the session; jobs carry on.
func (w *Runtime) detach() {
	w.sendMu.Lock()
	stream := w.stream
	w.stream = nil
	w.sendMu.Unlock()
	if stream != nil {
		_ = stream.CloseSend()
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	for id, session := range w.terms {
		session.close()
		delete(w.terms, id)
	}
}

func (w *Runtime) send(message *agentv1.AgentMessage) error {
	w.sendMu.Lock()
	defer w.sendMu.Unlock()
	if w.stream == nil {
		return errDisconnected
	}
	return w.stream.Send(message)
}

func (w *Runtime) heartbeatLoop(done <-chan struct{}, interval int, finish func()) {
	defer finish()
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			stats, err := w.stats.GetStats()
			if err != nil {
				continue
			}
			raw, err := json.Marshal(stats)
			if err != nil {
				continue
			}
			if err := w.send(&agentv1.AgentMessage{
				Payload: &agentv1.AgentMessage_Heartbeat{
					Heartbeat: &agentv1.Heartbeat{StatsJson: string(raw)},
				},
			}); err != nil {
				return
			}
		}
	}
}

func (w *Runtime) receiveLoop(stream agentv1.Agent_ConnectClient, done <-chan struct{}, finish func()) {
	defer finish()
	for {
		message, err := stream.Recv()
		if err != nil {
			select {
			case <-done:
			default:
				logger.Warn("Agent stream closed: %v", err)
			}
			return
		}
		w.handle(message)
	}
}

func (w *Runtime) handle(message *agentv1.MasterMessage) {
	switch payload := message.Payload.(type) {
	case *agentv1.MasterMessage_RunJob:
		w.startJob(message.RequestId, payload.RunJob)
	case *agentv1.MasterMessage_Cancel:
		w.cancelJob(message.RequestId)
	case *agentv1.MasterMessage_JobAck:
		w.forgetJob(message.RequestId)
	case *agentv1.MasterMessage_FileRequest:
		go w.handleFile(message.RequestId, payload.FileRequest)
	case *agentv1.MasterMessage_FileTransfer:
		go w.handleTransfer(message.RequestId, payload.FileTransfer)
	case *agentv1.MasterMessage_ProcessRequest:
		go w.handleProcesses(message.RequestId, payload.ProcessRequest)
	case *agentv1.MasterMessage_TerminalOpen:
		w.openTerminal(message.RequestId, payload.TerminalOpen)
	case *agentv1.MasterMessage_TerminalInput:
		w.terminalInput(message.RequestId, payload.TerminalInput.GetData())
	case *agentv1.MasterMessage_TerminalResize:
		w.resizeTerminal(message.RequestId, payload.TerminalResize.GetCols(), payload.TerminalResize.GetRows())
	case *agentv1.MasterMessage_TerminalClose:
		w.closeTerminal(message.RequestId)
	case *agentv1.MasterMessage_Shutdown:
		// Cancel rather than exit so running jobs and terminals are cleaned up.
		logger.Info("Master requested agent shutdown (uninstall=%v)", payload.Shutdown.GetUninstall())
		if payload.Shutdown.GetUninstall() {
			removeInstall()
		}
		w.shutdown()
	}
}
