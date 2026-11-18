// Package tailnet discovers peers through tailscaled's LocalAPI over a Unix socket.
package tailnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ErrUnavailable means no tailscaled socket or API endpoint is configured.
var ErrUnavailable = errors.New("tailscale local API is not available")

const apiHost = "local-tailscaled.sock"

// Node is one machine in the tailnet.
type Node struct {
	Hostname string   `json:"hostname"`
	DNSName  string   `json:"dns_name"`
	OS       string   `json:"os"`
	IPs      []string `json:"ips"`
	Online   bool     `json:"online"`
	Self     bool     `json:"self,omitempty"`
}

// Status is a small, stable subset of tailscaled's status response.
type Status struct {
	BackendState string `json:"backend_state"`
	Self         Node   `json:"self"`
	Peers        []Node `json:"peers"`
}

// Client queries the local tailscaled API.
type Client struct {
	socket  string
	baseURL string
	http    *http.Client
}

// New creates a client. An empty socket triggers discovery. A socket value
// containing "://" is treated as a direct API base URL (useful for tests).
func New(socket string) *Client {
	if socket == "" {
		socket = DiscoverSocket()
	}

	client := &Client{socket: socket, baseURL: "http://" + apiHost}
	if strings.Contains(socket, "://") {
		client.baseURL = strings.TrimRight(socket, "/")
		client.socket = ""
		client.http = &http.Client{Timeout: 5 * time.Second}
		return client
	}

	dialer := &net.Dialer{Timeout: 3 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			if socket == "" {
				return nil, ErrUnavailable
			}
			return dialer.DialContext(ctx, "unix", socket)
		},
		DisableKeepAlives: true,
	}
	client.http = &http.Client{Transport: transport, Timeout: 5 * time.Second}
	return client
}

// DiscoverSocket finds the common tailscaled sockets used on Linux.
func DiscoverSocket() string {
	if value := strings.TrimSpace(os.Getenv("GLOSKI_TAILSCALE_SOCKET")); value != "" {
		return value
	}

	candidates := []string{
		"/run/tailscale/tailscaled.sock",
		"/var/run/tailscale/tailscaled.sock",
		"/run/tailscale/tailscale.sock",
		"/var/run/tailscale/tailscale.sock",
		"/var/run/tailscaled.sock",
		"/tmp/tailscaled.sock",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode()&os.ModeSocket != 0 {
			return candidate
		}
	}
	return ""
}

// Available reports whether a socket or direct API URL is configured.
func (c *Client) Available() bool {
	return c != nil && (c.socket != "" || c.baseURL != "")
}

// Socket returns the configured socket path, if any.
func (c *Client) Socket() string {
	if c == nil {
		return ""
	}
	return c.socket
}

// Status fetches the current tailnet status.
func (c *Client) Status(ctx context.Context) (*Status, error) {
	if c == nil || !c.Available() {
		return nil, ErrUnavailable
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/localapi/v0/status", nil)
	if err != nil {
		return nil, err
	}
	request.Host = apiHost

	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("tailscaled API request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return nil, fmt.Errorf("tailscaled API returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var payload rawStatus
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode tailscaled status: %w", err)
	}

	status := &Status{
		BackendState: payload.BackendState,
		Self:         payload.Self.node(true),
	}
	status.Peers = make([]Node, 0, len(payload.Peer))
	for _, peer := range payload.Peer {
		node := peer.node(false)
		if node.Hostname == status.Self.Hostname && node.DNSName == status.Self.DNSName {
			continue
		}
		status.Peers = append(status.Peers, node)
	}
	sort.SliceStable(status.Peers, func(i, j int) bool {
		if status.Peers[i].Online != status.Peers[j].Online {
			return status.Peers[i].Online
		}
		return strings.ToLower(status.Peers[i].Hostname) < strings.ToLower(status.Peers[j].Hostname)
	})
	return status, nil
}

// Self fetches this node's tailnet identity.
func (c *Client) Self(ctx context.Context) (*Node, error) {
	status, err := c.Status(ctx)
	if err != nil {
		return nil, err
	}
	self := status.Self
	return &self, nil
}

// PrimaryIP returns the first IPv4 address, falling back to the first address.
func PrimaryIP(ips []string) string {
	for _, ip := range ips {
		if strings.Contains(ip, ".") {
			return ip
		}
	}
	if len(ips) > 0 {
		return ips[0]
	}
	return ""
}

// CleanDNSName removes the trailing dot Tailscale uses in DNS names.
func CleanDNSName(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(name), ".")
}

// KnownHostsPath returns a master-owned known_hosts path for SSH TOFU.
func KnownHostsPath(dataDir string) string {
	if dataDir == "" {
		return ""
	}
	return filepath.Join(dataDir, "known_hosts")
}

type rawStatus struct {
	BackendState string             `json:"BackendState"`
	Self         rawNode            `json:"Self"`
	Peer         map[string]rawNode `json:"Peer"`
}

type rawNode struct {
	HostName     string   `json:"HostName"`
	DNSName      string   `json:"DNSName"`
	OS           string   `json:"OS"`
	TailscaleIPs []string `json:"TailscaleIPs"`
	Online       bool     `json:"Online"`
}

func (n rawNode) node(self bool) Node {
	ips := make([]string, 0, len(n.TailscaleIPs))
	for _, ip := range n.TailscaleIPs {
		if strings.TrimSpace(ip) != "" {
			ips = append(ips, ip)
		}
	}
	return Node{
		Hostname: strings.TrimSpace(n.HostName),
		DNSName:  CleanDNSName(n.DNSName),
		OS:       strings.TrimSpace(n.OS),
		IPs:      ips,
		Online:   n.Online,
		Self:     self,
	}
}
