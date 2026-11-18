package worker

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
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

func (w *Runtime) openTerminal(id string, request *agentv1.TerminalOpen) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, shell)
	if cwd := strings.TrimSpace(request.GetCwd()); cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = w.send(&agentv1.AgentMessage{
			RequestId: id,
			Payload: &agentv1.AgentMessage_TerminalOutput{
				TerminalOutput: &agentv1.TerminalOutput{Data: []byte(err.Error() + "\r\n"), Closed: true},
			},
		})
		cancel()
		return
	}
	if request.GetCols() > 0 && request.GetRows() > 0 {
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(request.GetRows()), Cols: uint16(request.GetCols())})
	}

	session := &PTYSession{ptmx: ptmx, cmd: cmd, cancel: cancel}
	w.mu.Lock()
	w.terms[id] = session
	w.mu.Unlock()
	defer func() {
		session.close()
		w.mu.Lock()
		delete(w.terms, id)
		w.mu.Unlock()
		_, _ = cmd.Process.Wait()
		_ = w.send(&agentv1.AgentMessage{
			RequestId: id,
			Payload: &agentv1.AgentMessage_TerminalOutput{
				TerminalOutput: &agentv1.TerminalOutput{Closed: true},
			},
		})
	}()

	buffer := make([]byte, 16*1024)
	for {
		n, err := ptmx.Read(buffer)
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
	// Browser resize frames are [0x01, cols_hi, cols_lo, rows_hi, rows_lo];
	// all other frames are keystrokes.
	if len(data) >= 5 && data[0] == 0x01 {
		cols := int(data[1])<<8 | int(data[2])
		rows := int(data[3])<<8 | int(data[4])
		_ = pty.Setsize(session.ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
		return
	}
	_, _ = session.ptmx.Write(data)
}

func (w *Runtime) closeTerminal(id string) {
	w.mu.Lock()
	session := w.terms[id]
	w.mu.Unlock()
	if session != nil {
		session.close()
	}
}
