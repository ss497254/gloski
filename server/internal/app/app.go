// Package app provides application-level orchestration and dependency injection.
package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/cron"
	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/downloads"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/internal/jobs"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/packages"
	"github.com/ss497254/gloski/internal/system"
)

// App is the main application container that holds all services.
type App struct {
	Config *config.Config

	// Database
	DB *database.Database

	// Core services
	Auth      *auth.Service
	Files     *files.Service
	System    *system.Service
	Jobs      *jobs.Service
	Downloads *downloads.Service

	// Background services
	statsCollector *system.Collector
	statsHub       *system.Hub

	// Optional services (may be nil if not available)
	Packages *packages.Service
	Cron     *cron.Service

	// Internal state
	mu       sync.RWMutex
	shutdown bool
}

// New creates a new application instance with all services initialized.
func New(cfg *config.Config) (*App, error) {
	app := &App{
		Config: cfg,
	}

	// Initialize database
	db, err := database.Open(cfg.DatabasePath())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	app.DB = db
	logger.Info("Database initialized at %s", cfg.DatabasePath())

	// Initialize system stats store, hub, and collector
	// Store capacity uses default from NewStore (300 samples = 10 minutes at 2s interval)
	statsStore := system.NewStore(0) // 0 = use default capacity
	app.statsHub = system.NewHub(statsStore)
	go app.statsHub.Run() // Start hub in background
	logger.Info("WebSocket stats hub initialized")

	app.statsCollector = system.NewCollector(statsStore, app.statsHub, 2*time.Second)
	app.statsCollector.Start()

	// Initialize core services (always available)
	authService, err := auth.NewService(cfg)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize auth service: %w", err)
	}
	app.Auth = authService
	app.Files = files.NewService(cfg)
	app.System = system.NewService(statsStore, app.statsHub)

	// Initialize jobs service if enabled
	if cfg.Jobs.Enabled {
		jobsService, err := jobs.NewService(db, jobs.Config{
			LogsDir: cfg.LogsDir(),
			MaxJobs: cfg.Jobs.MaxJobs,
		})
		if err != nil {
			logger.Warn("Failed to initialize jobs service: %v", err)
		} else {
			app.Jobs = jobsService
			logger.Info("Jobs service initialized")
		}
	}

	// Initialize downloads service if enabled
	if cfg.Downloads.Enabled {
		downloadService, err := downloads.NewService(db, downloads.Config{
			MaxConcurrent: cfg.Downloads.MaxConcurrent,
			MaxRetries:    cfg.Downloads.MaxRetries,
			BaseURL:       cfg.BaseURL,
		})
		if err != nil {
			logger.Warn("Failed to initialize downloads service: %v", err)
		} else {
			app.Downloads = downloadService
			logger.Info("Downloads service initialized")
		}
	}

	// Initialize optional services (may fail gracefully)
	app.initOptionalServices()

	logger.Info("Application initialized")
	return app, nil
}

// initOptionalServices initializes services that may not be available on all systems.
func (a *App) initOptionalServices() {
	// Package manager service
	pkgSvc, err := packages.NewService()
	if err != nil {
		logger.Debug("Package manager service not available: %v", err)
	} else {
		a.Packages = pkgSvc
		logger.Info("Package manager service initialized (%s)", pkgSvc.Manager())
	}

	// Cron service
	cronSvc, err := cron.NewService()
	if err != nil {
		logger.Debug("Cron service not available: %v", err)
	} else {
		a.Cron = cronSvc
		logger.Info("Cron service initialized")
	}
}

// Shutdown gracefully shuts down all services.
func (a *App) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	if a.shutdown {
		a.mu.Unlock()
		return nil
	}
	a.shutdown = true
	a.mu.Unlock()

	logger.Info("Shutting down application services...")

	var errs []error

	// Stop stats collector
	if a.statsCollector != nil {
		a.statsCollector.Stop()
	}

	// Shutdown jobs service (kills running jobs)
	if a.Jobs != nil {
		if err := a.Jobs.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("jobs: %w", err))
		}
	}

	// Shutdown downloads service
	if a.Downloads != nil {
		// Calculate remaining time from context deadline
		timeout := 5 * time.Second
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining > 0 {
				timeout = remaining
			}
		}
		a.Downloads.Shutdown(timeout)
	}

	// Close database
	if a.DB != nil {
		if err := a.DB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("database: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	logger.Info("All services shut down")
	return nil
}

// HasPackages returns true if package management is available.
func (a *App) HasPackages() bool {
	return a.Packages != nil
}

// HasCron returns true if cron is available.
func (a *App) HasCron() bool {
	return a.Cron != nil
}

// HasDownloads returns true if downloads feature is available.
func (a *App) HasDownloads() bool {
	return a.Downloads != nil
}

// HasJobs returns true if jobs feature is available.
func (a *App) HasJobs() bool {
	return a.Jobs != nil
}

// Features returns a map of available features.
func (a *App) Features() map[string]bool {
	return map[string]bool{
		"packages":  a.HasPackages(),
		"cron":      a.HasCron(),
		"downloads": a.HasDownloads(),
		"jobs":      a.HasJobs(),
	}
}
