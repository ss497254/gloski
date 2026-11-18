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

func (s *Server) startAgentTransport(cfg *config.Config) {
	s.agentSock = filepath.Join(cfg.DataDir, "agent.sock")
	if err := os.MkdirAll(filepath.Dir(s.agentSock), 0700); err == nil {
		_ = os.Remove(s.agentSock)
	}

	if listener, err := net.Listen("unix", s.agentSock); err == nil {
		s.listeners = append(s.listeners, listener)
		go func() {
			if err := s.agentGRPC.Serve(listener); err != nil {
				logger.Warn("Local agent gRPC server stopped: %v", err)
			}
		}()
		logger.Info("Agent gRPC listening on unix://%s", s.agentSock)
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
		s.listeners = append(s.listeners, listener)
		go func() {
			if err := s.agentGRPC.Serve(listener); err != nil {
				logger.Warn("Remote agent gRPC server stopped: %v", err)
			}
		}()
		logger.Info("Agent gRPC listening on %s", address)
	} else {
		logger.Warn("Remote agent gRPC unavailable on %s: %v", address, err)
	}
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

	mu       sync.Mutex
	stopping bool
	cmd      *exec.Cmd
	done     chan struct{}
}

func (l *localAgent) run() {
	defer close(l.done)
	backoff := time.Second

	for {
		l.mu.Lock()
		if l.stopping {
			l.mu.Unlock()
			return
		}
		l.mu.Unlock()

		logFile, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			logger.Error("Failed to open local agent log: %v", err)
			return
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

		l.mu.Lock()
		l.cmd = command
		l.mu.Unlock()

		logger.Info("Starting local agent %s (%s)", l.name, l.binary)
		err = command.Run()
		_ = logFile.Close()

		l.mu.Lock()
		l.cmd = nil
		stopping := l.stopping
		l.mu.Unlock()
		if stopping {
			return
		}
		logger.Warn("Local agent exited (%v); restarting in %s", err, backoff)
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (l *localAgent) stop() {
	l.mu.Lock()
	l.stopping = true
	command := l.cmd
	l.mu.Unlock()
	if command != nil && command.Process != nil {
		_ = command.Process.Signal(syscall.SIGTERM)
	}
	select {
	case <-l.done:
	case <-time.After(5 * time.Second):
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
