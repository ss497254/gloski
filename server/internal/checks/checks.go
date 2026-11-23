// Package checks is the result model for saved checks. A result keeps facts
// (what the command said) separate from the verdict (what it means), and knows
// the difference between "not applicable", "never ran" and "old result".
package checks

import (
	"strings"
	"time"
)

// Status is a check's verdict. Skipped and unknown are not failures.
type Status string

const (
	StatusOK       Status = "ok"
	StatusWarning  Status = "warning"
	StatusCritical Status = "critical"
	StatusSkipped  Status = "skipped"
	StatusUnknown  Status = "unknown"
)

// Detail is one ordered key/value line from a check's output.
type Detail struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Result is one run of one check on one device.
type Result struct {
	CheckID    string    `json:"check_id"`
	DeviceID   string    `json:"device_id"`
	Status     Status    `json:"status"`
	Summary    string    `json:"summary"`
	Details    []Detail  `json:"details,omitempty"`
	RanAt      time.Time `json:"ran_at"`
	DurationMS int64     `json:"duration_ms"`
	Stale      bool      `json:"stale"`

	// Interval is the check's schedule. Readers use it to recompute Stale
	// against the current time; a stored boolean would go stale itself.
	Interval time.Duration `json:"-"`
}

// StaleAt reports whether the result is older than twice its schedule. A
// result with no known schedule is never called stale: there is no basis for
// the claim.
func (r Result) StaleAt(now time.Time) bool {
	return r.Interval > 0 && now.Sub(r.RanAt) > 2*r.Interval
}

// RefreshStale recomputes the serialized verdict against now.
func (r *Result) RefreshStale(now time.Time) {
	r.Stale = r.StaleAt(now)
}

// ParseOutput turns a check command's output and exit code into a Result.
// The convention is deliberately small:
//
//	exit 0 = ok, 1 = warning, 2 = critical, anything else = unknown
//	first line "GLOSKI-SKIP: <reason>" = skipped
//	first line of output = summary, remaining lines = details
//
// A skipped line wins over the exit code; a command that prints nothing is
// unknown, because there is no fact to show.
func ParseOutput(checkID, deviceID, output string, exitCode int, ranAt time.Time, elapsed, interval time.Duration) Result {
	result := Result{
		CheckID:    checkID,
		DeviceID:   deviceID,
		RanAt:      ranAt.UTC(),
		DurationMS: elapsed.Milliseconds(),
		Interval:   interval,
	}
	if result.DurationMS < 0 {
		result.DurationMS = 0
	}

	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	if len(lines) > 0 && strings.HasPrefix(lines[0], "GLOSKI-SKIP:") {
		reason := strings.TrimSpace(strings.TrimPrefix(lines[0], "GLOSKI-SKIP:"))
		if reason == "" {
			reason = "Not applicable to this device."
		}
		result.Status = StatusSkipped
		result.Summary = reason
		result.RefreshStale(time.Now())
		return result
	}

	first := -1
	for i, line := range lines {
		if line != "" {
			first = i
			break
		}
	}
	if first < 0 {
		result.Status = StatusUnknown
		result.Summary = "The check produced no output."
		result.RefreshStale(time.Now())
		return result
	}

	result.Status = statusForExit(exitCode)
	result.Summary = lines[first]
	result.Details = parseDetails(lines[first+1:])
	result.RefreshStale(time.Now())
	return result
}

func statusForExit(code int) Status {
	switch code {
	case 0:
		return StatusOK
	case 1:
		return StatusWarning
	case 2:
		return StatusCritical
	default:
		return StatusUnknown
	}
}

// parseDetails keeps the command's order. "key=value" and "key: value" become
// key/value pairs; other lines keep their text as the value.
func parseDetails(lines []string) []Detail {
	var details []Detail
	for _, line := range lines {
		if line == "" {
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok {
			details = append(details, Detail{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
			continue
		}
		if key, value, ok := strings.Cut(line, ": "); ok {
			details = append(details, Detail{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
			continue
		}
		details = append(details, Detail{Value: line})
	}
	return details
}
