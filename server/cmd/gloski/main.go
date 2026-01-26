package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ss497254/gloski/internal/api"
	"github.com/ss497254/gloski/internal/app"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// Command line flags
	configPath := flag.String("config", "", "Path to config file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version
	if *showVersion {
		logger.Info("Gloski %s (built %s)", version, buildTime)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load config: %v", err)
	}

	// Set log level
	logger.SetLevel(logger.ParseLevel(cfg.LogLevel))

	logger.Info("Gloski %s starting...", version)
	logger.Debug("Config loaded: host=%s port=%d", cfg.Host, cfg.Port)

	// Initialize application with all services
	application, err := app.New(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize application: %v", err)
	}

	// Create API server
	apiServer := api.NewServer(application, api.ServerOptions{
		Version: version,
	})

	// HTTP server setup
	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      apiServer.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		logger.Info("Server listening on http://%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Received signal %v, shutting down...", sig)

	// Get shutdown timeout
	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 5
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
	defer cancel()

	countdownDone := make(chan struct{})

	// Start countdown in background (visual indicator during shutdown)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		remaining := shutdownTimeout
		for remaining > 0 {
			select {
			case <-countdownDone:
				return
			case <-ticker.C:
				logger.Info("Shutdown in progress... %ds remaining", remaining)
				remaining--
			}
		}
	}()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced to shutdown: %v", err)
	}

	// Shutdown application services
	if err := application.Shutdown(ctx); err != nil {
		logger.Error("Application shutdown error: %v", err)
	}

	// Stop countdown and log appropriate message
	close(countdownDone)

	logger.Info("Shutdown complete")
}
