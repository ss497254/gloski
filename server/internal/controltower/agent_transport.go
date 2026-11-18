package controltower

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/tailnet"
)

const (
	agentHealthInterval = 5 * time.Second
	agentUnhealthyLimit = 20 * time.Second
	agentMaxBackoff     = 30 * time.Second
	agentShutdownWait   = 3 * time.Second
	agentKillWait       = 2 * time.Second
	agentFinalWait      = 5 * time.Second
)

func (t *Tower) startAgentTransport(cfg *config.Config) {
	t.agentSocketPath = filepath.Join(cfg.DataDir, "agent.sock")
	if err := os.MkdirAll(filepath.Dir(t.agentSocketPath), 0700); err == nil {
		_ = os.Remove(t.agentSocketPath)
	}

	if listener, err := net.Listen("unix", t.agentSocketPath); err == nil {
		t.serveGRPC(listener, "unix://"+t.agentSocketPath)
	} else {
		logger.Error("Failed to listen on agent socket %s: %v", t.agentSocketPath, err)
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
		t.serveGRPC(listener, address)
	} else {
		logger.Warn("Remote agent gRPC unavailable on %s: %v", address, err)
	}
}

func (t *Tower) serveGRPC(listener net.Listener, label string) {
	t.listeners = append(t.listeners, listener)
	go func() {
		if err := t.agentGRPCServer.Serve(listener); err != nil {
			logger.Warn("Agent gRPC server %s stopped: %v", label, err)
		}
	}()
	logger.Info("Agent gRPC listening on %s", label)
}

func (t *Tower) agentGRPCURL() string {
	if t.cfg.Master.AgentGRPCURL != "" {
		return strings.TrimRight(t.cfg.Master.AgentGRPCURL, "/")
	}
	host := t.tailnetAgentHost()
	if host == "" {
		host = t.cfg.Master.AgentGRPCHost
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	port := t.cfg.Master.AgentGRPCPort
	if port <= 0 {
		port = 9090
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(host, strconv.Itoa(port)))
}

func (t *Tower) tailnetAgentHost() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	self, err := t.tailnetClient.Self(ctx)
	if err != nil {
		return ""
	}
	if self.DNSName != "" {
		return self.DNSName
	}
	return tailnet.PrimaryIP(self.IPs)
}
