package controltower

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/provisioning"
	"github.com/ss497254/gloski/internal/tailnet"
)

// Diagnostic states. Skipped is neutral: a check that does not apply, or one
// that was not needed because an earlier step answered the question, must not
// look like a failure.
const (
	diagOK       = "ok"
	diagWarning  = "warning"
	diagCritical = "critical"
	diagSkipped  = "skipped"
)

const (
	diagnoseDialTimeout = 5 * time.Second
	diagnoseSSHTimeout  = 25 * time.Second
	diagnoseTailnetWait = 5 * time.Second
)

// diagnosisStep is one link in the connection chain. Summary is the factual
// one-liner; Detail holds the raw error when there is one; Advice is what a
// person can do about it.
type diagnosisStep struct {
	Key     string `json:"key"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Detail  string `json:"detail,omitempty"`
	Advice  string `json:"advice,omitempty"`
}

// diagnosis is the result of walking a device's connection chain. Steps are in
// cause order; the first critical step is the headline.
type diagnosis struct {
	DeviceID    string          `json:"device_id"`
	Device      string          `json:"device"`
	Status      string          `json:"status"`
	Headline    string          `json:"headline"`
	Advice      string          `json:"advice,omitempty"`
	Steps       []diagnosisStep `json:"steps"`
	GeneratedAt time.Time       `json:"generated_at"`
}

func (d *diagnosis) add(step diagnosisStep) {
	d.Steps = append(d.Steps, step)
}

// finish picks the overall status and the headline. Critical beats warning;
// skipped steps never become a verdict on their own.
func (d *diagnosis) finish() {
	for _, wanted := range []string{diagCritical, diagWarning} {
		for _, step := range d.Steps {
			if step.Status == wanted {
				d.Status = wanted
				d.Headline = step.Summary
				d.Advice = step.Advice
				return
			}
		}
	}
	d.Status = diagOK
	d.Headline = "The agent is connected and runs the master's build."
}

func (t *Tower) handleDiagnoseDevice(w http.ResponseWriter, r *http.Request) {
	device, err := t.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	httpapi.SuccessWithMessage(w, t.diagnoseDevice(r.Context(), device))
}

// diagnoseDevice walks the chain from the master outwards and stops at the
// first failure. When the agent is connected the SSH links are not needed, so
// they are left out instead of being reported as unknown.
func (t *Tower) diagnoseDevice(ctx context.Context, device *fleet.Device) diagnosis {
	result := diagnosis{
		DeviceID:    device.ID,
		Device:      device.Name,
		Steps:       []diagnosisStep{},
		GeneratedAt: time.Now().UTC(),
	}
	deployment := device.Deploy
	connected := t.agentService.Connected(device.ID)

	if deployment == nil {
		result.add(diagnosisStep{
			Key:     "deployment",
			Title:   "Deployment record",
			Status:  diagSkipped,
			Summary: "This device was not deployed by the master.",
			Detail:  "The master has no SSH details, so it cannot repair this agent itself.",
			Advice:  "Redeploy the device from the dashboard so the master can manage its agent.",
		})
	} else {
		step := diagnosisStep{
			Key:     "deployment",
			Title:   "Deployment record",
			Status:  diagOK,
			Summary: fmt.Sprintf("The master installs this agent over SSH at %s:%d.", deployment.Host, sshPort(deployment)),
		}
		if deployment.Error != "" {
			step.Detail = "Last recorded error: " + firstLine(deployment.Error, 500)
		}
		result.add(step)
	}

	if deployment != nil && deployment.Status == fleet.DeployStopped {
		result.add(diagnosisStep{
			Key:     "agent",
			Title:   "Agent session",
			Status:  diagWarning,
			Summary: "The agent is stopped on purpose.",
			Advice:  "Press Start agent to run it again.",
		})
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagSkipped,
			Summary: "Not checked: the agent is stopped.",
		})
		result.finish()
		return result
	}

	if connected {
		result.add(diagnosisStep{
			Key:     "agent",
			Title:   "Agent session",
			Status:  diagOK,
			Summary: "The agent has a live session with the master.",
		})
		t.diagnoseBuild(device, &result)
		result.finish()
		return result
	}

	if deployment != nil {
		if t.diagnoseTailscale(ctx, deployment, &result) {
			t.diagnoseSSH(ctx, deployment, &result)
		} else {
			skipSSHChain(&result, "the device is offline in the tailnet")
		}
	}

	// Everything above was healthy, or could not be checked, yet the agent is
	// not connected. This step names the remaining problem and what to do.
	summary := "The agent has no live session with the master."
	detail := fmt.Sprintf("The master listens for agents at %s.", t.agentGRPCURL())
	if !device.LastSeen.IsZero() {
		detail += fmt.Sprintf(" The device was last seen %s.", time.Since(device.LastSeen).Round(time.Second))
	}
	advice := fmt.Sprintf("Check that %s can reach %s, then press Start agent.", device.Name, t.agentGRPCURL())
	if deployment == nil {
		advice = "Redeploy the device from the dashboard so the master can install and start its agent."
	}
	if rejection := t.agentService.Rejection(device.Name); rejection != "" {
		summary = "The master refused the agent's connection."
		detail = "The master turned the agent away: " + rejection + "."
		advice = "Press Start agent so the master installs a compatible build."
	}
	result.add(diagnosisStep{
		Key:     "agent",
		Title:   "Agent session",
		Status:  diagCritical,
		Summary: summary,
		Detail:  detail,
		Advice:  advice,
	})
	result.add(diagnosisStep{
		Key:     "build",
		Title:   "Agent build",
		Status:  diagSkipped,
		Summary: "Not checked: the agent is not connected.",
	})
	result.finish()
	return result
}

// diagnoseBuild compares the connected agent's binary with the one the master
// would install. An old build is a warning, not a failure: the agent works, and
// the lifecycle will replace it.
func (t *Tower) diagnoseBuild(device *fleet.Device, result *diagnosis) {
	arch, hash, connected := t.agentService.Build(device.ID)
	if !connected {
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagSkipped,
			Summary: "Not checked: the agent is not connected.",
		})
		return
	}
	if arch == "" || hash == "" {
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagWarning,
			Summary: "The agent did not report which build it runs.",
			Advice:  "Press Restart agent to reinstall it.",
		})
		return
	}
	build := t.dist.Build(arch)
	if build == nil {
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagWarning,
			Summary: fmt.Sprintf("The master has no agent build for linux/%s.", arch),
			Advice:  "Run 'make build' on the master.",
		})
		return
	}
	if hash != build.SHA256 {
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagWarning,
			Summary: "The agent runs an older build than the master.",
			Advice:  "The master replaces it automatically when the device has no running jobs; press Restart agent to do it now.",
		})
		return
	}
	result.add(diagnosisStep{
		Key:     "build",
		Title:   "Agent build",
		Status:  diagOK,
		Summary: "The agent runs the master's build.",
	})
}

// diagnoseTailscale checks whether the deployment host is a tailnet peer that
// is online. A host the master reaches by another route is not a failure.
// It returns false when the chain should stop here.
func (t *Tower) diagnoseTailscale(ctx context.Context, deployment *fleet.Deployment, result *diagnosis) bool {
	if !t.tailnetClient.Available() {
		result.add(diagnosisStep{
			Key:     "tailscale",
			Title:   "Tailscale",
			Status:  diagSkipped,
			Summary: "Tailscale is not running on the master, so the tailnet was not checked.",
		})
		return true
	}
	if isLoopbackHost(deployment.Host) {
		result.add(diagnosisStep{
			Key:     "tailscale",
			Title:   "Tailscale",
			Status:  diagSkipped,
			Summary: "Not checked: the device is the master itself.",
		})
		return true
	}

	waitCtx, cancel := context.WithTimeout(ctx, diagnoseTailnetWait)
	defer cancel()
	status, err := t.tailnetClient.Status(waitCtx)
	if err != nil {
		result.add(diagnosisStep{
			Key:     "tailscale",
			Title:   "Tailscale",
			Status:  diagSkipped,
			Summary: "The master could not read the tailnet status.",
			Detail:  firstLine(err.Error(), maxIssueDetail),
		})
		return true
	}

	peer, found := findTailnetPeer(status.Peers, deployment.Host)
	switch {
	case found && peer.Online:
		result.add(diagnosisStep{
			Key:     "tailscale",
			Title:   "Tailscale",
			Status:  diagOK,
			Summary: fmt.Sprintf("Tailscale sees %s online.", peerLabel(peer)),
		})
		return true
	case found:
		result.add(diagnosisStep{
			Key:     "tailscale",
			Title:   "Tailscale",
			Status:  diagCritical,
			Summary: fmt.Sprintf("Tailscale sees %s but it is offline.", peerLabel(peer)),
			Advice:  "Power the device on and check that Tailscale is running on it.",
		})
		return false
	default:
		result.add(diagnosisStep{
			Key:     "tailscale",
			Title:   "Tailscale",
			Status:  diagWarning,
			Summary: fmt.Sprintf("%s is not a peer in the master's tailnet.", deployment.Host),
			Detail:  "The master will try SSH anyway; the device may be reached by another route.",
			Advice:  "If the device should be on the tailnet, check Tailscale on both machines.",
		})
		return true
	}
}

// diagnoseSSH walks the links the master controls: TCP, login, install and
// process. It stops adding links once one of them fails.
func (t *Tower) diagnoseSSH(ctx context.Context, deployment *fleet.Deployment, result *diagnosis) {
	target := t.deploymentTarget(deployment)
	// Validate here, not only inside Probe: it fills in the default SSH port
	// and install directory that the step summaries also need.
	if err := target.Validate(); err != nil {
		result.add(diagnosisStep{
			Key:     "ssh",
			Title:   "SSH port",
			Status:  diagCritical,
			Summary: "The master's SSH settings for this device are invalid.",
			Detail:  err.Error(),
			Advice:  "Remove and redeploy the device from the dashboard.",
		})
		skipRemoteSteps(result, "the deployment settings are invalid")
		return
	}
	addr := net.JoinHostPort(target.Host, strconv.Itoa(target.SSHPort))

	if err := dialSSH(ctx, addr); err != nil {
		advice := "Check that the device is powered on and on the network, and that sshd is running."
		if strings.Contains(err.Error(), "connection refused") {
			advice = "The device answered but refused the connection. Check that sshd is running and listens on this port."
		}
		result.add(diagnosisStep{
			Key:     "ssh",
			Title:   "SSH port",
			Status:  diagCritical,
			Summary: fmt.Sprintf("The master cannot open a TCP connection to %s.", addr),
			Detail:  err.Error(),
			Advice:  advice,
		})
		skipRemoteSteps(result, "the SSH port is unreachable")
		return
	}
	result.add(diagnosisStep{
		Key:     "ssh",
		Title:   "SSH port",
		Status:  diagOK,
		Summary: fmt.Sprintf("%s accepts TCP connections.", addr),
	})

	probeCtx, cancel := context.WithTimeout(ctx, diagnoseSSHTimeout)
	defer cancel()
	state, err := t.installer.Probe(probeCtx, target)
	if err != nil {
		result.add(diagnosisStep{
			Key:     "login",
			Title:   "SSH login",
			Status:  diagCritical,
			Summary: fmt.Sprintf("The master cannot log in to %s over SSH.", addr),
			Detail:  err.Error(),
			Advice:  sshLoginAdvice(target, err),
		})
		skipInstallAndProcess(result, "SSH login failed")
		return
	}
	result.add(diagnosisStep{
		Key:     "login",
		Title:   "SSH login",
		Status:  diagOK,
		Summary: fmt.Sprintf("The master logged in to %s as %s.", addr, targetUser(target)),
	})

	installed := true
	switch {
	case !state.Dir:
		installed = false
		result.add(diagnosisStep{
			Key:     "install",
			Title:   "Agent install",
			Status:  diagCritical,
			Summary: "The agent is not installed on the device.",
			Detail:  fmt.Sprintf("The install directory %s does not exist.", target.RemoteBase),
			Advice:  "Press Start agent to install it.",
		})
	case !state.Binary:
		installed = false
		result.add(diagnosisStep{
			Key:     "install",
			Title:   "Agent install",
			Status:  diagCritical,
			Summary: fmt.Sprintf("The agent binary is missing from %s.", target.RemoteBase),
			Advice:  "Press Start agent to reinstall it.",
		})
	case !state.Runnable:
		installed = false
		result.add(diagnosisStep{
			Key:     "install",
			Title:   "Agent install",
			Status:  diagCritical,
			Summary: fmt.Sprintf("The device cannot run programs from %s.", target.RemoteBase),
			Detail:  "The agent file exists but did not run. The directory may be mounted noexec, or the CPU type was detected wrongly.",
			Advice:  "Move the install directory to an executable filesystem, then press Start agent.",
		})
	default:
		result.add(diagnosisStep{
			Key:     "install",
			Title:   "Agent install",
			Status:  diagOK,
			Summary: fmt.Sprintf("The agent is installed in %s and can run.", target.RemoteBase),
		})
	}

	switch {
	case state.Running:
		result.add(diagnosisStep{
			Key:     "process",
			Title:   "Agent process",
			Status:  diagOK,
			Summary: fmt.Sprintf("The agent process is running (pid %s).", state.PID),
		})
	case installed:
		result.add(diagnosisStep{
			Key:     "process",
			Title:   "Agent process",
			Status:  diagCritical,
			Summary: "The agent is installed but not running.",
			Advice:  "Press Start agent to start it.",
		})
	default:
		result.add(diagnosisStep{
			Key:     "process",
			Title:   "Agent process",
			Status:  diagSkipped,
			Summary: "Not checked: the agent is not installed.",
		})
	}
}

func skipRemoteSteps(result *diagnosis, reason string) {
	result.add(diagnosisStep{
		Key:     "login",
		Title:   "SSH login",
		Status:  diagSkipped,
		Summary: "Not checked: " + reason + ".",
	})
	skipInstallAndProcess(result, reason)
}

func skipInstallAndProcess(result *diagnosis, reason string) {
	for _, step := range []struct{ key, title string }{
		{"install", "Agent install"},
		{"process", "Agent process"},
	} {
		result.add(diagnosisStep{
			Key:     step.key,
			Title:   step.title,
			Status:  diagSkipped,
			Summary: "Not checked: " + reason + ".",
		})
	}
}

func skipSSHChain(result *diagnosis, reason string) {
	result.add(diagnosisStep{
		Key:     "ssh",
		Title:   "SSH port",
		Status:  diagSkipped,
		Summary: "Not checked: " + reason + ".",
	})
	skipRemoteSteps(result, reason)
}

func dialSSH(ctx context.Context, addr string) error {
	dialer := net.Dialer{Timeout: diagnoseDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return conn.Close()
}

func sshLoginAdvice(target provisioning.Target, err error) string {
	switch {
	case errors.Is(err, provisioning.ErrHostKeyChanged):
		return "The device's SSH host key changed (it was probably reinstalled). Press Trust new key to accept it."
	case strings.Contains(err.Error(), "refused SSH login"):
		return fmt.Sprintf("Enable Tailscale SSH on the device, or add the master's public key to %s's authorized_keys.", targetUser(target))
	case strings.Contains(err.Error(), "no ssh credentials"):
		return "This master build predates key-less Tailscale SSH; restart or update the master."
	default:
		return "Check the SSH user, key and host key on the device."
	}
}

func sshPort(deployment *fleet.Deployment) int {
	if deployment.SSHPort == 0 {
		return 22
	}
	return deployment.SSHPort
}

func targetUser(target provisioning.Target) string {
	if target.User != "" {
		return target.User
	}
	if user := strings.TrimSpace(os.Getenv("USER")); user != "" {
		return user
	}
	return "the SSH user"
}

func isLoopbackHost(host string) bool {
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// findTailnetPeer matches a deployment host against a peer's addresses and
// names. Hosts may be an IP, a MagicDNS name or a short hostname.
func findTailnetPeer(peers []tailnet.Node, host string) (tailnet.Node, bool) {
	host = strings.TrimSuffix(strings.Trim(host, "[]"), ".")
	for _, peer := range peers {
		if strings.EqualFold(strings.TrimSuffix(peer.Hostname, "."), host) ||
			strings.EqualFold(strings.TrimSuffix(peer.DNSName, "."), host) {
			return peer, true
		}
		for _, ip := range peer.IPs {
			if ip == host {
				return peer, true
			}
		}
	}
	return tailnet.Node{}, false
}

func peerLabel(peer tailnet.Node) string {
	if peer.DNSName != "" {
		return tailnet.CleanDNSName(peer.DNSName)
	}
	if peer.Hostname != "" {
		return peer.Hostname
	}
	if ip := tailnet.PrimaryIP(peer.IPs); ip != "" {
		return ip
	}
	return "the device"
}
