package master

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/fleet"
)

// The groups are fixed so the columns stay in the same place as checks are
// added. A check with no matching prefix lands in Other rather than being
// hidden, and batch 5 only has to extend this one map.
var checkGroupPrefixes = []struct {
	name     string
	prefixes []string
}{
	{"Connectivity", []string{"connect", "ping", "tailnet", "network"}},
	{"Health", []string{"cpu", "memory", "health"}},
	{"Agent", []string{"agent", "deploy"}},
	{"Jobs", []string{"job", "task"}},
	{"Storage", []string{"disk", "storage", "smart", "zfs", "lxc"}},
	{"Services", []string{"service", "systemd", "process", "ssh"}},
	{"Updates", []string{"update", "package", "apt", "reboot"}},
	{"Backups", []string{"backup", "restic"}},
}

func checkGroup(checkID string) string {
	id := strings.ToLower(checkID)
	for _, group := range checkGroupPrefixes {
		for _, prefix := range group.prefixes {
			if strings.HasPrefix(id, prefix) {
				return group.name
			}
		}
	}
	return "Other"
}

// statusRank puts red cells first. Skipped, unknown, not-run and stale results
// all rank zero: none of them is a failure.
func statusRank(status checks.Status, stale bool) int {
	if stale {
		return 0
	}
	switch status {
	case checks.StatusCritical:
		return 3
	case checks.StatusWarning:
		return 2
	case checks.StatusOK:
		return 1
	default:
		return 0
	}
}

// builtinBlocks turns the facts already on the dashboard into matrix cells.
// They are not saved checks: they are recomputed on every request, so they
// cannot be stale and need no check definitions.
func builtinBlocks(device fleet.Device, problems int) map[string][]checkBlock {
	now := time.Now().UTC()
	blocks := map[string][]checkBlock{}

	connection := checkBlock{CheckID: "builtin_connectivity", Status: checks.StatusOK, Summary: "Agent session live", RanAt: now}
	if device.Status != fleet.StatusOnline {
		connection.Status = checks.StatusCritical
		connection.Summary = "No live agent session"
	}
	blocks["Connectivity"] = append(blocks["Connectivity"], connection)

	health := checkBlock{CheckID: "builtin_health", Status: checks.StatusOK, Summary: "CPU and memory within limits", RanAt: now}
	if device.Stats == nil {
		health.Status, health.Summary = checks.StatusUnknown, "No recent stats from this device"
	} else if device.Stats.CPU.UsagePercent > highCPUPercent {
		health.Status = checks.StatusWarning
		health.Summary = fmt.Sprintf("CPU at %.0f%%", device.Stats.CPU.UsagePercent)
	} else if device.Stats.Memory.UsedPercent > highMemoryPercent {
		health.Status = checks.StatusWarning
		health.Summary = fmt.Sprintf("Memory at %.0f%%", device.Stats.Memory.UsedPercent)
	}
	blocks["Health"] = append(blocks["Health"], health)

	agent := checkBlock{CheckID: "builtin_agent", Status: checks.StatusOK, Summary: "Agent managed and running", RanAt: now}
	if device.Master {
		agent.Summary = "Master agent on the local socket"
	} else if device.Deploy == nil {
		agent.Status, agent.Summary = checks.StatusSkipped, "Not deployed by this master"
	} else {
		switch device.Deploy.Status {
		case fleet.DeployStopped:
			agent.Status, agent.Summary = checks.StatusWarning, "Agent stopped on purpose"
		case fleet.DeployFailed:
			agent.Status, agent.Summary = checks.StatusCritical, "Last install failed"
		case fleet.DeployPending:
			agent.Status, agent.Summary = checks.StatusUnknown, "Install still pending"
		}
	}
	blocks["Agent"] = append(blocks["Agent"], agent)

	jobs := checkBlock{CheckID: "builtin_jobs", Status: checks.StatusOK, Summary: "No unfinished tasks", RanAt: now}
	if problems > 0 {
		jobs.Status = checks.StatusWarning
		jobs.Summary = fmt.Sprintf("%d tasks did not finish", problems)
	}
	blocks["Jobs"] = append(blocks["Jobs"], jobs)
	return blocks
}

