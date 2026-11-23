package provisioning

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh/agent"
)

func TestAuthMethodsWithoutKeysAreAllowed(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SSH_AUTH_SOCK", "")
	installer := &Installer{}
	methods, cleanup, err := installer.authMethods()
	defer cleanup()
	if err != nil {
		t.Fatalf("authMethods() with no credentials = %v, want no error for Tailscale SSH", err)
	}
	if len(methods) != 0 {
		t.Errorf("authMethods() returned %d methods, want none", len(methods))
	}
}

func TestEmptySSHAgentFallsBackToDiskKey(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "agent.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = agent.ServeAgent(agent.NewKeyring(), conn)
			}()
		}
	}()
	t.Cleanup(func() { _ = listener.Close() })
	t.Setenv("SSH_AUTH_SOCK", socket)

	// The fixture has a key on disk but no identities in the fake agent.
	// Before the empty-agent fix the ssh client gave up before offering it.
	f := newSSHFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := f.installer.Probe(ctx, f.target); err != nil {
		t.Fatalf("Probe with an empty ssh-agent = %v; want the disk key to be used", err)
	}
}
