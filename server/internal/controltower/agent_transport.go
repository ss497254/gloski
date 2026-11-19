package controltower

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
	agentTailnetRetry   = 15 * time.Second
)

func (t *Tower) startAgentTransport(cfg *config.Config) {
	t.agentSocketPath = filepath.Join(cfg.DataDir, "agent.sock")
	if err := os.MkdirAll(filepath.Dir(t.agentSocketPath), 0700); err == nil {
		_ = os.Remove(t.agentSocketPath)
	}

	if listener, err := net.Listen("unix", t.agentSocketPath); err == nil {
		// Only the master's own user may reach the local agent socket.
		_ = os.Chmod(t.agentSocketPath, 0600)
		// The socket carries gRPC and the HTTP file transfers the local agent
		// makes, just like the TCP agent port.
		grpcSide, httpSide := splitListener(listener)
		t.serveGRPC(grpcSide, "unix://"+t.agentSocketPath)
		t.serveAgentHTTP(httpSide)
	} else {
		logger.Error("Failed to listen on agent socket %s: %v", t.agentSocketPath, err)
	}

	port := cfg.Master.AgentGRPCPort
	if port <= 0 {
		port = 9090
	}

	// An explicit host is honoured as-is. Otherwise the stream is plaintext
	// h2c, so it is only offered on this machine's Tailscale addresses.
	if host := strings.TrimSpace(cfg.Master.AgentGRPCHost); host != "" {
		t.listenAgentTCP(host, port)
		return
	}
	if t.listenAgentTailnet(port) {
		return
	}
	logger.Warn("Tailscale is not up; remote agents cannot connect until it is (set GLOSKI_MASTER_AGENT_GRPC_HOST to bind an address explicitly)")
	go func() {
		ticker := time.NewTicker(agentTailnetRetry)
		defer ticker.Stop()
		for {
			select {
			case <-t.stop:
				return
			case <-ticker.C:
				if t.listenAgentTailnet(port) {
					return
				}
			}
		}
	}()
}

// listenAgentTailnet binds the remote agent port on every Tailscale address
// of this machine. It reports whether at least one listener started.
func (t *Tower) listenAgentTailnet(port int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	self, err := t.tailnetClient.Self(ctx)
	if err != nil || len(self.IPs) == 0 {
		return false
	}
	started := false
	for _, ip := range self.IPs {
		if t.listenAgentTCP(ip, port) {
			started = true
		}
	}
	return started
}

func (t *Tower) listenAgentTCP(host string, port int) bool {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Warn("Remote agent gRPC unavailable on %s: %v", address, err)
		return false
	}
	grpcSide, httpSide := splitListener(listener)
	t.serveGRPC(grpcSide, address)
	t.serveAgentHTTP(httpSide)
	return true
}

// serveAgentHTTP serves plain HTTP on an agent listener. Agents use it for the
// install script and to move file contents.
func (t *Tower) serveAgentHTTP(listener net.Listener) {
	go func() {
		server := &http.Server{Handler: t.agentHTTPHandler(), ReadHeaderTimeout: 10 * time.Second}
		_ = server.Serve(listener)
	}()
}

func (t *Tower) serveGRPC(listener net.Listener, label string) {
	t.listenersMu.Lock()
	t.listeners = append(t.listeners, listener)
	t.listenersMu.Unlock()
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
