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
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
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

func (m *Manager) connect(ctx context.Context, target Target) (*ssh.Client, error) {
	auth, cleanup, err := m.authMethods()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	hostKey, err := m.hostKeyCallback()
	if err != nil {
		return nil, err
	}

	user := target.User
	if user == "" {
		user = currentUser()
	}
	if user == "" {
		return nil, errors.New("ssh user is required")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: hostKey,
		Timeout:         10 * time.Second,
	}
	addr := net.JoinHostPort(target.Host, strconv.Itoa(target.SSHPort))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("ssh connect to %s: %w", addr, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	clientConn, channels, requests, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ssh handshake with %s: %w", addr, err)
	}
	return ssh.NewClient(clientConn, channels, requests), nil
}

func (m *Manager) authMethods() ([]ssh.AuthMethod, func(), error) {
	var methods []ssh.AuthMethod
	var closers []io.Closer
	cleanup := func() {
		for _, closer := range closers {
			_ = closer.Close()
		}
	}

	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
			closers = append(closers, conn)
		}
	}

	for _, keyPath := range m.keyFiles() {
		signer, err := loadSigner(keyPath, m.KeyPassphrase)
		if err != nil {
			if keyPath == m.KeyFile {
				cleanup()
				return nil, func() {}, fmt.Errorf("failed to load ssh key %s: %w", keyPath, err)
			}
			continue
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	if len(methods) == 0 {
		cleanup()
		return nil, func() {}, errors.New("no ssh credentials found (start ssh-agent or set GLOSKI_MASTER_SSH_KEY_FILE)")
	}
	return methods, cleanup, nil
}

func (m *Manager) prepare(target *Target) {
	if target.RemoteBase == "" {
		target.RemoteBase = m.RemoteBase
	}
}

func (m *Manager) keyFiles() []string {
	if m.KeyFile != "" {
		return []string{m.KeyFile}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
		filepath.Join(home, ".ssh", "id_rsa"),
	}
}

func loadSigner(path, passphrase string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if passphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
	}
	return ssh.ParsePrivateKey(data)
}

func (m *Manager) hostKeyCallback() (ssh.HostKeyCallback, error) {
	knownHosts := m.KnownHosts
	if knownHosts == "" {
		knownHosts = filepath.Join(os.TempDir(), "gloski-known_hosts")
	}
	if err := os.MkdirAll(filepath.Dir(knownHosts), 0700); err != nil {
		return nil, fmt.Errorf("failed to create known_hosts directory: %w", err)
	}
	file, err := os.OpenFile(knownHosts, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create known_hosts file: %w", err)
	}
	_ = file.Close()

	base, err := knownhosts.New(knownHosts)
	if err != nil {
		return nil, fmt.Errorf("failed to read known_hosts: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := base(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			return m.trustHost(knownHosts, hostname, key)
		}
		return err
	}, nil
}

func (m *Manager) trustHost(file, hostname string, key ssh.PublicKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	normalized := knownhosts.Normalize(hostname)
	line := knownhosts.Line([]string{normalized}, key)
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		return err
	}
	return nil
}

type remotePaths struct {
	binary  string
	dataDir string
	envFile string
	pidFile string
	logFile string
	upload  string
}

func targetPaths(target Target) remotePaths {
	base := target.RemoteBase
	if base == "" {
		base = defaultRemoteDir
	}
	binary := path.Join(base, "gloski-agent")
	return remotePaths{
		binary:  binary,
		dataDir: path.Join(base, "gloski-agent-data"),
		envFile: path.Join(base, "gloski-agent.env"),
		pidFile: path.Join(base, "gloski-agent.pid"),
		logFile: path.Join(base, "gloski-agent.log"),
		upload:  binary + ".upload",
	}
}

func startScript(target Target) string {
	paths := targetPaths(target)
	var env strings.Builder
	writeEnv := func(key, value string) {
		env.WriteString(key)
		env.WriteString("=")
		env.WriteString(quote(value))
		env.WriteString("\n")
	}
	writeEnv("GLOSKI_MODE", "agent")
	writeEnv("GLOSKI_AGENT_MASTER_URL", target.MasterURL)
	writeEnv("GLOSKI_AGENT_JOIN_TOKEN", target.JoinToken)
	writeEnv("GLOSKI_AGENT_NAME", target.Name)
	writeEnv("GLOSKI_AGENT_DEPLOY_ID", target.DeployID)
	writeEnv("GLOSKI_DATA_DIR", paths.dataDir)

	return fmt.Sprintf(`set -eu
mkdir -p %s
umask 077
cat > %s <<'GLOSKI_ENV'
%sGLOSKI_ENV
chmod 600 %s
if [ -f %s ]; then
  pid=$(cat %s 2>/dev/null || true)
  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    echo "$pid"
    exit 0
  fi
  rm -f %s
fi
nohup sh -c "set -a; . %s; exec %s" >> %s 2>&1 < /dev/null &
pid=$!
echo "$pid" > %s
echo "$pid"
`,
		quote(paths.dataDir),
		quote(paths.envFile),
		env.String(),
		quote(paths.envFile),
		quote(paths.pidFile),
		quote(paths.pidFile),
		quote(paths.pidFile),
		quote(paths.envFile),
		quote(paths.binary),
		quote(paths.logFile),
		quote(paths.pidFile),
	)
}

func stopScript(target Target) string {
	paths := targetPaths(target)
	return fmt.Sprintf(`set +e
pid=""
if [ -f %s ]; then
  pid=$(cat %s 2>/dev/null)
fi
if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
  kill "$pid" 2>/dev/null
  i=0
  while kill -0 "$pid" 2>/dev/null && [ "$i" -lt 5 ]; do
    sleep 1
    i=$((i+1))
  done
  kill -9 "$pid" 2>/dev/null
  echo stopped
else
  echo not-running
fi
rm -f %s
`,
		quote(paths.pidFile),
		quote(paths.pidFile),
		quote(paths.pidFile),
	)
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func lastLine(value string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

func currentUser() string {
	if user := strings.TrimSpace(os.Getenv("USER")); user != "" {
		return user
	}
	return strings.TrimSpace(os.Getenv("USERNAME"))
}
