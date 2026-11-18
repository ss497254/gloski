package master

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/tailscale"
)

const (
	agentHealthInterval = 5 * time.Second
	agentUnhealthyLimit = 20 * time.Second
	agentMaxBackoff     = 30 * time.Second
	agentShutdownWait   = 3 * time.Second
	agentKillWait       = 2 * time.Second
	agentFinalWait      = 5 * time.Second
)

func (s *Server) startAgentTransport(cfg *config.Config) {
	s.agentSock = filepath.Join(cfg.DataDir, "agent.sock")
	if err := os.MkdirAll(filepath.Dir(s.agentSock), 0700); err == nil {
		_ = os.Remove(s.agentSock)
	}

	if listener, err := net.Listen("unix", s.agentSock); err == nil {
		s.serveGRPC(listener, "unix://"+s.agentSock)
	} else {
		logger.Error("Failed to listen on agent socket %s: %v", s.agentSock, err)
	}

	host := cfg.Master.AgentGRPCHost
	if host == "" {
		host = "0.0.0.0"
	}
	port := cfg.Master.AgentGRPCPort
	if port <= 0 {
		port = 9090
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	if listener, err := net.Listen("tcp", address); err == nil {
		s.serveGRPC(listener, address)
	} else {
		logger.Warn("Remote agent gRPC unavailable on %s: %v", address, err)
	}
}

func (s *Server) serveGRPC(listener net.Listener, label string) {
	s.listeners = append(s.listeners, listener)
	go func() {
		if err := s.agentGRPC.Serve(listener); err != nil {
			logger.Warn("Agent gRPC server %s stopped: %v", label, err)
		}
	}()
	logger.Info("Agent gRPC listening on %s", label)
}

func (s *Server) startLocalAgent(cfg *config.Config, version string) {
	if !cfg.Master.LocalAgent || s.agentSock == "" || s.deployer.Binary() == "" {
		return
	}
	name := cfg.Master.LocalAgentName
	if name == "" {
		name = s.name
	}
	agent := &localAgent{
		binary:    s.deployer.Binary(),
		masterURL: "unix://" + s.agentSock,
		joinToken: s.devices.JoinToken(),
		name:      name,
		version:   version,
		shell:     cfg.Shell,
		dataDir:   filepath.Join(cfg.DataDir, "local-agent"),
		logPath:   filepath.Join(cfg.DataDir, "local-agent.log"),
		done:      make(chan struct{}),
		stopCh:    make(chan struct{}),
		health: func() bool {
			list, err := s.devices.List()
			if err != nil {
				logger.Warn("local agent health check failed: %v", err)
				return false
			}
			for i := range list {
				if list[i].Master && list[i].Name == name {
					return s.agents.Connected(list[i].ID)
				}
			}
			return false
		},
	}
	s.local = agent
	go agent.run()
}

type localAgent struct {
	binary    string
	masterURL string
	joinToken string
	name      string
	version   string
	shell     string
	dataDir   string
	logPath   string
	health    func() bool

	mu       sync.Mutex
	stopping bool
	cmd      *exec.Cmd
	done     chan struct{}
	stopCh   chan struct{}
	stopOnce sync.Once
}

func (l *localAgent) run() {
	defer close(l.done)
	backoff := time.Second

	for {
		if l.isStopping() {
			return
		}

		logFile, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			logger.Error("local agent: failed to open log %s: %v", l.logPath, err)
			l.wait(backoff)
			continue
		}

		command := exec.Command(l.binary)
		command.Env = append(os.Environ(),
			"GLOSKI_MODE=agent",
			"GLOSKI_AGENT_MASTER_URL="+l.masterURL,
			"GLOSKI_AGENT_JOIN_TOKEN="+l.joinToken,
			"GLOSKI_AGENT_NAME="+l.name,
			"GLOSKI_AGENT_IS_MASTER=true",
			"GLOSKI_DATA_DIR="+l.dataDir,
			"GLOSKI_SHELL="+l.shell,
		)
		command.Stdout = logFile
		command.Stderr = logFile
		command.Stdin = nil

		logger.Info("local agent: starting %s (binary=%s)", l.name, l.binary)
		if err := command.Start(); err != nil {
			logger.Error("local agent: failed to start: %v", err)
			_ = logFile.Close()
			l.wait(backoff)
			continue
		}
		pid := command.Process.Pid
		logger.Info("local agent: started pid=%d", pid)

		l.mu.Lock()
		l.cmd = command
		l.mu.Unlock()

		exited := make(chan error, 1)
		go func() { exited <- command.Wait() }()

		ticker := time.NewTicker(agentHealthInterval)
		unhealthySince := time.Time{}
		healthy := false

	healthLoop:
		for {
			select {
			case <-l.stopCh:
				logger.Info("local agent: stop requested for pid=%d", pid)
				ticker.Stop()
				l.terminate(command, exited)
				break healthLoop
			case err := <-exited:
				ticker.Stop()
				logger.Warn("local agent: process pid=%d exited: %v", pid, err)
				break healthLoop
			case <-ticker.C:
				connected := l.health != nil && l.health()
				if connected {
					if !healthy {
						logger.Info("local agent: gRPC session is live (pid=%d)", pid)
					}
					healthy = true
					unhealthySince = time.Time{}
					continue
				}
				if healthy {
					logger.Warn("local agent: gRPC session lost (pid=%d)", pid)
				}
				healthy = false
				if unhealthySince.IsZero() {
					unhealthySince = time.Now()
					logger.Warn("local agent: no gRPC session for pid=%d, waiting before restart", pid)
					continue
				}
				if time.Since(unhealthySince) >= agentUnhealthyLimit {
					logger.Warn("local agent: no gRPC session for 20s, restarting pid=%d", pid)
					l.terminate(command, exited)
					ticker.Stop()
					break healthLoop
				}
			}
		}

		_ = logFile.Close()
		l.mu.Lock()
		l.cmd = nil
		l.mu.Unlock()
		if l.isStopping() {
			return
		}
		logger.Info("local agent: restart in %s", backoff)
		l.wait(backoff)
		if backoff < agentMaxBackoff {
			backoff *= 2
		}
	}
}

