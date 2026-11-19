package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/supervisor"
	"github.com/ss497254/gloski/internal/worker"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// "start" and "supervise" keep an installed agent alive; see package
	// supervisor. With no subcommand this process is the agent itself.
	if len(os.Args) > 1 && (os.Args[1] == "start" || os.Args[1] == "supervise") {
		flags := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
		dir := flags.String("dir", "", "Install directory")
		_ = flags.Parse(os.Args[2:])
		if *dir == "" {
			fmt.Fprintln(os.Stderr, "-dir is required")
			os.Exit(2)
		}
		if os.Args[1] == "supervise" {
			os.Exit(supervisor.Run(*dir))
		}
		if err := supervisor.Start(*dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

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
