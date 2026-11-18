package tailscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusParsesPeersAndSelf(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/localapi/v0/status" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"BackendState": "Running",
			"Self": {
				"HostName": "tower",
				"DNSName": "tower.tailnet.ts.net.",
				"OS": "linux",
				"TailscaleIPs": ["100.64.0.1", "fd7a:115c:a1e0::1"],
				"Online": true
			},
			"Peer": {
				"one": {
					"HostName": "worker",
					"DNSName": "worker.tailnet.ts.net.",
					"OS": "linux",
					"TailscaleIPs": ["100.64.0.2"],
					"Online": true
				},
				"two": {
					"HostName": "laptop",
					"DNSName": "laptop.tailnet.ts.net.",
					"OS": "macOS",
					"TailscaleIPs": ["100.64.0.3"],
					"Online": false
				},
				"self": {
					"HostName": "tower",
					"DNSName": "tower.tailnet.ts.net.",
					"OS": "linux",
					"TailscaleIPs": ["100.64.0.1"],
					"Online": true
				}
			}
		}`))
	}))
	defer server.Close()

	client := New(server.URL)
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.BackendState != "Running" {
		t.Fatalf("BackendState = %q", status.BackendState)
	}
	if status.Self.DNSName != "tower.tailnet.ts.net" {
		t.Fatalf("Self.DNSName = %q", status.Self.DNSName)
	}
	if len(status.Peers) != 2 {
		t.Fatalf("Peers length = %d, want 2 (self excluded)", len(status.Peers))
	}
	if status.Peers[0].Hostname != "worker" {
		t.Fatalf("first peer = %q, want online worker first", status.Peers[0].Hostname)
	}
	if status.Peers[0].Self {
		t.Fatal("peer marked as self")
	}
}

func TestStatusUnavailable(t *testing.T) {
	client := New("")
	// A discovered environment socket is fine; force the empty case.
	client.socket = ""
	client.baseURL = ""
	if _, err := client.Status(context.Background()); err != ErrUnavailable {
		t.Fatalf("Status() error = %v, want ErrUnavailable", err)
	}
}

func TestPrimaryIP(t *testing.T) {
	if got := PrimaryIP([]string{"fd7a::1", "100.64.0.5"}); got != "100.64.0.5" {
		t.Fatalf("PrimaryIP() = %q", got)
	}
	if got := PrimaryIP([]string{"fd7a::1"}); got != "fd7a::1" {
		t.Fatalf("PrimaryIP() = %q", got)
	}
	if got := PrimaryIP(nil); got != "" {
		t.Fatalf("PrimaryIP() = %q", got)
	}
}

func TestEndpointURL(t *testing.T) {
	node := Node{DNSName: "worker.tailnet.ts.net.", IPs: []string{"100.64.0.2"}}
	if got := EndpointURL(node, "http", 18081); got != "http://worker.tailnet.ts.net:18081" {
		t.Fatalf("EndpointURL() = %q", got)
	}
	node.DNSName = ""
	if got := EndpointURL(node, "http", 18081); got != "http://100.64.0.2:18081" {
		t.Fatalf("EndpointURL() = %q", got)
	}
}
