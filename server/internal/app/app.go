// Package app provides application-level orchestration and dependency injection.
package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/cron"
	"github.com/ss497254/gloski/internal/docker"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/packages"
	"github.com/ss497254/gloski/internal/system"
	"github.com/ss497254/gloski/internal/tasks"
)

// App is the main application container that holds all services.
type App struct {
	Config *config.Config

	// Core services
	Auth   *auth.Service
	Files  *files.Service
	System *system.Service
	Tasks  *tasks.Service

	// Optional services (may be nil if not available)
	Docker   *docker.Service
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

	// Initialize core services (always available)
	app.Auth = auth.NewService(cfg)
	app.Files = files.NewService(cfg)
	app.System = system.NewService()
	app.Tasks = tasks.NewService()

	// Initialize optional services (may fail gracefully)
	app.initOptionalServices()

	logger.Info("Application initialized")
	return app, nil
}

// initOptionalServices initializes services that may not be available on all systems.
func (a *App) initOptionalServices() {
	// Docker service
	dockerSvc, err := docker.NewService()
	if err != nil {
		logger.Debug("Docker service not available: %v", err)
	} else {
		a.Docker = dockerSvc
		logger.Info("Docker service initialized")
	}

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

	// Shutdown tasks service (kills running tasks)
	if err := a.Tasks.Shutdown(); err != nil {
		errs = append(errs, fmt.Errorf("tasks: %w", err))
	}

	// Shutdown docker service
	if a.Docker != nil {
		if err := a.Docker.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("docker: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	logger.Info("All services shut down")
	return nil
}

// HasDocker returns true if Docker is available.
func (a *App) HasDocker() bool {
	return a.Docker != nil
}

// HasPackages returns true if package management is available.
func (a *App) HasPackages() bool {
	return a.Packages != nil
}

// HasCron returns true if cron is available.
func (a *App) HasCron() bool {
	return a.Cron != nil
}

// Features returns a map of available features.
func (a *App) Features() map[string]bool {
	return map[string]bool{
		"docker":   a.HasDocker(),
		"packages": a.HasPackages(),
		"cron":     a.HasCron(),
		"systemd":  a.System.HasSystemd(),
	}
}
