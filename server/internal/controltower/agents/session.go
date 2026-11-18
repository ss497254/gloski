package agents

import (
	"sync"

	"github.com/ss497254/gloski/proto/agentv1"
)

type connection struct {
	deviceID string
	token    string
	stream   agentv1.Agent_ConnectServer
	replaced bool // a newer connection took over this device

	sendMu  sync.Mutex // gRPC streams do not allow concurrent SendMsg
	mu      sync.Mutex
	pending map[string]chan *agentv1.AgentMessage
	terms   map[string]chan []byte
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
	channel := conn.terms[id]
	if output.GetClosed() {
		delete(conn.terms, id)
	}
	conn.mu.Unlock()
	if channel == nil {
		return
	}
	if output.GetClosed() {
		close(channel)
		return
	}
	select {
	case channel <- output.GetData():
	default:
	}
}
