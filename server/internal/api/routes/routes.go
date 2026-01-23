package routes

import (
	"net/http"

	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/api/middleware"
	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/cron"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/internal/packages"
	"github.com/ss497254/gloski/internal/system"
	"github.com/ss497254/gloski/internal/tasks"
)

// Config holds all dependencies needed for routing
type Config struct {
	Cfg         *config.Config
	AuthService *auth.Service
	FileService *files.Service
	TaskService *tasks.Service
	SysService  *system.Service

	// Optional services
	PackagesService *packages.Service
	CronService     *cron.Service

	// Features map for /api/system/info
	Features map[string]bool

	// Server metadata
	Version string
}

// Setup configures all routes and returns the root handler
func Setup(cfg Config) http.Handler {
	mux := http.NewServeMux()

	// Create handlers
	healthHandler := handlers.NewHealthHandler()
	healthHandler.SetFeatures(cfg.Features)
	if cfg.Version != "" {
		healthHandler.SetVersion(cfg.Version)
	}
	authHandler := handlers.NewAuthHandler(cfg.AuthService)
	systemHandler := handlers.NewSystemHandler(cfg.SysService)
	filesHandler := handlers.NewFilesHandler(cfg.FileService)
	tasksHandler := handlers.NewTasksHandler(cfg.TaskService)
	terminalHandler := handlers.NewTerminalHandler(cfg.Cfg, cfg.AuthService)

	// Set features if available
	if cfg.Features != nil {
		systemHandler.SetFeatures(cfg.Features)
	}

	// Optional handlers
	packagesHandler := handlers.NewPackagesHandler(cfg.PackagesService)
	cronHandler := handlers.NewCronHandler(cfg.CronService)

	// Auth middleware
	requireAuth := middleware.Auth(cfg.AuthService)

	// Health check (public)
	mux.HandleFunc("GET /api/health", healthHandler.Check)
	mux.HandleFunc("GET /api/health/ready", healthHandler.Ready)
	mux.HandleFunc("GET /api/health/live", healthHandler.Live)

	// Auth routes
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.Handle("GET /api/auth/status", requireAuth(http.HandlerFunc(authHandler.Status)))

	// System routes (protected)
	mux.Handle("GET /api/system/stats", requireAuth(http.HandlerFunc(systemHandler.GetStats)))
	mux.Handle("GET /api/system/info", requireAuth(http.HandlerFunc(systemHandler.GetInfo)))
	mux.Handle("GET /api/system/processes", requireAuth(http.HandlerFunc(systemHandler.GetProcesses)))

	// File routes (protected)
	mux.Handle("GET /api/files", requireAuth(http.HandlerFunc(filesHandler.List)))
	mux.Handle("GET /api/files/read", requireAuth(http.HandlerFunc(filesHandler.Read)))
	mux.Handle("POST /api/files/write", requireAuth(http.HandlerFunc(filesHandler.Write)))
	mux.Handle("POST /api/files/mkdir", requireAuth(http.HandlerFunc(filesHandler.Mkdir)))
	mux.Handle("DELETE /api/files", requireAuth(http.HandlerFunc(filesHandler.Delete)))
	mux.Handle("POST /api/files/upload", requireAuth(http.HandlerFunc(filesHandler.Upload)))
	mux.Handle("GET /api/files/download", requireAuth(http.HandlerFunc(filesHandler.Download)))

	// Search route (protected)
	mux.Handle("GET /api/search", requireAuth(http.HandlerFunc(filesHandler.Search)))

	// Terminal WebSocket (auth via query param)
	mux.HandleFunc("GET /api/terminal", terminalHandler.Handle)

	// Task routes (protected)
	mux.Handle("GET /api/tasks", requireAuth(http.HandlerFunc(tasksHandler.List)))
	mux.Handle("POST /api/tasks", requireAuth(http.HandlerFunc(tasksHandler.Start)))
	mux.Handle("GET /api/tasks/{id}", requireAuth(http.HandlerFunc(tasksHandler.Get)))
	mux.Handle("GET /api/tasks/{id}/logs", requireAuth(http.HandlerFunc(tasksHandler.GetLogs)))
	mux.Handle("DELETE /api/tasks/{id}", requireAuth(http.HandlerFunc(tasksHandler.Stop)))

	// Systemd routes (protected)
	mux.Handle("GET /api/systemd", requireAuth(http.HandlerFunc(tasksHandler.ListSystemd)))
	mux.Handle("POST /api/systemd/{unit}/{action}", requireAuth(http.HandlerFunc(tasksHandler.SystemdAction)))
	mux.Handle("GET /api/systemd/{unit}/logs", requireAuth(http.HandlerFunc(tasksHandler.SystemdLogs)))

	// Package management routes (protected, optional)
	mux.Handle("GET /api/packages/info", requireAuth(http.HandlerFunc(packagesHandler.Info)))
	mux.Handle("GET /api/packages/installed", requireAuth(http.HandlerFunc(packagesHandler.ListInstalled)))
	mux.Handle("GET /api/packages/upgrades", requireAuth(http.HandlerFunc(packagesHandler.CheckUpgrades)))
	mux.Handle("GET /api/packages/search", requireAuth(http.HandlerFunc(packagesHandler.Search)))
	mux.Handle("GET /api/packages/{name}", requireAuth(http.HandlerFunc(packagesHandler.GetPackageInfo)))

	// Cron routes (protected, optional)
	mux.Handle("GET /api/cron/jobs", requireAuth(http.HandlerFunc(cronHandler.ListJobs)))
	mux.Handle("POST /api/cron/jobs", requireAuth(http.HandlerFunc(cronHandler.AddJob)))
	mux.Handle("DELETE /api/cron/jobs", requireAuth(http.HandlerFunc(cronHandler.RemoveJob)))

	// Apply global middleware
	corsConfig := middleware.CORSConfig{
		AllowedOrigins: cfg.Cfg.AllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
		MaxAge:         86400,
	}

	handler := middleware.Chain(
		middleware.Logging,
		middleware.CORS(corsConfig),
	)(mux)

	return handler
}
