package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/ss497254/gloski/internal/agentdist"
	"github.com/ss497254/gloski/internal/logger"
)

// The agent does not manage its own lifecycle. The master installs, starts,
// restarts and replaces it; the agent only reports which build it is and
// removes itself when told the device was deleted.

// buildHash is the SHA-256 of this agent's binary. The master compares it
// with the build it would install to decide whether to replace the agent.
func buildHash() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return ""
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// installDir is where the master installed this agent, or "" when it was
// started by hand or as the master's local agent.
func installDir() string {
	return strings.TrimSpace(os.Getenv("GLOSKI_AGENT_INSTALL_DIR"))
}

// removeInstall deletes this agent's own install. The script kills this
// process, so it runs in its own session to finish the job.
func removeInstall() {
	dir := installDir()
	if dir == "" {
		return
	}
	command := exec.Command("sh", "-c", agentdist.UninstallScript(dir, false))
	command.Dir = "/"
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		logger.Error("Could not run the uninstall script: %v", err)
		return
	}
	_ = command.Process.Release()
}
