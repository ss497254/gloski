package checks

import (
	"fmt"
	"sync"
	"time"
)

const (
	rawSampleLimit    = 61
	minuteBucketLimit = 61
	recentWindow      = 5
)

type aggregateSample struct {
	at     time.Time
	status Status
}

// Aggregator keeps a bounded short-term view of one probe series. Raw samples
// are kept for 61 runs and minute buckets for 61 minutes; only the aggregated
// result is ever written to check_results.
type Aggregator struct {
	mu         sync.Mutex
	raw        []aggregateSample
	minutes    []aggregateSample
	lastMinute time.Time
}

// Add records one probe run and returns the verdict for this interval. A
// single critical sample in the recent window dominates, so a service that
// flaps still looks broken. Never-run and skipped samples are neutral.
func (a *Aggregator) Add(result Result, now time.Time) Result {
	a.mu.Lock()
	defer a.mu.Unlock()
	now = now.UTC()

	a.raw = append(a.raw, aggregateSample{at: now, status: result.Status})
	if len(a.raw) > rawSampleLimit {
		a.raw = append([]aggregateSample(nil), a.raw[len(a.raw)-rawSampleLimit:]...)
	}
	minute := now.Truncate(time.Minute)
	if !minute.Equal(a.lastMinute) {
		a.minutes = append(a.minutes, aggregateSample{at: minute, status: result.Status})
		if len(a.minutes) > minuteBucketLimit {
			a.minutes = append([]aggregateSample(nil), a.minutes[len(a.minutes)-minuteBucketLimit:]...)
		}
		a.lastMinute = minute
	}

	start := len(a.raw) - recentWindow
	if start < 0 {
		start = 0
	}
	critical, warning := 0, 0
	for _, sample := range a.raw[start:] {
		switch sample.status {
		case StatusCritical:
			critical++
		case StatusWarning:
			warning++
		}
	}

	verdict := StatusOK
	switch {
	case critical > 0:
		verdict = StatusCritical
	case warning > 0:
		verdict = StatusWarning
	}
	failures := critical + warning
	aggregated := result
	aggregated.Status = verdict
	if failures > 0 {
		aggregated.Summary = fmt.Sprintf("%s (%d/%d recent runs failed)", result.Summary, failures, len(a.raw)-start)
	}
	aggregated.Details = append(append([]Detail{}, result.Details...),
		Detail{Key: "recent_runs", Value: fmt.Sprintf("%d", len(a.raw)-start)},
		Detail{Key: "recent_failures", Value: fmt.Sprintf("%d", failures)},
		Detail{Key: "minute_buckets", Value: fmt.Sprintf("%d", len(a.minutes))},
	)
	return aggregated
}

func (a *Aggregator) sizes() (int, int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.raw), len(a.minutes)
}
