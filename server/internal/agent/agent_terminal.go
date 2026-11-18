package agent

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
	"github.com/ss497254/gloski/proto/agentv1"
)

type ptySession struct {
	ptmx   *os.File
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

func (p *ptySession) close() {
	p.cancel()
	_ = p.ptmx.Close()
}

func (a *agentRuntime) openTerminal(id string, request *agentv1.TerminalOpen) {
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
		_ = a.send(&agentv1.AgentMessage{
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

	session := &ptySession{ptmx: ptmx, cmd: cmd, cancel: cancel}
	a.mu.Lock()
	a.terms[id] = session
	a.mu.Unlock()
	defer func() {
		session.close()
		a.mu.Lock()
		delete(a.terms, id)
		a.mu.Unlock()
		_, _ = cmd.Process.Wait()
		_ = a.send(&agentv1.AgentMessage{
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
			if sendErr := a.send(&agentv1.AgentMessage{
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

func (a *agentRuntime) terminalInput(id string, data []byte) {
	a.mu.Lock()
	session := a.terms[id]
	a.mu.Unlock()
	if session == nil {
		return
	}
	if len(data) >= 5 && data[0] == 0x01 {
		cols := int(data[1])<<8 | int(data[2])
		rows := int(data[3])<<8 | int(data[4])
		_ = pty.Setsize(session.ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
		return
	}
	_, _ = session.ptmx.Write(data)
}

func (a *agentRuntime) closeTerminal(id string) {
	a.mu.Lock()
	session := a.terms[id]
	a.mu.Unlock()
	if session != nil {
		session.close()
	}
}
