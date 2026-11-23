package controltower

import (
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
func buildCheckMatrix(devices []fleet.Device, results []checks.Result) checkMatrix {
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
