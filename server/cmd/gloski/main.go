// Command gloski runs the master: the dashboard, browser API, SQLite state and
// agent lifecycle. The device-side binary is gloski-agent.
package main

import (
	"flag"
	"os"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/controltower"
	"github.com/ss497254/gloski/internal/logger"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("c", "", "Path to config file")
	mode := flag.String("mode", "", "Operating mode (master by default)")
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
		if err := controltower.Run(cfg, version); err != nil {
			logger.Fatal("Control tower failed: %v", err)
		}
	case config.ModeAgent:
		logger.Fatal("gloski no longer runs in agent mode; use the gloski-agent binary")
	default:
		logger.Fatal("Unknown mode %q", cfg.Mode)
	}
}
