package provisioning

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"testing/fstest"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/ss497254/gloski/internal/agentdist"
	"github.com/ss497254/gloski/internal/provisioning/sshtest"
)

// A stand-in agent: answers -v like the real one and otherwise just stays
// alive, which is all the install script needs from it.
const fakeAgent = "#!/bin/sh\n[ \"$1\" = -v ] && exit 0\nexec sleep 300\n"

type sshFixture struct {
	installer *Installer
	target    Target
	server    *sshtest.Server
	dir       string
}

func newSSHFixture(t *testing.T) *sshFixture {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("the install script supports Linux only")
	}
	server, err := sshtest.Start("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })

	temp := t.TempDir()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(private, "")
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(temp, "id_ed25519")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatal(err)
	}

	dist := agentdist.New(fstest.MapFS{
		"gloski-agent-linux-" + runtime.GOARCH: {Data: []byte(fakeAgent)},
	}, "")
	host, portText, _ := net.SplitHostPort(server.Addr())
	port, _ := strconv.Atoi(portText)
	fixture := &sshFixture{
		installer: NewInstaller(dist, keyFile, filepath.Join(temp, "known_hosts")),
		server:    server,
		dir:       filepath.Join(temp, "device", "gloski"),
		target: Target{
			Name: "test box", Host: host, User: "tester", SSHPort: port,
			MasterURL: "http://127.0.0.1:9090", JoinToken: "token", DeployID: "deploy",
		},
	}
	fixture.target.RemoteBase = fixture.dir
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, _ = fixture.installer.Remove(ctx, fixture.target)
	})
	return fixture
}

func (f *sshFixture) install(t *testing.T, force bool) Result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := f.installer.Install(ctx, f.target, force)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	return result
}

func alive(pid string) bool {
	number, err := strconv.Atoi(pid)
	return err == nil && syscall.Kill(number, 0) == nil
}

func TestInstallOverSSH(t *testing.T) {
	f := newSSHFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	first := f.install(t, false)
	if !first.Started || !alive(first.PID) || first.Arch != runtime.GOARCH {
		t.Fatalf("first install = %+v, want a started agent", first)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "agent.json")); err != nil {
		t.Fatalf("agent settings were not written: %v", err)
	}

	// A healthy agent is left alone.
	again := f.install(t, false)
	if again.Started || again.PID != first.PID {
		t.Fatalf("repeat install = %+v, want the same running agent (pid %s)", again, first.PID)
	}

	// A crashed agent is started again.
	pid, _ := strconv.Atoi(first.PID)
	_ = syscall.Kill(pid, syscall.SIGKILL)
	time.Sleep(200 * time.Millisecond)
	revived := f.install(t, false)
	if !revived.Started || revived.PID == first.PID || !alive(revived.PID) {
		t.Fatalf("install after a crash = %+v, want a new agent", revived)
	}

	// Force restarts a healthy agent.
	forced := f.install(t, true)
	if !forced.Started || forced.PID == revived.PID || alive(revived.PID) {
		t.Fatalf("forced install = %+v, want the old agent (pid %s) replaced", forced, revived.PID)
	}

	// A wiped directory is installed from scratch.
	_ = syscall.Kill(mustAtoi(forced.PID), syscall.SIGKILL)
	if err := os.RemoveAll(f.dir); err != nil {
		t.Fatal(err)
	}
	if rebuilt := f.install(t, false); !rebuilt.Started || !alive(rebuilt.PID) {
		t.Fatalf("install after a wipe = %+v", rebuilt)
	}

	stopped, err := f.installer.Stop(ctx, f.target)
	if err != nil || !stopped {
		t.Fatalf("Stop() = %v, %v; want true", stopped, err)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "gloski-agent")); err != nil {
		t.Fatal("Stop() must leave the agent installed")
	}
	if _, err := f.installer.Remove(ctx, f.target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.dir); !os.IsNotExist(err) {
		t.Fatal("Remove() left the install directory behind")
	}
}

func TestReinstallPreservesUnmanagedConfigFields(t *testing.T) {
	f := newSSHFixture(t)
	if result := f.install(t, false); !result.Started {
		t.Fatalf("first install = %+v, want a started agent", result)
	}

	configPath := filepath.Join(f.dir, "agent.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Agent map[string]any `json:"agent"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	config.Agent["port"] = 19999
	config.Agent["heartbeat_interval"] = 7
	edited, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, edited, 0600); err != nil {
		t.Fatal(err)
	}

	if result := f.install(t, true); !result.Started {
		t.Fatalf("reinstall = %+v, want a started agent", result)
	}
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var after struct {
		Agent map[string]any `json:"agent"`
	}
	if err := json.Unmarshal(data, &after); err != nil {
		t.Fatal(err)
	}
	if after.Agent["port"] != float64(19999) || after.Agent["heartbeat_interval"] != float64(7) {
		t.Fatalf("reinstall dropped operator settings: %+v", after.Agent)
	}
	if after.Agent["join_token"] != f.target.JoinToken {
		t.Fatalf("reinstall lost the master's token: %+v", after.Agent)
	}
}

func TestChangedHostKeyIsReportedAndCanBeTrusted(t *testing.T) {
	f := newSSHFixture(t)
	f.install(t, false)

	// The same address now answers with a different host key, as a
	// reinstalled device would.
	address := f.server.Addr()
	_ = f.server.Close()
	replacement, err := sshtest.Start(address)
	if err != nil {
		t.Skipf("could not reuse %s: %v", address, err)
	}
	defer replacement.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := f.installer.Install(ctx, f.target, false); !errors.Is(err, ErrHostKeyChanged) {
		t.Fatalf("Install() error = %v, want ErrHostKeyChanged", err)
	}
	if err := f.installer.ForgetHost(f.target.Host, f.target.SSHPort); err != nil {
		t.Fatal(err)
	}
	if _, err := f.installer.Install(ctx, f.target, false); err != nil {
		t.Fatalf("Install() after trusting the new key: %v", err)
	}
}

func mustAtoi(value string) int {
	number, _ := strconv.Atoi(value)
	return number
}
