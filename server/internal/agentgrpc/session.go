// Per-agent stream state. Sends are serialized because gRPC streams must not be written concurrently.

package agentgrpc

import (
	"sync"

	"github.com/ss497254/gloski/proto/agentv1"
)

type session struct {
	deviceID string
	token    string
	stream   agentv1.Agent_ConnectServer
	replaced bool

	sendMu  sync.Mutex
	mu      sync.Mutex
	pending map[string]chan *agentv1.AgentMessage
	terms   map[string]chan []byte
}

// New creates an agent gRPC service backed by the master database.

func (sess *session) send(message *agentv1.MasterMessage) error {
	sess.sendMu.Lock()
	defer sess.sendMu.Unlock()
	return sess.stream.Send(message)
}

func (sess *session) respond(id string, message *agentv1.AgentMessage) {
	sess.mu.Lock()
	channel := sess.pending[id]
	sess.mu.Unlock()
	if channel == nil {
		return
	}
	select {
	case channel <- message:
	default:
	}
}

func (sess *session) terminalOutput(id string, output *agentv1.TerminalOutput) {
	sess.mu.Lock()
	channel := sess.terms[id]
	if output.GetClosed() {
		delete(sess.terms, id)
	}
	sess.mu.Unlock()
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
