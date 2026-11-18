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
	"github.com/ss497254/gloski/internal/master"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("c", "", "Path to config file")
	mode := flag.String("mode", "", "Operating mode: standalone or master")
	showVersion := flag.Bool("v", false, "Show version information")
	flag.Parse()

	if *showVersion {
		logger.Info("Gloski %s (built %s)", version, buildTime)
		os.Exit(0)
	}

	if *mode != "" {
		os.Setenv("GLOSKI_MODE", *mode)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Debug("Failed to load config.")
		logger.Fatal("%v", err)
	}
	logger.SetLevel(logger.ParseLevel(cfg.LogLevel))

	switch cfg.Mode {
	case config.ModeMaster:
		logger.Info("Gloski control tower %s starting...", version)
		if err := master.Run(cfg, version); err != nil {
			logger.Fatal("Control tower failed: %v", err)
		}
	case config.ModeAgent:
		logger.Fatal("gloski no longer runs in agent mode; use the gloski-agent binary")
	case config.ModeStandalone:
		logger.Info("Gloski %s starting in standalone mode...", version)
		runStandalone(cfg)
	default:
		logger.Fatal("Unknown mode %q", cfg.Mode)
	}
}

func runStandalone(cfg *config.Config) {
	logger.Info("Gloski %s starting...", version)
	logger.Debug("Config loaded: host=%s port=%d", cfg.Host, cfg.Port)

	application, err := app.New(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize application: %v", err)
	}

	apiServer := api.NewServer(application, api.ServerOptions{Version: version})

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           apiServer.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("Server listening on http://%s%s", srv.Addr, cfg.APIPrefix)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("Received signal %v, shutting down...", sig)

	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 5
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
	defer cancel()

	countdownDone := make(chan struct{})
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

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced to shutdown: %v", err)
	}
	apiServer.Shutdown()
	if err := application.Shutdown(ctx); err != nil {
		logger.Error("Application shutdown error: %v", err)
	}
	close(countdownDone)
	logger.Info("Shutdown complete")
}
