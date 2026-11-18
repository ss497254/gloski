package agents

import (
	"path/filepath"
	"testing"

	"github.com/ss497254/gloski/internal/database"
)

func TestJobStoreLifecycle(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "master.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store, err := NewJobStore(db)
	if err != nil {
		t.Fatal(err)
	}

	job, err := store.Create("device-1", "echo hello", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkRunning(job.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendLog(job.ID, []byte("hello\nworld\n"), false); err != nil {
		t.Fatal(err)
	}
	if err := store.Finish(job.ID, 0, ""); err != nil {
		t.Fatal(err)
	}

	stored, err := store.Get(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != JobFinished || stored.StartedAt.IsZero() || stored.FinishedAt.IsZero() {
		t.Fatalf("unexpected job: %+v", stored)
	}

	logs, err := store.Logs(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 || logs[0] != "hello" || logs[1] != "world" {
		t.Fatalf("logs = %q", logs)
	}

	jobs, err := store.List("device-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d", len(jobs))
	}
}
