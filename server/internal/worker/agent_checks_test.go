package worker

import (
	"strings"
	"testing"

	"github.com/ss497254/gloski/proto/agentv1"
)

type checkClient struct {
	agentv1.Agent_ConnectClient
	sent []*agentv1.AgentMessage
}

func (c *checkClient) Send(message *agentv1.AgentMessage) error {
	c.sent = append(c.sent, message)
	return nil
}

func TestHandleCheckSendsTypedResult(t *testing.T) {
	client := &checkClient{}
	runtime := &Runtime{deviceID: "dev", stream: client}
	runtime.handleCheck("req-1", &agentv1.CheckRequest{
		CheckId: "connect_tcp_abc", Kind: "tcp", Target: "127.0.0.1:1",
		Accepted: []int32{200}, TimeoutMs: 1000,
	})
	if len(client.sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(client.sent))
	}
	message := client.sent[0]
	if message.RequestId != "req-1" {
		t.Fatalf("request id = %q, want req-1", message.RequestId)
	}
	result := message.GetCheckResult()
	if result == nil || !strings.HasPrefix(result.GetCheckId(), "connect_tcp_") {
		t.Fatalf("unexpected payload: %+v", message.Payload)
	}
	if result.GetStatus() == "" || result.GetSummary() == "" {
		t.Fatalf("result has no verdict: %+v", result)
	}
}

func TestHandleCheckIgnoresNilRequest(t *testing.T) {
	client := &checkClient{}
	runtime := &Runtime{deviceID: "dev", stream: client}
	runtime.handleCheck("req-2", nil)
	if len(client.sent) != 0 {
		t.Fatalf("nil request sent %d messages", len(client.sent))
	}
}
