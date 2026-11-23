package provisioning

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// agentSignerTimeout bounds the master's conversation with ssh-agent while it
// lists identities. Keys on disk are always tried as a fallback.
const agentSignerTimeout = 2 * time.Second

func (i *Installer) connect(ctx context.Context, target Target) (*ssh.Client, error) {
	auth, cleanup, err := i.authMethods()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	hostKey, err := i.hostKeyCallback()
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
		return nil, fmt.Errorf("cannot reach %s over SSH (is the device on and on the tailnet?): %w", addr, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	clientConn, channels, requests, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		var keyErr *knownhosts.KeyError
		switch {
		case errors.As(err, &keyErr):
			return nil, fmt.Errorf("%w: %s presented a different key than last time (it was probably reinstalled)", ErrHostKeyChanged, addr)
		case strings.Contains(err.Error(), "unable to authenticate"):
			return nil, fmt.Errorf("%s refused the master's SSH key for user %q; add the master's public key to that user's authorized_keys", addr, user)
		default:
			return nil, fmt.Errorf("ssh handshake with %s: %w", addr, err)
		}
	}
	return ssh.NewClient(clientConn, channels, requests), nil
}

func (i *Installer) authMethods() ([]ssh.AuthMethod, func(), error) {
	var methods []ssh.AuthMethod
	var closers []io.Closer
	cleanup := func() {
		for _, closer := range closers {
			_ = closer.Close()
		}
	}

	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		conn, err := (&net.Dialer{Timeout: agentSignerTimeout}).Dial("unix", socket)
		if err == nil {
			// An agent with no identities must not be added as an auth
			// method: the ssh client then gives up before trying keys from
			// disk. Ask it for its signers first, with a deadline so a dead
			// agent (or a hardware token waiting for a touch) cannot stall
			// a deploy or a diagnosis. Clear it before the handshake signs.
			_ = conn.SetDeadline(time.Now().Add(agentSignerTimeout))
			signers, signerErr := agent.NewClient(conn).Signers()
			_ = conn.SetDeadline(time.Time{})
			if signerErr == nil && len(signers) > 0 {
				methods = append(methods, ssh.PublicKeys(signers...))
				closers = append(closers, conn)
			} else {
				_ = conn.Close()
			}
		}
	}

	for _, keyPath := range i.keyFiles() {
		signer, err := loadSigner(keyPath, i.KeyPassphrase)
		if err != nil {
			if keyPath == i.KeyFile {
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

func (i *Installer) keyFiles() []string {
	if i.KeyFile != "" {
		return []string{i.KeyFile}
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

func (i *Installer) hostKeyCallback() (ssh.HostKeyCallback, error) {
	knownHosts := i.KnownHosts
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
		// Want is empty only for an unknown host. Trust it once; a real
		// mismatch has entries in Want and must be rejected.
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			return i.trustHost(knownHosts, hostname, key)
		}
		return err
	}, nil
}

func (i *Installer) trustHost(file, hostname string, key ssh.PublicKey) error {
	i.mu.Lock()
	defer i.mu.Unlock()

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

// ErrHostKeyChanged means a device's SSH host key no longer matches the one
// remembered from the first connection.
var ErrHostKeyChanged = errors.New("SSH host key changed")

// ForgetHost drops the remembered host key for a device, so the next
// connection trusts whatever key it presents.
func (i *Installer) ForgetHost(host string, port int) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if port == 0 {
		port = 22
	}
	data, err := os.ReadFile(i.KnownHosts)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	prefix := knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(port))) + " "
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" && !strings.HasPrefix(line, prefix) {
			kept = append(kept, line)
		}
	}
	content := strings.Join(kept, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(i.KnownHosts, []byte(content), 0600)
}
