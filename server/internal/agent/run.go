// Stateless agent lifecycle: connect, register, send heartbeats, and dispatch commands from the master.

package agent

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
)

// Run starts a stateless device agent. It keeps nothing durable and
// reconnects to the master whenever the stream drops.
func Run(cfg *config.Config, version string) error {
	if cfg.Agent.MasterURL == "" {
		return fmt.Errorf("agent.master_url is required (set GLOSKI_AGENT_MASTER_URL)")
	}
	if cfg.Mode != config.ModeAgent {
		return fmt.Errorf("gloski-agent must run in agent mode (current mode: %s)", cfg.Mode)
	}
	if strings.TrimSpace(cfg.Agent.JoinToken) == "" {
		return fmt.Errorf("agent.join_token is required (set GLOSKI_AGENT_JOIN_TOKEN)")
	}

	name := strings.TrimSpace(cfg.Agent.Name)
	if name == "" {
		name, _ = os.Hostname()
	}
	if name == "" {
		name = "gloski-agent"
	}

	statsStore := system.NewStore(300)
	statsHub := system.NewHub(statsStore)
	collector := system.NewCollector(statsStore, statsHub, 2*time.Second)
	collector.Start()
	defer collector.Stop()
	statsService := system.NewService(statsStore, statsHub)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	backoff := time.Second
	for {
		select {
		case <-quit:
			logger.Info("Agent stopped")
			return nil
		default:
		}

		if err := runSession(cfg, version, name, statsService, quit); err != nil {
			logger.Warn("Agent session ended: %v", err)
		}

		logger.Info("Agent will reconnect in %s", backoff)
		select {
		case <-quit:
			logger.Info("Agent stopped")
			return nil
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// runSession owns one gRPC connection until it drops or the agent is asked to stop.
func runSession(cfg *config.Config, version, name string, statsService *system.Service, quit <-chan os.Signal) error {
	logger.Info("Agent connecting to master at %s", cfg.Agent.MasterURL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, err := dialMaster(ctx, cfg.Agent.MasterURL)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to connect to master: %w", err)
	}
	defer conn.Close()

	stream, err := agentv1.NewAgentClient(conn).Connect(context.Background())
	if err != nil {
		return fmt.Errorf("failed to open agent stream: %w", err)
	}

	agent := &agentRuntime{
		cfg:     cfg,
		version: version,
		name:    name,
		stream:  stream,
		stats:   statsService,
		files:   files.NewService(cfg),
		jobs:    make(map[string]context.CancelFunc),
		terms:   make(map[string]*ptySession),
	}
	defer agent.cleanup()
	defer func() { _ = stream.CloseSend() }()

	hostname, _ := os.Hostname()
	stats, _ := statsService.GetStats()
	statsJSON, _ := json.Marshal(stats)

	requestID := strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := agent.send(&agentv1.AgentMessage{
		RequestId: requestID,
		Payload: &agentv1.AgentMessage_Register{
			Register: &agentv1.RegisterRequest{
				JoinToken: cfg.Agent.JoinToken,
				Name:      name,
				Hostname:  hostname,
				Platform:  runtime.GOOS + "/" + runtime.GOARCH,
				Version:   version,
				DeployId:  cfg.Agent.DeployID,
				Master:    cfg.Agent.Master,
				StatsJson: string(statsJSON),
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
	agent.deviceID = registration.GetDeviceId()
	interval := int(registration.GetHeartbeatInterval())
	if interval <= 0 {
		interval = cfg.Agent.HeartbeatInterval
	}
	if interval <= 0 {
		interval = 10
	}
	logger.Info("Agent %s (%s) registered with master", name, agent.deviceID)

	sessionDone := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(sessionDone) }) }
	go agent.heartbeatLoop(sessionDone, interval, finish)
	go agent.receiveLoop(sessionDone, finish)

	select {
	case <-quit:
		return nil
	case <-sessionDone:
		return errors.New("master stream closed")
	}
}

type agentRuntime struct {
	cfg      *config.Config
	version  string
	name     string
	stream   agentv1.Agent_ConnectClient
	stats    *system.Service
	files    *files.Service
	deviceID string

	sendMu sync.Mutex
	mu     sync.Mutex
	jobs   map[string]context.CancelFunc
	terms  map[string]*ptySession
}

func (a *agentRuntime) send(message *agentv1.AgentMessage) error {
	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	return a.stream.Send(message)
}

func (a *agentRuntime) cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, cancel := range a.jobs {
		cancel()
		delete(a.jobs, id)
	}
	for id, session := range a.terms {
		session.close()
		delete(a.terms, id)
	}
}

func (a *agentRuntime) heartbeatLoop(done <-chan struct{}, interval int, finish func()) {
	defer finish()
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			stats, err := a.stats.GetStats()
			if err != nil {
				continue
			}
			raw, err := json.Marshal(stats)
			if err != nil {
				continue
			}
			if err := a.send(&agentv1.AgentMessage{
				Payload: &agentv1.AgentMessage_Heartbeat{
					Heartbeat: &agentv1.Heartbeat{StatsJson: string(raw)},
				},
			}); err != nil {
				return
			}
		}
	}
}

func (a *agentRuntime) receiveLoop(done <-chan struct{}, finish func()) {
	defer finish()
	for {
		message, err := a.stream.Recv()
		if err != nil {
			select {
			case <-done:
			default:
				logger.Warn("Agent stream closed: %v", err)
			}
			return
		}
		a.handle(message)
	}
}

func (a *agentRuntime) handle(message *agentv1.MasterMessage) {
	switch payload := message.Payload.(type) {
	case *agentv1.MasterMessage_RunJob:
		go a.runJob(message.RequestId, payload.RunJob)
	case *agentv1.MasterMessage_Cancel:
		a.mu.Lock()
		cancel := a.jobs[message.RequestId]
		a.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	case *agentv1.MasterMessage_FileRequest:
		go a.handleFile(message.RequestId, payload.FileRequest)
	case *agentv1.MasterMessage_ProcessRequest:
		go a.handleProcesses(message.RequestId, payload.ProcessRequest)
	case *agentv1.MasterMessage_TerminalOpen:
		go a.openTerminal(message.RequestId, payload.TerminalOpen)
	case *agentv1.MasterMessage_TerminalInput:
		a.terminalInput(message.RequestId, payload.TerminalInput.GetData())
	case *agentv1.MasterMessage_TerminalClose:
		a.closeTerminal(message.RequestId)
	case *agentv1.MasterMessage_Shutdown:
		logger.Info("Master requested agent shutdown")
		os.Exit(0)
	}
}
