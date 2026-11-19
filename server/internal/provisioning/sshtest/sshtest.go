// Package sshtest is a small SSH server for tests. It accepts any public key
// and runs each requested command with sh on the local machine, which is
// enough to stand in for a device when testing installs.
package sshtest

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os/exec"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Server is a running test SSH server.
type Server struct {
	listener net.Listener
}

// Start listens on addr (use "127.0.0.1:0" for any free port) with a freshly
// generated host key, so every Start looks like a reinstalled machine.
func Start(addr string) (*Server, error) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	hostKey, err := ssh.NewSignerFromKey(private)
	if err != nil {
		return nil, err
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
	}
	config.AddHostKey(hostKey)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server := &Server{listener: listener}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serve(conn, config)
		}
	}()
	return server, nil
}

// Addr is the address the server listens on.
func (s *Server) Addr() string { return s.listener.Addr().String() }

// Close stops accepting connections.
func (s *Server) Close() error { return s.listener.Close() }

func serve(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		_ = conn.Close()
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(requests)

	for newChannel := range channels {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "only sessions are supported")
			continue
		}
		channel, channelRequests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go session(channel, channelRequests)
	}
}

func session(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()
	for request := range requests {
		if request.Type != "exec" {
			_ = request.Reply(false, nil)
			continue
		}
		var payload struct{ Command string }
		if err := ssh.Unmarshal(request.Payload, &payload); err != nil {
			_ = request.Reply(false, nil)
			return
		}
		_ = request.Reply(true, nil)

		command := exec.Command("sh", "-c", payload.Command)
		stdin, _ := command.StdinPipe()
		stdout, _ := command.StdoutPipe()
		stderr, _ := command.StderrPipe()
		status := uint32(0)
		if err := command.Start(); err != nil {
			status = 127
		} else {
			go func() {
				_, _ = io.Copy(stdin, channel)
				_ = stdin.Close()
			}()
			var copies sync.WaitGroup
			copies.Add(2)
			go func() { defer copies.Done(); _, _ = io.Copy(channel, stdout) }()
			go func() { defer copies.Done(); _, _ = io.Copy(channel.Stderr(), stderr) }()
			copies.Wait()
			if err := command.Wait(); err != nil {
				status = 1
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) && exitErr.ExitCode() >= 0 {
					status = uint32(exitErr.ExitCode())
				}
			}
		}
		_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
		return
	}
}
