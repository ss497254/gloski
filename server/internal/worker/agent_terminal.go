package worker

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/proto/agentv1"
)

type PTYSession struct {
	ptmx   *os.File
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

func (p *PTYSession) close() {
	p.cancel()
	_ = p.ptmx.Close()
}

// openTerminal starts a shell on a PTY. The session is registered before
// this returns, so the input and resize messages that follow on the stream
// always find it; only the output pump runs in the background.
func (w *Runtime) openTerminal(id string, request *agentv1.TerminalOpen) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, shell)
	if cwd := strings.TrimSpace(request.GetCwd()); cwd != "" {
		if expanded, err := files.ExpandTilde(cwd); err == nil {
			cwd = expanded
		}
		// A stale or mistyped directory should not prevent the shell from
		// opening; it starts in the agent's working directory instead.
		if info, err := os.Stat(cwd); err == nil && info.IsDir() {
			cmd.Dir = cwd
		}
	}
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	size := &pty.Winsize{Rows: 24, Cols: 80}
	if cols, rows := request.GetCols(), request.GetRows(); cols > 0 && rows > 0 && cols <= 0xffff && rows <= 0xffff {
		size = &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)}
	}
	ptmx, err := pty.StartWithSize(cmd, size)
	if err != nil {
		cancel()
		_ = w.send(&agentv1.AgentMessage{
			RequestId: id,
			Payload: &agentv1.AgentMessage_TerminalOutput{
				TerminalOutput: &agentv1.TerminalOutput{Data: []byte(err.Error() + "\r\n"), Closed: true},
			},
		})
		return
	}

	session := &PTYSession{ptmx: ptmx, cmd: cmd, cancel: cancel}
	w.mu.Lock()
	w.terms[id] = session
	w.mu.Unlock()

	go w.pumpTerminal(id, session)
}

// pumpTerminal forwards PTY output until the shell exits or the stream fails.
func (w *Runtime) pumpTerminal(id string, session *PTYSession) {
	defer func() {
		session.close()
		w.mu.Lock()
		if w.terms[id] == session {
			delete(w.terms, id)
		}
		w.mu.Unlock()
		_, _ = session.cmd.Process.Wait()
		_ = w.send(&agentv1.AgentMessage{
			RequestId: id,
			Payload: &agentv1.AgentMessage_TerminalOutput{
				TerminalOutput: &agentv1.TerminalOutput{Closed: true},
			},
		})
	}()

	buffer := make([]byte, 16*1024)
	for {
		n, err := session.ptmx.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			if sendErr := w.send(&agentv1.AgentMessage{
				RequestId: id,
				Payload: &agentv1.AgentMessage_TerminalOutput{
					TerminalOutput: &agentv1.TerminalOutput{Data: chunk},
				},
			}); sendErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (w *Runtime) terminalInput(id string, data []byte) {
	w.mu.Lock()
	session := w.terms[id]
	w.mu.Unlock()
	if session == nil {
		return
	}
	_, _ = session.ptmx.Write(data)
}

func (w *Runtime) resizeTerminal(id string, cols, rows uint32) {
	w.mu.Lock()
	session := w.terms[id]
	w.mu.Unlock()
	if session == nil || cols == 0 || rows == 0 || cols > 0xffff || rows > 0xffff {
		return
	}
	_ = pty.Setsize(session.ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

func (w *Runtime) closeTerminal(id string) {
	w.mu.Lock()
	session := w.terms[id]
	w.mu.Unlock()
	if session != nil {
		session.close()
	}
}
