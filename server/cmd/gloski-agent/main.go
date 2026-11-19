// Command gloski-agent runs the device-side agent: one gRPC stream to the
// master, executing jobs, file operations and terminal sessions.
package main

import (
	"flag"
	"fmt"
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
		fmt.Printf("gloski-agent %s (built %s)\n", version, buildTime)
		return
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
