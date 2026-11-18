package main

import (
	"flag"
	"os"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/worker"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("c", "", "Path to config file")
	showVersion := flag.Bool("v", false, "Show version information")
	flag.Parse()

	if *showVersion {
		logger.Info("Gloski agent %s (built %s)", version, buildTime)
		os.Exit(0)
	}

	// This binary is always an agent, regardless of the deployed config file.
	os.Setenv("GLOSKI_MODE", config.ModeAgent)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load config: %v", err)
	}
	logger.SetLevel(logger.ParseLevel(cfg.LogLevel))

	logger.Info("Gloski agent %s starting...", version)
	if err := worker.Run(cfg, version); err != nil {
		logger.Fatal("Agent failed: %v", err)
	}
}
