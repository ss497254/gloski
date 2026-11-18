package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ss497254/gloski/internal/api"
	"github.com/ss497254/gloski/internal/app"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/devices"
	"github.com/ss497254/gloski/internal/logger"
)

// Run starts the device agent and blocks until it receives SIGINT/SIGTERM.
func Run(cfg *config.Config, version string) error {
	if cfg.Agent.MasterURL == "" {
		return fmt.Errorf("agent.master_url is required (set GLOSKI_AGENT_MASTER_URL)")
	}
	if cfg.Mode != config.ModeAgent {
		return fmt.Errorf("gloski-agent must run in agent mode (current mode: %s)", cfg.Mode)
	}

	creds, err := loadCredentials(cfg.AgentStatePath())
	if err != nil {
		logger.Warn("Failed to read agent state, re-enrolling: %v", err)
		creds = nil
	}

	name := strings.TrimSpace(cfg.Agent.Name)
	if name == "" {
		name, _ = os.Hostname()
	}
	if name == "" {
		name = "gloski-agent"
	}

	if creds == nil || creds.MasterURL != strings.TrimRight(cfg.Agent.MasterURL, "/") {
		creds, err = enroll(cfg, name, version)
		if err != nil {
			return err
		}
	}

	logger.Info("Agent %s (%s) enrolled with %s", name, creds.DeviceID, creds.MasterURL)

	// The master authenticates to this agent with the token it issued.
	cfg.APIKey = creds.DeviceToken

	application, err := app.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize agent services: %w", err)
	}
	defer application.Shutdown(context.Background())

	apiServer := api.NewServer(application, api.ServerOptions{Version: version})

	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           apiServer.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("Agent API listening on http://%s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Agent API failed to start: %v", err)
		}
	}()

	interval := creds.HeartbeatInterval
	if interval <= 0 {
		interval = cfg.Agent.HeartbeatInterval
	}
	if interval <= 0 {
		interval = 10
	}

	stopHeartbeat := make(chan struct{})
	heartbeatDone := make(chan struct{})
	go heartbeatLoop(cfg, application, creds, interval, version, stopHeartbeat, heartbeatDone)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("Received signal %v, shutting down agent...", sig)

	close(stopHeartbeat)
	select {
	case <-heartbeatDone:
	case <-time.After(3 * time.Second):
	}

	timeout := cfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = 5
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Agent HTTP server forced to shut down: %v", err)
	}
	apiServer.Shutdown()
	if err := application.Shutdown(ctx); err != nil {
		logger.Error("Agent service shutdown error: %v", err)
	}

	logger.Info("Agent stopped")
	return nil
}

func enroll(cfg *config.Config, name, version string) (*Credentials, error) {
	if cfg.Agent.JoinToken == "" {
		return nil, fmt.Errorf("agent is not enrolled and agent.join_token is empty (set GLOSKI_AGENT_JOIN_TOKEN)")
	}

	hostname, _ := os.Hostname()
	req := devices.RegisterRequest{
		JoinToken: cfg.Agent.JoinToken,
		Name:      name,
		URL:       advertiseURL(cfg),
		Port:      cfg.Port,
		Hostname:  hostname,
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		Version:   version,
	}

	client := NewClient(cfg.Agent.MasterURL)
	resp, err := client.Register(req)
	if err != nil {
		return nil, fmt.Errorf("enrollment failed: %w", err)
	}

	creds := &Credentials{
		DeviceID:          resp.DeviceID,
		DeviceToken:       resp.DeviceToken,
		MasterURL:         strings.TrimRight(cfg.Agent.MasterURL, "/"),
		Name:              name,
		HeartbeatInterval: resp.HeartbeatInterval,
	}
	if err := saveCredentials(cfg.AgentStatePath(), creds); err != nil {
		return nil, fmt.Errorf("failed to save agent credentials: %w", err)
	}
	logger.Info("Enrollment complete; credentials saved to %s", cfg.AgentStatePath())
	return creds, nil
}

func heartbeatLoop(cfg *config.Config, application *app.App, creds *Credentials, interval int, version string, stop, done chan struct{}) {
	defer close(done)

	client := NewClient(cfg.Agent.MasterURL)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	send := func() {
		latest, err := application.System.GetStats()
		if err != nil {
			logger.Debug("Failed to collect stats for heartbeat: %v", err)
		}
		req := devices.HeartbeatRequest{
			URL:     advertiseURL(cfg),
			Version: version,
			Stats:   latest,
		}
		if err := client.Heartbeat(creds.DeviceToken, req); err != nil {
			logger.Warn("Heartbeat failed: %v", err)
		}
	}

	// Report immediately, then on the advertised interval.
	send()
	for {
		select {
		case <-ticker.C:
			send()
		case <-stop:
			return
		}
	}
}

func advertiseURL(cfg *config.Config) string {
	if cfg.Agent.AdvertiseURL != "" {
		return strings.TrimRight(cfg.Agent.AdvertiseURL, "/")
	}

	host := cfg.Host
	switch host {
	case "", "127.0.0.1", "localhost", "::1":
		// The master fills in the observed address. An agent bound only to
		// loopback is reachable solely on this machine.
		return ""
	case "0.0.0.0", "::":
		// A wildcard bind can be reached on any interface, but the master
		// may see the connection through a local proxy (tailscale serve).
		// Advertise the tailnet address explicitly when we can find it.
		host = preferredAdvertiseIP()
		if host == "" {
			return ""
		}
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(cfg.Port))
}

// preferredAdvertiseIP picks a stable address for the master to call back on.
// Tailscale addresses win over other interfaces because the control tower is
// expected to operate over the tailnet.
func preferredAdvertiseIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	fallback := ""
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
				continue
			}
			if isTailscaleIP(ip) {
				return ip.String()
			}
			if fallback == "" {
				fallback = ip.String()
			}
		}
	}
	return fallback
}

func isTailscaleIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		return v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
	}
	// Tailscale's IPv6 range is fd7a:115c:a1e0::/48.
	return len(ip) == net.IPv6len &&
		ip[0] == 0xfd && ip[1] == 0x7a && ip[2] == 0x11 &&
		ip[3] == 0x5c && ip[4] == 0xa1 && ip[5] == 0xe0
}
