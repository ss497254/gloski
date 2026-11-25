package master

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/provisioning"
	"github.com/ss497254/gloski/internal/tailnet"
)

// diagnoseTailscale checks whether the deployment host is a tailnet peer that
// is online. A host the master reaches by another route is not a failure.
// It returns false when the chain should stop here.
func (d *deployHandlers) diagnoseTailscale(ctx context.Context, deployment *fleet.Deployment, result *diagnosis) bool {
	if !d.tailnetClient.Available() {
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
	status, err := d.tailnetClient.Status(waitCtx)
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
func (d *deployHandlers) diagnoseSSH(ctx context.Context, deployment *fleet.Deployment, result *diagnosis) {
	target := d.deploymentTarget(deployment)
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
	state, err := d.installer.Probe(probeCtx, target)
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
