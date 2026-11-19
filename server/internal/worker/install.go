package worker

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/ss497254/gloski/internal/agentdist"
	"github.com/ss497254/gloski/internal/logger"
)

// installDir is where this agent was installed, or "" when it was started by
// hand or by the master as its local agent.
func installDir() string {
	return strings.TrimSpace(os.Getenv("GLOSKI_AGENT_INSTALL_DIR"))
}

// removeInstall runs the uninstall script for this agent's own install. The
// script stops the supervisor, which stops this process, so it runs in its own
// session to outlive both. With stopOnly the files are kept.
func removeInstall(stopOnly bool) {
	dir := installDir()
	if dir == "" {
		return
	}
	command := exec.Command("sh", "-c", agentdist.UninstallScript(dir, stopOnly))
	command.Dir = "/"
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		logger.Error("Could not run the uninstall script: %v", err)
		return
	}
	_ = command.Process.Release()
}