// matrixVerdict counts blocks by state. A count is not a score: unknown and
// stale blocks are reported, never folded into a failure.
func matrixVerdict(matrix checkMatrix, issues []dashboardIssue) map[string]int {
	counts := map[string]int{"ok": 0, "warning": 0, "critical": 0, "unknown": 0}
	known := make(map[string]bool, len(matrix.Rows))
	for _, row := range matrix.Rows {
		known[row.DeviceID] = true
		for _, cell := range row.Cells {
			for _, block := range cell.Checks {
				// A stale result is grey in the UI and never a live failure.
				if block.Stale || block.Status == checks.StatusSkipped || block.Status == checks.StatusUnknown {
					counts["unknown"]++
					continue
				}
				switch block.Status {
				case checks.StatusOK:
					counts["ok"]++
				case checks.StatusWarning:
					counts["warning"]++
				case checks.StatusCritical:
					counts["critical"]++
				default:
					counts["unknown"]++
				}
			}
		}
	}
	// Issues without a device row (or without any device at all) have no block
	// in the matrix, so they still need to reach the verdict.
	for _, issue := range issues {
		if issue.DeviceID != "" && known[issue.DeviceID] {
			continue
		}
		switch issue.Severity {
		case "error":
			counts["critical"]++
		case "warning":
			counts["warning"]++
		}
	}
	return counts
}

type checkMatrix struct {
	Groups []string         `json:"groups"`
	Rows   []checkMatrixRow `json:"rows"`
}

type checkMatrixRow struct {
	DeviceID     string           `json:"device_id"`
	DeviceName   string           `json:"device_name"`
	DeviceStatus string           `json:"device_status"`
	Cells        []checkGroupCell `json:"cells"`
	severity     int              // unexported: only used to sort rows
}

type checkGroupCell struct {
	Group  string       `json:"group"`
	Checks []checkBlock `json:"checks"`
}

type checkBlock struct {
	CheckID string          `json:"check_id"`
	Status  checks.Status   `json:"status"`
	Summary string          `json:"summary"`
	Stale   bool            `json:"stale"`
	RanAt   time.Time       `json:"ran_at"`
	Details []checks.Detail `json:"details,omitempty"`
}

// buildCheckMatrix joins devices with the latest result of every known check.
// A device with no stored result gets an "unknown" block, which the UI shows
// as grey: never having run is not a failure.
func buildCheckMatrix(devices []fleet.Device, results []checks.Result, problemJobs map[string]int) checkMatrix {
	byPair := make(map[[2]string]checks.Result, len(results))
	known := make(map[string][]string)
	seen := make(map[[2]string]bool)
	for _, result := range results {
		key := [2]string{result.DeviceID, result.CheckID}
		byPair[key] = result
		group := checkGroup(result.CheckID)
		if seen[[2]string{group, result.CheckID}] {
			continue
		}
		seen[[2]string{group, result.CheckID}] = true
		known[group] = append(known[group], result.CheckID)
	}
	for group := range known {
		sort.Strings(known[group])
	}

	groups := make([]string, 0, len(checkGroupPrefixes)+1)
	for _, group := range checkGroupPrefixes {
		groups = append(groups, group.name)
	}
	if len(known["Other"]) > 0 {
		groups = append(groups, "Other")
	}

	matrix := checkMatrix{Groups: groups, Rows: make([]checkMatrixRow, 0, len(devices))}
	for _, device := range devices {
		row := checkMatrixRow{
			DeviceID:     device.ID,
			DeviceName:   device.Name,
			DeviceStatus: device.Status,
			Cells:        make([]checkGroupCell, 0, len(groups)),
		}
		// Built-in signals keep every group informative even with no saved
		// checks. Stored results are listed first so a real check takes
		// priority; the built-in block still reflects facts such as an offline
		// session or a failed job that the saved checks say nothing about.
		builtins := builtinBlocks(device, problemJobs[device.ID])
		for _, group := range groups {
			cell := checkGroupCell{Group: group, Checks: make([]checkBlock, 0, len(known[group]))}
			for _, id := range known[group] {
				block := checkBlock{CheckID: id, Status: checks.StatusUnknown, Summary: "Not run"}
				if result, ok := byPair[[2]string{device.ID, id}]; ok {
					block.Status = result.Status
					block.Summary = result.Summary
					block.Stale = result.Stale
					block.RanAt = result.RanAt
					block.Details = result.Details
				}
				if rank := statusRank(block.Status, block.Stale); rank > row.severity {
					row.severity = rank
				}
				cell.Checks = append(cell.Checks, block)
			}
			for _, block := range builtins[group] {
				if rank := statusRank(block.Status, block.Stale); rank > row.severity {
					row.severity = rank
				}
				cell.Checks = append(cell.Checks, block)
			}
			row.Cells = append(row.Cells, cell)
		}
		matrix.Rows = append(matrix.Rows, row)
	}

	sort.SliceStable(matrix.Rows, func(i, j int) bool {
		if matrix.Rows[i].severity != matrix.Rows[j].severity {
			return matrix.Rows[i].severity > matrix.Rows[j].severity
		}
		return matrix.Rows[i].DeviceName < matrix.Rows[j].DeviceName
	})
	return matrix
}
