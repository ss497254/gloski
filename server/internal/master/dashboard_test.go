package master

import (
	"testing"
	"time"
)

func TestSortIssues(t *testing.T) {
	now := time.Now()
	issues := []dashboardIssue{
		{ID: "warning-new", Severity: "warning", since: now},
		{ID: "error-new", Severity: "error", since: now},
		{ID: "warning-old", Severity: "warning", since: now.Add(-time.Hour)},
		{ID: "error-old", Severity: "error", since: now.Add(-time.Hour)},
		{ID: "info", Severity: "info", since: now.Add(-2 * time.Hour)},
	}
	sortIssues(issues)

	want := []string{"error-old", "error-new", "warning-old", "warning-new", "info"}
	for i, id := range want {
		if issues[i].ID != id {
			t.Fatalf("issues[%d].ID = %q, want %q (all: %+v)", i, issues[i].ID, id, issues)
		}
	}
}
