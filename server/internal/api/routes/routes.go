package routes

import (
	"database/sql"
	"net/http"

	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/api/middleware"
	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/cron"
	"github.com/ss497254/gloski/internal/downloads"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/internal/jobs"
	"github.com/ss497254/gloski/internal/packages"
	"github.com/ss497254/gloski/internal/system"
)

// Config holds all dependencies needed for routing
type Config struct {
	Cfg         *config.Config
	AuthService *auth.Service
	FileService *files.Service
	JobsService *jobs.Service
	SysService  *system.Service

	// Database for direct DB handlers
	DB *sql.DB

	// Optional services
	PackagesService *packages.Service
	CronService     *cron.Service
	DownloadService *downloads.Service

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
	mux.Handle("GET /api/auth/status", requireAuth(http.HandlerFunc(authHandler.Status)))

	// System routes (protected)
	mux.Handle("GET /api/system/stats", requireAuth(http.HandlerFunc(systemHandler.GetStats)))
	mux.Handle("GET /api/system/stats/history", requireAuth(http.HandlerFunc(systemHandler.GetStatsHistory)))
	mux.Handle("GET /api/system/info", requireAuth(http.HandlerFunc(systemHandler.GetInfo)))
	mux.Handle("GET /api/system/processes", requireAuth(http.HandlerFunc(systemHandler.GetProcesses)))

	// File routes (protected)
	mux.Handle("GET /api/files", requireAuth(http.HandlerFunc(filesHandler.List)))
	mux.Handle("GET /api/files/read", requireAuth(http.HandlerFunc(filesHandler.Read)))
	mux.Handle("POST /api/files/write", requireAuth(http.HandlerFunc(filesHandler.Write)))
	mux.Handle("POST /api/files/mkdir", requireAuth(http.HandlerFunc(filesHandler.Mkdir)))
	mux.Handle("POST /api/files/rename", requireAuth(http.HandlerFunc(filesHandler.Rename)))
	mux.Handle("DELETE /api/files", requireAuth(http.HandlerFunc(filesHandler.Delete)))
	mux.Handle("POST /api/files/upload", requireAuth(http.HandlerFunc(filesHandler.Upload)))
	mux.Handle("GET /api/files/download", requireAuth(http.HandlerFunc(filesHandler.Download)))

	// Pinned folders routes (protected, part of files resource)
	if cfg.DB != nil {
		filesHandler.SetDB(cfg.DB)
		mux.Handle("GET /api/files/pinned", requireAuth(http.HandlerFunc(filesHandler.ListPinned)))
		mux.Handle("POST /api/files/pinned", requireAuth(http.HandlerFunc(filesHandler.CreatePinned)))
		mux.Handle("DELETE /api/files/pinned/{id}", requireAuth(http.HandlerFunc(filesHandler.DeletePinned)))
	}

	// Search route (protected)
	mux.Handle("GET /api/search", requireAuth(http.HandlerFunc(filesHandler.Search)))

	// Terminal WebSocket (auth via query param)
	mux.HandleFunc("GET /api/terminal", terminalHandler.Handle)

	// Jobs routes (protected, optional)
	if cfg.JobsService != nil {
		jobsHandler := handlers.NewJobsHandler(cfg.JobsService)
		mux.Handle("GET /api/jobs", requireAuth(http.HandlerFunc(jobsHandler.List)))
		mux.Handle("POST /api/jobs", requireAuth(http.HandlerFunc(jobsHandler.Start)))
		mux.Handle("GET /api/jobs/{id}", requireAuth(http.HandlerFunc(jobsHandler.Get)))
		mux.Handle("GET /api/jobs/{id}/logs", requireAuth(http.HandlerFunc(jobsHandler.GetLogs)))
		mux.Handle("POST /api/jobs/{id}/stop", requireAuth(http.HandlerFunc(jobsHandler.Stop)))
		mux.Handle("DELETE /api/jobs/{id}", requireAuth(http.HandlerFunc(jobsHandler.Delete)))
	}

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

	// Download manager routes (protected, optional)
	if cfg.DownloadService != nil {
		downloadsHandler := handlers.NewDownloadsHandler(cfg.DownloadService)
		shareHandler := handlers.NewShareHandler(cfg.DownloadService)

		// Download management (protected)
		mux.Handle("GET /api/downloads", requireAuth(http.HandlerFunc(downloadsHandler.List)))
		mux.Handle("POST /api/downloads", requireAuth(http.HandlerFunc(downloadsHandler.Add)))
		mux.Handle("GET /api/downloads/{id}", requireAuth(http.HandlerFunc(downloadsHandler.Get)))
		mux.Handle("DELETE /api/downloads/{id}", requireAuth(http.HandlerFunc(downloadsHandler.Delete)))
		mux.Handle("POST /api/downloads/{id}/pause", requireAuth(http.HandlerFunc(downloadsHandler.Pause)))
		mux.Handle("POST /api/downloads/{id}/resume", requireAuth(http.HandlerFunc(downloadsHandler.Resume)))
		mux.Handle("POST /api/downloads/{id}/cancel", requireAuth(http.HandlerFunc(downloadsHandler.Cancel)))
		mux.Handle("POST /api/downloads/{id}/retry", requireAuth(http.HandlerFunc(downloadsHandler.Retry)))
		mux.Handle("GET /api/downloads/{id}/file", requireAuth(http.HandlerFunc(downloadsHandler.DownloadFile)))
		mux.Handle("POST /api/downloads/{id}/share", requireAuth(http.HandlerFunc(downloadsHandler.CreateShareLink)))
		mux.Handle("DELETE /api/downloads/{id}/share/{token}", requireAuth(http.HandlerFunc(downloadsHandler.RevokeShareLink)))

		// Public share endpoint (no auth)
		mux.HandleFunc("GET /api/share/{token}", shareHandler.Download)
	}

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
