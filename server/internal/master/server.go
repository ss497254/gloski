// Package master implements the Gloski control tower: the fleet registry,
// device proxy, embedded dashboard, and agent enrollment API.
package master

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/agentgrpc"
	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/api/middleware"
	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/deploy"
	"github.com/ss497254/gloski/internal/devices"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/tailscale"
	"github.com/ss497254/gloski/proto/agentv1"
	"google.golang.org/grpc"
)

// Server is the master control tower.
type Server struct {
	cfg       *config.Config
	db        *database.Database
	devices   *devices.Service
	hub       *devices.Hub
	auth      *auth.Service
	deployer  *deploy.Manager
	tailscale *tailscale.Client
	transport *http.Transport
	router    http.Handler
	version   string
	name      string
	startedAt time.Time
	webDir    string
	indexHTML []byte
	stop      chan struct{}
	done      chan struct{}

	agents    *agentgrpc.Server
	agentGRPC *grpc.Server
	agentSock string
	listeners []net.Listener
	local     *localAgent
}

// New creates a master server and opens its database.
func New(cfg *config.Config, version string) (*Server, error) {
	db, err := database.Open(cfg.MasterDBPath())
	if err != nil {
		return nil, fmt.Errorf("failed to open master database: %w", err)
	}

	devicesSvc, err := devices.NewService(db, cfg.Master, version, cfg.DataDir)
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

	agentService, err := agentgrpc.New(db, devicesSvc, version, cfg.Master.HeartbeatInterval)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize agent service: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(32<<20), grpc.MaxSendMsgSize(32<<20))
	agentv1.RegisterAgentServer(grpcServer, agentService)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 30 * time.Second
	transport.MaxIdleConnsPerHost = 8

	s := &Server{
		cfg:       cfg,
		db:        db,
		devices:   devicesSvc,
		hub:       devices.NewHub(),
		auth:      authSvc,
		deployer:  deploy.New(resolveAgentBinary(cfg.Master.AgentBinary), cfg.Master.SSHKeyFile, tailscale.KnownHostsPath(cfg.DataDir)),
		tailscale: tailscale.New(cfg.Tailscale.Socket),
		transport: transport,
		version:   version,
		name:      hostname,
		startedAt: time.Now().UTC(),
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
		agents:    agentService,
		agentGRPC: grpcServer,
	}

	devicesSvc.SetOnChange(func() {
		list, err := devicesSvc.List()
		if err != nil {
			return
		}
		s.hub.Broadcast(devices.Event{Type: "devices", Devices: list})
	})

	s.webDir = resolveWebDir(cfg.Master.WebDir)
	if s.webDir != "" {
		if data, err := os.ReadFile(filepath.Join(s.webDir, "index.html")); err == nil {
			s.indexHTML = data
		}
	}

	go s.fleetLoop()
	s.router = s.routes()
	s.startAgentTransport(cfg)
	s.startLocalAgent(cfg, version)
	logger.Info("Control tower initialized (db=%s, web=%s)", cfg.MasterDBPath(), displayPath(s.webDir))
	return s, nil
}

// Router returns the master HTTP handler.
func (s *Server) Router() http.Handler {
	return s.router
}

// JoinToken returns the token agents use to enroll.
func (s *Server) JoinToken() string {
	return s.devices.JoinToken()
}

// Close releases master resources.
func (s *Server) Close() error {
	close(s.stop)
	<-s.done
	if s.local != nil {
		s.local.stop()
	}
	if s.agentGRPC != nil {
		s.agentGRPC.GracefulStop()
	}
	for _, listener := range s.listeners {
		_ = listener.Close()
	}
	if s.agentSock != "" {
		_ = os.Remove(s.agentSock)
	}
	return s.db.Close()
}

