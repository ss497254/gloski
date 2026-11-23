// Package controltower serves the dashboard and browser API and owns the gRPC
// listeners that agents connect to. Fleet state lives in internal/fleet.
package controltower

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ss497254/gloski/internal/actions"
	"github.com/ss497254/gloski/internal/agentdist"
	"github.com/ss497254/gloski/internal/agentdist/builds"
	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/metrics"
	"github.com/ss497254/gloski/internal/provisioning"
	"github.com/ss497254/gloski/internal/tailnet"
	"github.com/ss497254/gloski/proto/agentv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// Tower owns the control tower state, HTTP API, and agent connections.
type Tower struct {
	cfg           *config.Config
	db            *database.Database
	fleetSvc      *fleet.Registry
	fleetEvents   *fleet.EventHub
	auth          *auth.Service
	installer     *provisioning.Installer
	dist          *agentdist.Dist
	agentBinary   string // agent binary beside the master, if any
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
	listenersMu       sync.Mutex
	listeners         []net.Listener
	localAgentProcess *localAgent
	lifecycle         *lifecycle

	// transfers holds file transfers the agent still has to answer on HTTP.
	transfersMu sync.Mutex
	transfers   map[string]*transfer

	checkStore  *checks.Store
	actionStore *actions.Store
	metricStore *metrics.Store

	// Read-only state for the self-test. Maintenance records whether its last
	// run succeeded; self-test runs keep three runtime points for a trend.
	maintenanceMu      sync.Mutex
	lastMaintenance    time.Time
	lastMaintenanceErr error
	runtimeSamples     []runtimeSample
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

	httpapi.SetDetailedErrors(cfg.DetailedErrors)

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "gloski-tower"
	}

	metricStore := metrics.NewStore(db)
	agentService, err := agents.New(db, fleetSvc, metricStore, version, cfg.Master.HeartbeatInterval)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize agent service: %w", err)
	}

	fleetSvc.SetOnline(agentService.Connected)

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(32<<20),
		grpc.MaxSendMsgSize(32<<20),
		// Agents ping every 20 seconds; allow that, and ping back so a
		// vanished agent is noticed without waiting on TCP timeouts.
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{MinTime: 10 * time.Second, PermitWithoutStream: true}),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 20 * time.Second, Timeout: 10 * time.Second}),
	)
	agentv1.RegisterAgentServer(grpcServer, agentService)

	agentBinary := resolveAgentBinary(cfg.Master.AgentBinary)
	dist := agentdist.New(builds.FS, agentBinary)

	t := &Tower{
		dist:            dist,
		agentBinary:     agentBinary,
		cfg:             cfg,
		db:              db,
		fleetSvc:        fleetSvc,
		fleetEvents:     fleet.NewEventHub(),
		auth:            authSvc,
		installer:       provisioning.NewInstaller(dist, cfg.Master.SSHKeyFile, tailnet.KnownHostsPath(cfg.DataDir)),
		tailnetClient:   tailnet.New(cfg.Tailscale.Socket),
		version:         version,
		name:            hostname,
		startedAt:       time.Now().UTC(),
		stop:            make(chan struct{}),
		done:            make(chan struct{}),
		agentService:    agentService,
		agentGRPCServer: grpcServer,
		transfers:       make(map[string]*transfer),
		checkStore:      checks.NewStore(db),
		actionStore:     actions.NewStore(db),
		metricStore:     metricStore,
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
	go t.maintenanceLoop()
	t.lifecycle = newLifecycle(t)
	go t.lifecycle.loop()
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
		// Agent streams never end on their own, so GracefulStop alone would
		// wait forever. Hang them up first and force the stop if that stalls.
		t.agentService.Close()
		stopped := make(chan struct{})
		go func() {
			t.agentGRPCServer.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(agentShutdownWait):
			t.agentGRPCServer.Stop()
		}
	}
	t.listenersMu.Lock()
	for _, listener := range t.listeners {
		_ = listener.Close()
	}
	t.listenersMu.Unlock()
	if t.agentSocketPath != "" {
		_ = os.Remove(t.agentSocketPath)
	}
	return t.db.Close()
}

// fleetLoop keeps online/offline status fresh for dashboards. Heartbeats also
// trigger broadcasts, but a quiet or dead agent would otherwise leave the
// fleet showing a stale status indefinitely.
