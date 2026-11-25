package agentstream

import (
	"context"
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/proto/agentv1"
)

type checkStream struct {
	agentv1.Agent_ConnectServer
	sent chan *agentv1.MasterMessage
}

func (s *checkStream) Send(message *agentv1.MasterMessage) error {
	s.sent <- message
	return nil
}

func checkService(devices ...string) *Service {
	svc := &Service{connections: make(map[string]*connection), stopping: make(map[string]struct{})}
	for _, deviceID := range devices {
		stream := &checkStream{sent: make(chan *agentv1.MasterMessage, 2)}
		svc.connections[deviceID] = &connection{
			deviceID: deviceID,
			stream:   stream,
			pending:  make(map[string]chan *agentv1.AgentMessage),
		}
	}
	return svc
}

func TestCheckRequestResponse(t *testing.T) {
	svc := checkService("dev")
	conn := svc.connections["dev"]
	stream := conn.stream.(*checkStream)

	go func() {
		message := <-stream.sent
		request := message.GetCheckRequest()
		if request == nil {
			return
		}
		conn.respond(message.RequestId, &agentv1.AgentMessage{
			Payload: &agentv1.AgentMessage_CheckResult{
				CheckResult: &agentv1.CheckResult{
					CheckId: request.GetCheckId(), Status: "ok", Summary: "reachable",
					Details: []*agentv1.CheckDetail{{Key: "host", Value: "127.0.0.1"}},
				},
			},
		})
	}()

	definition := checks.Definition{Kind: checks.ProbeTCP, Target: "127.0.0.1:80", Accepted: []int{200}, Interval: time.Minute}
	result, err := svc.Check(context.Background(), "dev", definition)
	if err != nil {
		t.Fatal(err)
	}
	if result.CheckID != definition.ID() || result.DeviceID != "dev" || result.Status != checks.StatusOK {
		t.Fatalf("result = %+v", result)
	}
	if result.Summary != "reachable" || len(result.Details) != 1 || result.Details[0].Key != "host" {
		t.Fatalf("result details = %+v", result)
	}
}

func TestCheckTimesOutAndReportsNoAgent(t *testing.T) {
	old := checkRequestTimeout
	checkRequestTimeout = 100 * time.Millisecond
	defer func() { checkRequestTimeout = old }()

	svc := checkService("dev")
	definition := checks.Definition{Kind: checks.ProbeTCP, Target: "127.0.0.1:80"}
	if _, err := svc.Check(context.Background(), "dev", definition); err == nil {
		t.Fatal("a probe with no answer must time out")
	}
	if _, err := svc.Check(context.Background(), "missing", definition); err == nil {
		t.Fatal("a probe to an unknown device must report no agent")
	}
}

func TestCheckStopsWaitingWhenContextIsCanceled(t *testing.T) {
	svc := checkService("dev")
	stream := svc.connections["dev"].stream.(*checkStream)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := svc.Check(ctx, "dev", checks.Definition{Kind: checks.ProbeTCP, Target: "127.0.0.1:80"})
		done <- err
	}()

	<-stream.sent
	cancel()
	if err := <-done; err != context.Canceled {
		t.Fatalf("Check() error = %v, want context.Canceled", err)
	}
	if got := len(svc.connections["dev"].pending); got != 0 {
		t.Errorf("pending requests = %d, want 0", got)
	}
}