func (l *localAgent) isStopping() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stopping
}

func (l *localAgent) wait(duration time.Duration) {
	select {
	case <-l.stopCh:
	case <-time.After(duration):
	}
}

func (l *localAgent) terminate(command *exec.Cmd, exited <-chan error) {
	if command.Process == nil {
		return
	}
	_ = command.Process.Signal(syscall.SIGTERM)
	select {
	case <-exited:
	case <-time.After(agentShutdownWait):
		_ = command.Process.Kill()
		select {
		case <-exited:
		case <-time.After(agentKillWait):
		}
	}
}

func (l *localAgent) stop() {
	l.stopOnce.Do(func() { close(l.stopCh) })
	l.mu.Lock()
	l.stopping = true
	command := l.cmd
	l.mu.Unlock()
	if command != nil && command.Process != nil {
		_ = command.Process.Signal(syscall.SIGTERM)
	}
	select {
	case <-l.done:
	case <-time.After(agentFinalWait):
	}
}

func (s *Server) agentGRPCURL() string {
	if s.cfg.Master.AgentGRPCURL != "" {
		return strings.TrimRight(s.cfg.Master.AgentGRPCURL, "/")
	}
	host := s.tailnetAgentHost()
	if host == "" {
		host = s.cfg.Master.AgentGRPCHost
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	port := s.cfg.Master.AgentGRPCPort
	if port <= 0 {
		port = 9090
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(host, strconv.Itoa(port)))
}

func (s *Server) tailnetAgentHost() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	self, err := s.tailscale.Self(ctx)
	if err != nil {
		return ""
	}
	if self.DNSName != "" {
		return self.DNSName
	}
	return tailscale.PrimaryIP(self.IPs)
}
