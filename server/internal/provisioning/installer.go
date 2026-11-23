// Package provisioning installs and controls agents on remote machines over SSH.
// It uses x/crypto/ssh directly rather than the ssh or scp commands.
package provisioning

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/ss497254/gloski/internal/agentdist"
)

const (
	defaultRemoteDir = agentdist.DefaultDir
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

// Installer installs and controls agents over SSH. It runs the same install
// and uninstall scripts the master serves over HTTP.
type Installer struct {
	Dist          *agentdist.Dist
	KeyFile       string
	KeyPassphrase string
	KnownHosts    string

	mu sync.Mutex // guards writes to the known-hosts file
}

// NewInstaller creates an agent installer. knownHosts should be a writable file that
// the master owns; unknown hosts are trusted on first use.
func NewInstaller(dist *agentdist.Dist, keyFile, knownHosts string) *Installer {
	return &Installer{Dist: dist, KeyFile: keyFile, KnownHosts: knownHosts}
}

// Available reports whether there is an agent build to install.
func (i *Installer) Available() bool {
	return i != nil && i.Dist != nil && i.Dist.Available()
}

// Result describes a finished install.
type Result struct {
	PID  string
	Arch string
	// Started is false when a healthy agent was already running.
	Started bool
}

// Install makes sure the agent is installed, current and running. It is safe
// to call on a device that already has a healthy agent, which it leaves alone
// unless force is set.
func (i *Installer) Install(ctx context.Context, target Target, force bool) (Result, error) {
	if !i.Available() {
		return Result{}, errors.New("the master has no agent builds to install (run 'make build')")
	}
	if err := target.Validate(); err != nil {
		return Result{}, err
	}
	params := agentdist.InstallParams{
		Dir:       target.RemoteBase,
		MasterURL: target.MasterURL,
		JoinToken: target.JoinToken,
		Name:      target.Name,
		DeployID:  target.DeployID,
		Force:     force,
	}
	params.ConfigJSON = i.mergedConfig(ctx, target, map[string]any{
		"master_url": target.MasterURL,
		"join_token": target.JoinToken,
		"name":       target.Name,
		"deploy_id":  target.DeployID,
	})
	script := i.Dist.InstallScript(params)

	output, err := i.runScript(ctx, target, script)
	if arch := field(output, "GLOSKI_NEED_UPLOAD", "arch"); err != nil && arch != "" {
		// The device lacks the current build. Copy it over and run again.
		build := i.Dist.Build(arch)
		if build == nil {
			return Result{}, fmt.Errorf("the master has no agent build for linux/%s", arch)
		}
		if err := i.upload(ctx, target, build.Data); err != nil {
			return Result{}, err
		}
		output, err = i.runScript(ctx, target, script)
	}
	if err != nil {
		return Result{}, err
	}
	result := Result{
		PID:     field(output, "GLOSKI_INSTALLED", "pid"),
		Arch:    field(output, "GLOSKI_INSTALLED", "arch"),
		Started: field(output, "GLOSKI_INSTALLED", "started") == "yes",
	}
	if result.PID == "" {
		return Result{}, fmt.Errorf("unexpected installer output %q", output)
	}
	return result, nil
}

// mergedConfig overlays the fields the master owns on the config already on
// the device. Reinstalling must not wipe settings nobody asked us to change,
// such as a custom agent port or heartbeat interval. A device we cannot read
// simply gets the supplied fields.
func (i *Installer) mergedConfig(ctx context.Context, target Target, supplied map[string]any) string {
	configPath := path.Join(target.RemoteBase, "agent.json")
	output, err := i.runCommand(ctx, target, "cat "+quote(configPath)+" 2>/dev/null || true", nil)
	existing := map[string]any{}
	if err == nil && strings.TrimSpace(output) != "" {
		// A corrupt file is treated as absent; the master's fields win.
		_ = json.Unmarshal([]byte(output), &existing)
	}
	agent, _ := existing["agent"].(map[string]any)
	if agent == nil {
		agent = map[string]any{}
	}
	for key, value := range supplied {
		agent[key] = value
	}
	if len(existing) == 0 {
		existing = map[string]any{"agent": agent}
	} else {
		existing["agent"] = agent
	}
	merged, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		merged, _ = json.Marshal(map[string]any{"agent": supplied})
	}
	return string(merged)
}

// Stop stops the remote agent and its self-heal but leaves it installed. The
// boolean reports whether a process was actually stopped.
func (i *Installer) Stop(ctx context.Context, target Target) (bool, error) {
	return i.uninstall(ctx, target, true, "GLOSKI_STOPPED")
}

// Remove stops the remote agent and deletes everything it installed.
func (i *Installer) Remove(ctx context.Context, target Target) (bool, error) {
	return i.uninstall(ctx, target, false, "GLOSKI_UNINSTALLED")
}

func (i *Installer) uninstall(ctx context.Context, target Target, stopOnly bool, marker string) (bool, error) {
	if err := target.Validate(); err != nil {
		return false, err
	}
	output, err := i.runScript(ctx, target, agentdist.UninstallScript(target.RemoteBase, stopOnly))
	if err != nil {
		return false, err
	}
	switch field(output, marker, "state") {
	case "stopped":
		return true, nil
	case "not-running":
		return false, nil
	default:
		return false, fmt.Errorf("unexpected uninstaller output %q", output)
	}
}

func (i *Installer) upload(ctx context.Context, target Target, binary []byte) error {
	destination := path.Join(target.RemoteBase, "gloski-agent.upload")
	command := fmt.Sprintf("mkdir -p %s && cat > %s", quote(target.RemoteBase), quote(destination))
	if _, err := i.runCommand(ctx, target, command, bytes.NewReader(binary)); err != nil {
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