// fleetLoop keeps online/offline status fresh for dashboards. Heartbeats also
// trigger broadcasts, but a quiet or dead agent would otherwise leave the
// fleet showing a stale status indefinitely.
func (s *Server) fleetLoop() {
	defer close(s.done)

	interval := time.Duration(s.devices.HeartbeatInterval()) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			list, err := s.devices.List()
			if err != nil {
				logger.Warn("Failed to refresh fleet status: %v", err)
				continue
			}
			s.hub.Broadcast(devices.Event{Type: "devices", Devices: list})
		}
	}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// Public control-plane endpoints. Agents authenticate with a join token or
	// their device token; the dashboard is protected by tailnet access and,
	// optionally, the configured API key / JWT.
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		handlers.Success(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/tower/info", s.handleTowerInfo)
	mux.Handle("GET /api/dashboard", s.userAuth(http.HandlerFunc(s.handleDashboard)))
	mux.HandleFunc("POST /api/agents/register", s.handleRegister)
	mux.HandleFunc("POST /api/agents/heartbeat", s.handleHeartbeat)
	mux.HandleFunc("GET /api/fleet/ws", s.handleFleetWS)

	// Fleet API. If the operator configured an API key or JWT, require it.
	mux.Handle("GET /api/devices", s.userAuth(http.HandlerFunc(s.handleListDevices)))
	mux.Handle("POST /api/devices/deploy", s.userAuth(http.HandlerFunc(s.handleDeployDevice)))
	mux.Handle("GET /api/devices/{id}", s.userAuth(http.HandlerFunc(s.handleGetDevice)))
	mux.Handle("DELETE /api/devices/{id}", s.userAuth(http.HandlerFunc(s.handleDeleteDevice)))
	mux.Handle("POST /api/devices/{id}/start", s.userAuth(http.HandlerFunc(s.handleStartDevice)))
	mux.Handle("POST /api/devices/{id}/stop", s.userAuth(http.HandlerFunc(s.handleStopDevice)))

	// Browser-facing device API. The master translates each call into gRPC
	// commands on the device's agent stream.
	mux.Handle("GET /api/devices/{id}/api/system/stats", s.userAuth(http.HandlerFunc(s.handleDeviceStats)))
	mux.Handle("GET /api/devices/{id}/api/system/stats/ws", s.userAuth(http.HandlerFunc(s.handleDeviceStatsWS)))
	mux.Handle("GET /api/devices/{id}/api/system/processes", s.userAuth(http.HandlerFunc(s.handleDeviceProcesses)))
	mux.Handle("GET /api/devices/{id}/api/jobs", s.userAuth(http.HandlerFunc(s.handleListJobs)))
	mux.Handle("POST /api/devices/{id}/api/jobs", s.userAuth(http.HandlerFunc(s.handleRunJob)))
	mux.Handle("GET /api/devices/{id}/api/jobs/{jobId}", s.userAuth(http.HandlerFunc(s.handleGetJob)))
	mux.Handle("GET /api/devices/{id}/api/jobs/{jobId}/logs", s.userAuth(http.HandlerFunc(s.handleJobLogs)))
	mux.Handle("POST /api/devices/{id}/api/jobs/{jobId}/stop", s.userAuth(http.HandlerFunc(s.handleCancelJob)))
	mux.Handle("DELETE /api/devices/{id}/api/jobs/{jobId}", s.userAuth(http.HandlerFunc(s.handleDeleteJob)))
	mux.Handle("GET /api/devices/{id}/api/files", s.userAuth(http.HandlerFunc(s.handleListFiles)))
	mux.Handle("DELETE /api/devices/{id}/api/files", s.userAuth(http.HandlerFunc(s.handleDeleteFile)))
	mux.Handle("GET /api/devices/{id}/api/files/read", s.userAuth(http.HandlerFunc(s.handleReadFile)))
	mux.Handle("GET /api/devices/{id}/api/files/download", s.userAuth(http.HandlerFunc(s.handleDownloadFile)))
	mux.Handle("POST /api/devices/{id}/api/files/write", s.userAuth(http.HandlerFunc(s.handleWriteFile)))
	mux.Handle("POST /api/devices/{id}/api/files/mkdir", s.userAuth(http.HandlerFunc(s.handleMkdir)))
	mux.Handle("POST /api/devices/{id}/api/files/rename", s.userAuth(http.HandlerFunc(s.handleRename)))
	mux.Handle("POST /api/devices/{id}/api/files/upload", s.userAuth(http.HandlerFunc(s.handleUploadFile)))
	mux.Handle("GET /api/devices/{id}/api/terminal", s.userAuth(http.HandlerFunc(s.handleTerminalWS)))

	// SSH deployment and tailnet discovery.
	mux.Handle("GET /api/deployments", s.userAuth(http.HandlerFunc(s.handleListDeployments)))
	mux.Handle("GET /api/deployments/{id}", s.userAuth(http.HandlerFunc(s.handleGetDeployment)))
	mux.Handle("GET /api/tailnet/peers", s.userAuth(http.HandlerFunc(s.handleTailnetPeers)))

	// Dashboard static assets and SPA fallback.
	mux.HandleFunc("/", s.serveSPA)

	return middleware.Logging(securityHeaders(mux))
}

