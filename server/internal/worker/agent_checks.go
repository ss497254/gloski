package worker

import (
	"context"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/proto/agentv1"
)

// handleCheck runs one native probe on the device and sends its typed result
// back on the same stream. The master decides what to store.
func (w *Runtime) handleCheck(id string, request *agentv1.CheckRequest) {
	if request == nil {
		return
	}
	definition := checks.Definition{
		Kind:       checks.ProbeKind(request.GetKind()),
		Target:     request.GetTarget(),
		Accepted:   acceptedStatuses(request.GetAccepted()),
		Timeout:    time.Duration(request.GetTimeoutMs()) * time.Millisecond,
		SkipVerify: request.GetSkipVerify(),
	}
	result := definition.Run(context.Background(), w.deviceID)
	details := make([]*agentv1.CheckDetail, 0, len(result.Details))
	for _, detail := range result.Details {
		details = append(details, &agentv1.CheckDetail{Key: detail.Key, Value: detail.Value})
	}
	_ = w.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload: &agentv1.AgentMessage_CheckResult{
			CheckResult: &agentv1.CheckResult{
				CheckId:    result.CheckID,
				Status:     string(result.Status),
				Summary:    result.Summary,
				DurationMs: result.DurationMS,
				Details:    details,
			},
		},
	})
}

func acceptedStatuses(codes []int32) []int {
	accepted := make([]int, 0, len(codes))
	for _, code := range codes {
		accepted = append(accepted, int(code))
	}
	return accepted
}
