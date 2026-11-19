package controltower

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
)

func (t *Tower) startLocalAgent(cfg *config.Config, version string) {
	if !cfg.Master.LocalAgent || t.agentSocketPath == "" {
		return
	}
	binary := t.localAgentBinary()
	if binary == "" {
		logger.Warn("No agent binary found; the master device will not appear (run 'make build')")
		return
	}
	name := cfg.Master.LocalAgentName
	if name == "" {
		name = t.name
	}
	agent := &localAgent{
		binary:    binary,
		masterURL: "unix://" + t.agentSocketPath,
		joinToken: t.fleetSvc.JoinToken(),
		name:      name,
		version:   version,
		shell:     cfg.Shell,
		dataDir:   filepath.Join(cfg.DataDir, "local-agent"),
		logPath:   filepath.Join(cfg.DataDir, "local-agent.log"),
		done:      make(chan struct{}),
		stopCh:    make(chan struct{}),
		health: func() bool {
			list, err := t.fleetSvc.List()
			if err != nil {
				logger.Warn("local agent health check failed: %v", err)
				return false
			}
			for i := range list {
				if list[i].Master && list[i].Name == name {
					return t.agentService.Connected(list[i].ID)
				}
			}
			return false
		},
	}
	t.localAgentProcess = agent
	go agent.run()
}

type localAgent struct {
	binary    string
	masterURL string
	joinToken string
	name      string
	version   string
	shell     string
	dataDir   string
	logPath   string
	health    func() bool

	mu       sync.Mutex
	stopping bool
	cmd      *exec.Cmd
	done     chan struct{}
	stopCh   chan struct{}
	stopOnce sync.Once
}

func (l *localAgent) run() {
	defer close(l.done)
	backoff := time.Second

	for !l.isStopping() {
		command, logFile, exited, err := l.startProcess()
		if err != nil {
			l.wait(backoff)
			backoff = nextBackoff(backoff)
			continue
		}

		stopPermanently := l.supervise(command, exited)
		_ = logFile.Close()
		l.clearCommand()
		if stopPermanently {
			return
		}

		logger.Info("local agent: restart in %s", backoff)
		l.wait(backoff)
		backoff = nextBackoff(backoff)
	}
}

func (l *localAgent) startProcess() (*exec.Cmd, *os.File, <-chan error, error) {
	logFile, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		logger.Error("local agent: failed to open log %s: %v", l.logPath, err)
		return nil, nil, nil, err
	}

	command := exec.Command(l.binary)
	command.Env = append(os.Environ(),
		"GLOSKI_MODE=agent",
		"GLOSKI_AGENT_MASTER_URL="+l.masterURL,
		"GLOSKI_AGENT_JOIN_TOKEN="+l.joinToken,
		"GLOSKI_AGENT_NAME="+l.name,
		"GLOSKI_AGENT_IS_MASTER=true",
		"GLOSKI_DATA_DIR="+l.dataDir,
		"GLOSKI_SHELL="+l.shell,
	)
	command.Stdout = logFile
	command.Stderr = logFile
	command.Stdin = nil

	logger.Info("local agent: starting %s (binary=%s)", l.name, l.binary)
	if err := command.Start(); err != nil {
		logger.Error("local agent: failed to start: %v", err)
		_ = logFile.Close()
		return nil, nil, nil, err
	}
	logger.Info("local agent: started pid=%d", command.Process.Pid)

	l.mu.Lock()
	l.cmd = command
	l.mu.Unlock()

	exited := make(chan error, 1)
	go func() { exited <- command.Wait() }()
	return command, logFile, exited, nil
}

// supervise watches the process and its gRPC session. It returns true when the
// master is shutting down and no restart should be attempted.
func (l *localAgent) supervise(command *exec.Cmd, exited <-chan error) bool {
	pid := command.Process.Pid
	ticker := time.NewTicker(agentHealthInterval)
	defer ticker.Stop()

	unhealthySince := time.Time{}
	healthy := false

	for {
		select {
		case <-l.stopCh:
			logger.Info("local agent: stop requested for pid=%d", pid)
			l.terminate(command, exited)
			return true
		case err := <-exited:
			logger.Warn("local agent: process pid=%d exited: %v", pid, err)
			return false
		case <-ticker.C:
			if l.health != nil && l.health() {
				if !healthy {
					logger.Info("local agent: gRPC session is live (pid=%d)", pid)
				}
				healthy = true
				unhealthySince = time.Time{}
				continue
			}
			if healthy {
				logger.Warn("local agent: gRPC session lost (pid=%d)", pid)
			}
			healthy = false
			if unhealthySince.IsZero() {
				unhealthySince = time.Now()
				logger.Warn("local agent: no gRPC session for pid=%d, waiting before restart", pid)
				continue
			}
			if time.Since(unhealthySince) >= agentUnhealthyLimit {
				logger.Warn("local agent: no gRPC session for %s, restarting pid=%d", agentUnhealthyLimit, pid)
				l.terminate(command, exited)
				return false
			}
		}
	}
}

func (l *localAgent) clearCommand() {
	l.mu.Lock()
	l.cmd = nil
	l.mu.Unlock()
}

func nextBackoff(current time.Duration) time.Duration {
	if current < agentMaxBackoff {
		return current * 2
	}
	return current
}

func (l *localAgent) isStopping() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stopping
}

func (l *localAgent) wait(duration time.Duration) {
	select {
	case <-l.stopCh:
	case <-time.After(duration):
	}
}

func (l *localAgent) terminate(command *exec.Cmd, exited <-chan error) {
	if command.Process == nil {
		return
	}
	_ = command.Process.Signal(syscall.SIGTERM)
	select {
	case <-exited:
	case <-time.After(agentShutdownWait):
		_ = command.Process.Kill()
		select {
		case <-exited:
		case <-time.After(agentKillWait):
		}
	}
}

func (l *localAgent) stop() {
	l.stopOnce.Do(func() { close(l.stopCh) })
	l.mu.Lock()
	l.stopping = true
	command := l.cmd
	l.mu.Unlock()
	if command != nil && command.Process != nil {
		_ = command.Process.Signal(syscall.SIGTERM)
	}
	select {
	case <-l.done:
	case <-time.After(agentFinalWait):
	}
}
