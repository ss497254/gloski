// Package controltower serves the dashboard and browser API and owns the gRPC
// listeners that agents connect to. Fleet state lives in internal/fleet.
package controltower

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/provisioning"
	"github.com/ss497254/gloski/internal/tailnet"
	"github.com/ss497254/gloski/proto/agentv1"
	"google.golang.org/grpc"
)

// Tower owns the control tower state, HTTP API, and agent connections.
type Tower struct {
	cfg           *config.Config
	db            *database.Database
	fleetSvc      *fleet.Registry
	fleetEvents   *fleet.EventHub
	auth          *auth.Service
	installer     *provisioning.Installer
	tailnetClient *tailnet.Client
	router        http.Handler
	version       string
	name          string
	startedAt     time.Time
	webDir        string
	indexHTML     []byte
	stop          chan struct{}
	done          chan struct{}

	agentService      *agents.Service
	agentGRPCServer   *grpc.Server
	agentSocketPath   string
	listeners         []net.Listener
	localAgentProcess *localAgent
}

// New opens the database and wires the HTTP API, gRPC listeners, and local agent.
func New(cfg *config.Config, version string) (*Tower, error) {
	db, err := database.Open(cfg.MasterDBPath())
	if err != nil {
		return nil, fmt.Errorf("failed to open master database: %w", err)
	}

	fleetSvc, err := fleet.NewRegistry(db, cfg.Master, version, cfg.DataDir)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize device registry: %w", err)
	}

	authSvc, err := auth.NewService(cfg)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize auth service: %w", err)
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "gloski-tower"
	}

	agentService, err := agents.New(db, fleetSvc, version, cfg.Master.HeartbeatInterval)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize agent service: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(32<<20), grpc.MaxSendMsgSize(32<<20))
	agentv1.RegisterAgentServer(grpcServer, agentService)

	t := &Tower{
		cfg:             cfg,
		db:              db,
		fleetSvc:        fleetSvc,
		fleetEvents:     fleet.NewEventHub(),
		auth:            authSvc,
		installer:       provisioning.NewInstaller(resolveAgentBinary(cfg.Master.AgentBinary), cfg.Master.SSHKeyFile, tailnet.KnownHostsPath(cfg.DataDir)),
		tailnetClient:   tailnet.New(cfg.Tailscale.Socket),
		version:         version,
		name:            hostname,
		startedAt:       time.Now().UTC(),
		stop:            make(chan struct{}),
		done:            make(chan struct{}),
		agentService:    agentService,
		agentGRPCServer: grpcServer,
	}

	fleetSvc.SetOnChange(func() {
		list, err := fleetSvc.List()
		if err != nil {
			return
		}
		t.fleetEvents.Broadcast(fleet.Event{Type: "devices", Devices: list})
	})

	t.webDir = resolveWebDir(cfg.Master.WebDir)
	if t.webDir != "" {
		if data, err := os.ReadFile(filepath.Join(t.webDir, "index.html")); err == nil {
			t.indexHTML = data
		}
	}

	go t.fleetLoop()
	t.router = t.routes()
	t.startAgentTransport(cfg)
	t.startLocalAgent(cfg, version)
	logger.Info("Control tower initialized (db=%s, web=%s)", cfg.MasterDBPath(), displayPath(t.webDir))
	return t, nil
}

// Router returns the master HTTP handler.
func (t *Tower) Router() http.Handler {
	return t.router
}

// JoinToken returns the token agents use to enroll.
func (t *Tower) JoinToken() string {
	return t.fleetSvc.JoinToken()
}

// Close releases master resources.
func (t *Tower) Close() error {
	close(t.stop)
	<-t.done
	if t.localAgentProcess != nil {
		t.localAgentProcess.stop()
	}
	if t.agentGRPCServer != nil {
		t.agentGRPCServer.GracefulStop()
	}
	for _, listener := range t.listeners {
		_ = listener.Close()
	}
	if t.agentSocketPath != "" {
		_ = os.Remove(t.agentSocketPath)
	}
	return t.db.Close()
}

// fleetLoop keeps online/offline status fresh for dashboards. Heartbeats also
// trigger broadcasts, but a quiet or dead agent would otherwise leave the
// fleet showing a stale status indefinitely.
