// Package provisioning installs and controls agents on remote machines over SSH.
// It uses x/crypto/ssh directly rather than the ssh or scp commands.
package provisioning

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
type Installer struct {
	BinaryPath    string
	KeyFile       string
	KeyPassphrase string
	KnownHosts    string
	RemoteBase    string

	mu sync.Mutex
}

// NewInstaller creates an agent installer. knownHosts should be a writable file that
// the master owns; unknown hosts are trusted on first use.
func NewInstaller(binaryPath, keyFile, knownHosts string) *Installer {
	return &Installer{
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
func (i *Installer) Available() bool {
	if i == nil || i.BinaryPath == "" {
		return false
	}
	info, err := os.Stat(i.BinaryPath)
	return err == nil && !info.IsDir()
}

// Binary returns the configured agent binary path.
func (i *Installer) Binary() string {
	if i == nil {
		return ""
	}
	return i.BinaryPath
}

// Deploy stops any running agent, uploads the current binary, and starts it.
func (i *Installer) Deploy(ctx context.Context, target Target) (string, error) {
	if !i.Available() {
		return "", fmt.Errorf("agent binary not found at %q", i.BinaryPath)
	}
	i.prepare(&target)
	if err := target.Validate(); err != nil {
		return "", err
	}

	if _, err := i.Stop(ctx, target); err != nil {
		return "", fmt.Errorf("failed to stop existing agent: %w", err)
	}
	if err := i.upload(ctx, target); err != nil {
		return "", err
	}
	return i.Start(ctx, target)
}

// Start launches the already-uploaded agent and returns its remote PID.
func (i *Installer) Start(ctx context.Context, target Target) (string, error) {
	i.prepare(&target)
	if err := target.Validate(); err != nil {
		return "", err
	}

	output, err := i.runScript(ctx, target, startScript(target))
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
func (i *Installer) Stop(ctx context.Context, target Target) (bool, error) {
	i.prepare(&target)
	if err := target.Validate(); err != nil {
		return false, err
	}
	output, err := i.runScript(ctx, target, stopScript(target))
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

func (i *Installer) upload(ctx context.Context, target Target) error {
	file, err := os.Open(i.BinaryPath)
	if err != nil {
		return fmt.Errorf("failed to open agent binary: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("agent binary %q is a directory", i.BinaryPath)
	}

	paths := targetPaths(target)
	script := fmt.Sprintf(`set -eu
mkdir -p %s
cat > %s
chmod 755 %s
mv %s %s
`, quote(paths.dataDir), quote(paths.upload), quote(paths.upload), quote(paths.upload), quote(paths.binary))

	if _, err := i.runCommand(ctx, target, script, file); err != nil {
		return fmt.Errorf("failed to copy agent binary: %w", err)
	}
	return nil
}

func (i *Installer) runScript(ctx context.Context, target Target, script string) (string, error) {
	return i.runCommand(ctx, target, "sh -s", strings.NewReader(script))
}

func (i *Installer) runCommand(ctx context.Context, target Target, command string, stdin io.Reader) (string, error) {
	client, err := i.connect(ctx, target)
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
