package terminal

import (
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type Terminal struct {
	id     string
	cmd    *exec.Cmd
	ptmx   *os.File
	ws     *websocket.Conn
	mu     sync.Mutex
	closed bool
}

func New(id string, ws *websocket.Conn, shell string, cwd string) (*Terminal, error) {
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	if cwd != "" {
		cmd.Dir = cwd
	}

	// Start command with a pty
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	t := &Terminal{
		id:   id,
		cmd:  cmd,
		ptmx: ptmx,
		ws:   ws,
	}

	return t, nil
}

func (t *Terminal) Run() {
	// Read from PTY and write to WebSocket
	go t.readPTY()

	// Read from WebSocket and write to PTY
	t.readWS()
}

func (t *Terminal) readPTY() {
	buf := make([]byte, 4096)
	for {
		n, err := t.ptmx.Read(buf)
		if err != nil {
			t.Close()
			return
		}

		t.mu.Lock()
		if t.closed {
			t.mu.Unlock()
			return
		}
		err = t.ws.WriteMessage(websocket.BinaryMessage, buf[:n])
		t.mu.Unlock()

		if err != nil {
			t.Close()
			return
		}
	}
}

func (t *Terminal) readWS() {
	for {
		msgType, data, err := t.ws.ReadMessage()
		if err != nil {
			t.Close()
			return
		}

		if msgType == websocket.BinaryMessage || msgType == websocket.TextMessage {
			// Check for resize message (starts with specific prefix)
			if len(data) > 0 && data[0] == 1 {
				// Resize: format is [1, cols (2 bytes), rows (2 bytes)]
				if len(data) >= 5 {
					cols := uint16(data[1])<<8 | uint16(data[2])
					rows := uint16(data[3])<<8 | uint16(data[4])
					t.Resize(cols, rows)
				}
				continue
			}

			// Regular input
			_, err = t.ptmx.Write(data)
			if err != nil {
				t.Close()
				return
			}
		}
	}
}

func (t *Terminal) Resize(cols, rows uint16) error {
	return pty.Setsize(t.ptmx, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})
}

func (t *Terminal) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return
	}
	t.closed = true

	// Close PTY to signal EOF to shell
	t.ptmx.Close()

	// Kill entire process group
	if t.cmd.Process != nil {
		// Negative PID kills the entire process group
		syscall.Kill(-t.cmd.Process.Pid, syscall.SIGTERM)

		// Give processes time to cleanup
		done := make(chan error, 1)
		go func() { done <- t.cmd.Wait() }()

		select {
		case <-done:
			// Clean exit
		case <-time.After(2 * time.Second):
			// Force kill
			syscall.Kill(-t.cmd.Process.Pid, syscall.SIGKILL)
			<-done
		}
	}

	t.ws.Close()
}
