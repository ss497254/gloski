// Package deploy copies the agent binary to a remote machine and manages its
// process over SSH. SSH is implemented in-process with x/crypto/ssh; the
// package does not shell out to ssh or scp.
package deploy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	defaultRemoteDir = "/tmp"
	defaultAgentPort = 18081
)

var (
	safeHost   = regexp.MustCompile(`^[A-Za-z0-9._:%\[\]-]+$`)
	safeUser   = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	safeRemote = regexp.MustCompile(`^/?(?:[A-Za-z0-9._-]+/)*[A-Za-z0-9._-]+$`)
)

// Target describes one remote agent deployment.
type Target struct {
	Name    string
	Host    string
	User    string
	SSHPort int
	// AgentPort is retained in deployment records only; agents do not listen on HTTP.
	AgentPort  int
	MasterURL  string
	JoinToken  string
	DeployID   string
	RemoteBase string
}

// Validate normalizes and checks a deployment target.
func (t *Target) Validate() error {
	t.Host = strings.TrimSpace(t.Host)
	t.User = strings.TrimSpace(t.User)
	t.Name = strings.TrimSpace(t.Name)
	t.MasterURL = strings.TrimRight(strings.TrimSpace(t.MasterURL), "/")
	t.RemoteBase = strings.TrimSpace(t.RemoteBase)

	if t.Host == "" || !safeHost.MatchString(t.Host) || strings.HasPrefix(t.Host, "-") {
		return fmt.Errorf("invalid ssh host %q", t.Host)
	}
	if t.User != "" && (!safeUser.MatchString(t.User) || strings.HasPrefix(t.User, "-")) {
		return fmt.Errorf("invalid ssh user %q", t.User)
	}
	if t.SSHPort == 0 {
		t.SSHPort = 22
	}
	if t.AgentPort == 0 {
		t.AgentPort = defaultAgentPort
	}
	if t.SSHPort < 1 || t.SSHPort > 65535 {
		return fmt.Errorf("invalid ssh port %d", t.SSHPort)
	}
	if t.AgentPort < 1 || t.AgentPort > 65535 {
		return fmt.Errorf("invalid agent port %d", t.AgentPort)
	}
	if t.Name == "" {
		t.Name = t.Host
	}
	if t.MasterURL == "" {
		return errors.New("master URL is required")
	}
	parsed, err := url.Parse(t.MasterURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("invalid master URL %q", t.MasterURL)
	}
	if t.RemoteBase == "" {
		t.RemoteBase = defaultRemoteDir
	}
	if !strings.HasPrefix(t.RemoteBase, "/") || !safeRemote.MatchString(strings.TrimPrefix(t.RemoteBase, "/")) {
		return fmt.Errorf("invalid remote directory %q", t.RemoteBase)
	}
	return nil
}

// Manager performs deployments with SSH keys from the environment.
type Manager struct {
	BinaryPath    string
	KeyFile       string
	KeyPassphrase string
	KnownHosts    string
	RemoteBase    string

	mu sync.Mutex
}

// New creates a deployment manager. knownHosts should be a writable file that
// the master owns; unknown hosts are trusted on first use.
func New(binaryPath, keyFile, knownHosts string) *Manager {
	return &Manager{
		BinaryPath: binaryPath,
		KeyFile:    keyFile,
		KnownHosts: knownHosts,
		RemoteBase: defaultRemoteDir,
	}
}

// DefaultRemoteBase returns the default remote directory used for agents.
func DefaultRemoteBase() string {
	return defaultRemoteDir
}

// Available reports whether the agent binary can be found.
func (m *Manager) Available() bool {
	if m == nil || m.BinaryPath == "" {
		return false
	}
	info, err := os.Stat(m.BinaryPath)
	return err == nil && !info.IsDir()
}

// Binary returns the configured agent binary path.
func (m *Manager) Binary() string {
	if m == nil {
		return ""
	}
	return m.BinaryPath
}

// Deploy stops any running agent, uploads the current binary, and starts it.
func (m *Manager) Deploy(ctx context.Context, target Target) (string, error) {
	if !m.Available() {
		return "", fmt.Errorf("agent binary not found at %q", m.BinaryPath)
	}
	m.prepare(&target)
	if err := target.Validate(); err != nil {
		return "", err
	}

	if _, err := m.Stop(ctx, target); err != nil {
		return "", fmt.Errorf("failed to stop existing agent: %w", err)
	}
	if err := m.upload(ctx, target); err != nil {
		return "", err
	}
	return m.Start(ctx, target)
}

// Start launches the already-uploaded agent and returns its remote PID.
func (m *Manager) Start(ctx context.Context, target Target) (string, error) {
	m.prepare(&target)
	if err := target.Validate(); err != nil {
		return "", err
	}

	output, err := m.runScript(ctx, target, startScript(target))
	if err != nil {
		return "", err
	}
	pid := lastLine(output)
	if pid == "" {
		return "", errors.New("remote agent did not return a pid")
	}
	if _, err := strconv.Atoi(pid); err != nil {
		return "", fmt.Errorf("unexpected remote agent output %q", pid)
	}
	return pid, nil
}

// Stop terminates the remote agent. The boolean reports whether a process was
// actually stopped.
func (m *Manager) Stop(ctx context.Context, target Target) (bool, error) {
	m.prepare(&target)
	if err := target.Validate(); err != nil {
		return false, err
	}
	output, err := m.runScript(ctx, target, stopScript(target))
	if err != nil {
		return false, err
	}
	switch lastLine(output) {
	case "stopped":
		return true, nil
	case "not-running":
		return false, nil
	default:
		return false, fmt.Errorf("unexpected remote stop output %q", output)
	}
}

func (m *Manager) upload(ctx context.Context, target Target) error {
	file, err := os.Open(m.BinaryPath)
	if err != nil {
		return fmt.Errorf("failed to open agent binary: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("agent binary %q is a directory", m.BinaryPath)
	}

	paths := targetPaths(target)
	script := fmt.Sprintf(`set -eu
mkdir -p %s
cat > %s
chmod 755 %s
mv %s %s
`, quote(paths.dataDir), quote(paths.upload), quote(paths.upload), quote(paths.upload), quote(paths.binary))

	if _, err := m.runCommand(ctx, target, script, file); err != nil {
		return fmt.Errorf("failed to copy agent binary: %w", err)
	}
	return nil
}

func (m *Manager) runScript(ctx context.Context, target Target, script string) (string, error) {
	return m.runCommand(ctx, target, "sh -s", strings.NewReader(script))
}

func (m *Manager) runCommand(ctx context.Context, target Target, command string, stdin io.Reader) (string, error) {
	client, err := m.connect(ctx, target)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to open ssh session: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	session.Stdin = stdin

	if err := session.Run(command); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if len(message) > 4096 {
			message = message[:4096] + "…"
		}
		if message != "" {
			return stdout.String(), fmt.Errorf("%w: %s", err, message)
		}
		return stdout.String(), err
	}
	return strings.TrimSpace(stdout.String()), nil
}
