package controltower

import (
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/fleet"
)

func TestCheckGroup(t *testing.T) {
	cases := map[string]string{
		"ping_tcp":      "Connectivity",
		"storage_disk":  "Storage",
		"smart_health":  "Storage",
		"service_sshd":  "Services",
		"updates_apt":   "Updates",
		"backup_restic": "Backups",
		"mystery":       "Other",
	}
	for id, want := range cases {
		if got := checkGroup(id); got != want {
			t.Errorf("checkGroup(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestBuildCheckMatrixRedFirst(t *testing.T) {
	devices := []fleet.Device{
		{ID: "quiet", Name: "quiet", Status: fleet.StatusOnline},
		{ID: "warning", Name: "warning", Status: fleet.StatusOnline},
		{ID: "critical", Name: "critical", Status: fleet.StatusOffline},
	}
	now := time.Now()
	results := []checks.Result{
		// Use real check IDs so grouping is exercised too.
		{CheckID: "disk_root", DeviceID: "critical", Status: checks.StatusCritical, Summary: "full", RanAt: now},
		{CheckID: "disk_root", DeviceID: "warning", Status: checks.StatusWarning, Summary: "high", RanAt: now},
		{CheckID: "ping_tcp", DeviceID: "warning", Status: checks.StatusOK, Summary: "ok", RanAt: now},
		{CheckID: "ping_tcp", DeviceID: "quiet", Status: checks.StatusCritical, Summary: "stale", RanAt: now.Add(-time.Hour), Stale: true},
		{CheckID: "mystery", DeviceID: "critical", Status: checks.StatusOK, Summary: "ok", RanAt: now},
	}
	matrix := buildCheckMatrix(devices, results)
	if len(matrix.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(matrix.Rows))
	}
	wantOrder := []string{"critical", "warning", "quiet"}
	for i, want := range wantOrder {
		if matrix.Rows[i].DeviceID != want {
			t.Fatalf("row %d = %s, want %s", i, matrix.Rows[i].DeviceID, want)
		}
	}

	groups := map[string]bool{}
	for _, group := range matrix.Groups {
		groups[group] = true
	}
	for _, group := range []string{"Connectivity", "Storage", "Services", "Updates", "Backups", "Other"} {
		if !groups[group] {
			t.Errorf("missing group %q in %v", group, matrix.Groups)
		}
	}

	// The quiet device has a stale critical result: it must not sort first and
	// its block must carry the stale marker for the grey rendering.
	quiet := matrix.Rows[2]
	for _, cell := range quiet.Cells {
		for _, block := range cell.Checks {
			if block.CheckID == "ping_tcp" {
				if !block.Stale || block.Status != checks.StatusCritical {
					t.Fatalf("stale block = %+v", block)
				}
			}
			if block.CheckID == "disk_root" && (block.Status != checks.StatusUnknown || block.Summary != "Not run") {
				t.Fatalf("not-run block = %+v", block)
			}
		}
	}
}
