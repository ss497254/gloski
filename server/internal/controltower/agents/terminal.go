package agents

import (
	"sync"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/proto/agentv1"
)

const (
	// maxTerminalBacklog bounds the output held for a browser that is reading
	// slower than the shell writes.
	maxTerminalBacklog = 4 << 20
	terminalSkipNotice = "\r\n[gloski: output skipped, the browser fell behind]\r\n"
)

// TerminalOutput is the output side of one PTY session. The agent stream
// pushes into it without ever blocking, because that stream also carries
// every other message for the device; the browser connection drains it at
// its own pace. If the backlog grows past the limit the oldest output is
// discarded and a notice takes its place, the way a real terminal skips
// ahead under a flood.
type TerminalOutput struct {
	mu      sync.Mutex
	chunks  [][]byte
	size    int
	skipped bool
	closed  bool
	wake    chan struct{}
}

func newTerminalOutput() *TerminalOutput {
	return &TerminalOutput{wake: make(chan struct{}, 1)}
}

func (o *TerminalOutput) push(data []byte) {
	if len(data) == 0 {
		return
	}
	o.mu.Lock()
	if !o.closed {
		o.chunks = append(o.chunks, data)
		o.size += len(data)
		for o.size > maxTerminalBacklog && len(o.chunks) > 1 {
			o.size -= len(o.chunks[0])
			o.chunks[0] = nil
			o.chunks = o.chunks[1:]
			o.skipped = true
		}
	}
	o.mu.Unlock()
	o.signal()
}

func (o *TerminalOutput) close() {
	o.mu.Lock()
	o.closed = true
	o.mu.Unlock()
	o.signal()
}

func (o *TerminalOutput) signal() {
	select {
	case o.wake <- struct{}{}:
	default:
	}
}

// Next blocks until output is available and returns everything pending as
// one slice. It returns false once the session has ended and been drained,
// or when stop is closed.
func (o *TerminalOutput) Next(stop <-chan struct{}) ([]byte, bool) {
	for {
		o.mu.Lock()
		if len(o.chunks) > 0 {
			data := make([]byte, 0, o.size+len(terminalSkipNotice))
			if o.skipped {
				data = append(data, terminalSkipNotice...)
				o.skipped = false
			}
			for _, chunk := range o.chunks {
				data = append(data, chunk...)
			}
			o.chunks, o.size = nil, 0
			o.mu.Unlock()
			return data, true
		}
		closed := o.closed
		o.mu.Unlock()
		if closed {
			return nil, false
		}
		select {
		case <-o.wake:
		case <-stop:
			return nil, false
		}
	}
}

// OpenTerminal starts a PTY session on an agent and returns its output.
func (svc *Service) OpenTerminal(deviceID, cwd string, cols, rows uint32) (*TerminalOutput, string, error) {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return nil, "", err
	}
	id := uuid.NewString()
	output := newTerminalOutput()
	conn.mu.Lock()
	conn.terms[id] = output
	conn.mu.Unlock()

	message := &agentv1.MasterMessage{
		RequestId: id,
		Payload: &agentv1.MasterMessage_TerminalOpen{
			TerminalOpen: &agentv1.TerminalOpen{Cwd: cwd, Cols: cols, Rows: rows},
		},
	}
	if err := conn.send(message); err != nil {
		conn.mu.Lock()
		delete(conn.terms, id)
		conn.mu.Unlock()
		return nil, "", err
	}
	return output, id, nil
}

// TerminalInput forwards keystrokes to an agent PTY.
func (svc *Service) TerminalInput(deviceID, id string, data []byte) error {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return err
	}
	return conn.send(&agentv1.MasterMessage{
		RequestId: id,
		Payload: &agentv1.MasterMessage_TerminalInput{
			TerminalInput: &agentv1.TerminalInput{Data: data},
		},
	})
}

// ResizeTerminal tells an agent PTY its new window size.
func (svc *Service) ResizeTerminal(deviceID, id string, cols, rows uint32) error {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return err
	}
	return conn.send(&agentv1.MasterMessage{
		RequestId: id,
		Payload: &agentv1.MasterMessage_TerminalResize{
			TerminalResize: &agentv1.TerminalResize{Cols: cols, Rows: rows},
		},
	})
}

// CloseTerminal closes an agent PTY session.
func (svc *Service) CloseTerminal(deviceID, id string) error {
	conn, err := svc.connection(deviceID)
	if err != nil {
		return err
	}
	return conn.send(&agentv1.MasterMessage{
		RequestId: id,
		Payload:   &agentv1.MasterMessage_TerminalClose{TerminalClose: &agentv1.TerminalClose{}},
	})
}
