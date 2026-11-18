// SSH transport: key or agent authentication, host-key trust on first use, and remote command execution.

package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

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