func (s *Server) userAuth(next http.Handler) http.Handler {
	if s.cfg.APIKey == "" && s.cfg.JWTPublicKey == "" {
		return next
	}
	return middleware.Auth(s.auth)(next)
}

func (s *Server) serveSPA(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		handlers.NotFound(w, "endpoint not found")
		return
	}

	clean := path.Clean("/" + r.URL.Path)
	if clean == "/" {
		s.writeIndex(w)
		return
	}

	if s.webDir != "" {
		file := filepath.Join(s.webDir, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			if strings.HasPrefix(clean, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFile(w, r, file)
			return
		}
	}

	// Client-side routes fall back to index.html; missing assets return 404.
	if path.Ext(clean) != "" {
		http.NotFound(w, r)
		return
	}
	s.writeIndex(w)
}

func (s *Server) writeIndex(w http.ResponseWriter) {
	// The dashboard can be rebuilt while the master is running. Always read
	// index.html from disk so asset hashes stay in sync with web/dist; the
	// startup copy is only a fallback.
	index := s.indexHTML
	if s.webDir != "" {
		if data, err := os.ReadFile(filepath.Join(s.webDir, "index.html")); err == nil && len(data) > 0 {
			index = data
		}
	}
	if len(index) == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, placeholderHTML)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(index)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func resolveAgentBinary(configured string) string {
	candidates := []string{configured}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "gloski-agent"))
	}
	candidates = append(candidates,
		filepath.Join("bin", "gloski-agent"),
		filepath.Join("..", "bin", "gloski-agent"),
		filepath.Join("..", "..", "bin", "gloski-agent"),
	)

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return configured
}

func resolveWebDir(configured string) string {
	candidates := []string{configured}
	if configured == "" || configured == "web/dist" {
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates,
				filepath.Join(filepath.Dir(exe), "web", "dist"),
				filepath.Join(filepath.Dir(exe), "dist"),
			)
		}
		candidates = append(candidates, "web/dist", "../web/dist", "../../web/dist")
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func displayPath(dir string) string {
	if dir == "" {
		return "(dashboard not bundled; run make build-web)"
	}
	return dir
}

const placeholderHTML = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Gloski Tower</title>
<style>
body{font:14px/1.6 ui-monospace,SFMono-Regular,Menlo,monospace;background:#0b0f14;color:#c9d1d9;display:grid;place-items:center;height:100vh;margin:0}
main{max-width:42rem;padding:2rem}
h1{color:#7ee787;font-size:1.2rem}code{color:#79c0ff}
</style></head>
<body><main>
<h1>Gloski control tower is running</h1>
<p>The API is online, but the dashboard bundle was not found.</p>
<p>Build it with <code>make build-web</code> (or <code>cd web &amp;&amp; bun run build</code>) and restart the master, or point <code>master.web_dir</code> at <code>web/dist</code>.</p>
<p>API: <code>/api/tower/info</code> · <code>/api/devices</code></p>
</main></body></html>`
