package agents

import (
	"sync"
	"sync/atomic"

	"github.com/ss497254/gloski/proto/agentv1"
)

type connection struct {
	deviceID string
	token    string
	stream   agentv1.Agent_ConnectServer
	// arch and buildHash describe the agent binary on the other end.
	arch      string
	buildHash string

	// replaced is set when a newer connection took over this device, so this
	// one's teardown must not mark the device offline or abort its jobs.
	replaced atomic.Bool
	// closed ends the stream handler; closing it is how the master hangs up.
	closed    chan struct{}
	closeOnce sync.Once

	sendMu  sync.Mutex // gRPC streams do not allow concurrent SendMsg
	mu      sync.Mutex
	pending map[string]chan *agentv1.AgentMessage
	terms   map[string]*TerminalOutput
}

// close makes the stream handler return, which ends the RPC for the agent.
func (conn *connection) close() {
	conn.closeOnce.Do(func() { close(conn.closed) })
}

func (conn *connection) send(message *agentv1.MasterMessage) error {
	conn.sendMu.Lock()
	defer conn.sendMu.Unlock()
	return conn.stream.Send(message)
}

func (conn *connection) respond(id string, message *agentv1.AgentMessage) {
	conn.mu.Lock()
	channel := conn.pending[id]
	conn.mu.Unlock()
	if channel == nil {
		return
	}
	select {
	case channel <- message:
	default:
	}
}

func (conn *connection) terminalOutput(id string, output *agentv1.TerminalOutput) {
	conn.mu.Lock()
	terminal := conn.terms[id]
	if output.GetClosed() {
		delete(conn.terms, id)
	}
	conn.mu.Unlock()
	if terminal == nil {
		return
	}
	terminal.push(output.GetData())
	if output.GetClosed() {
		terminal.close()
	}
}
