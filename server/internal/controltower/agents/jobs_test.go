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
	if err := store.Finish(job.ID, 0, "", false); err != nil {
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

func TestJobStoreTerminalStates(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "master.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err := NewJobStore(db)
	if err != nil {
		t.Fatal(err)
	}
	status := func(id string) string {
		job, err := store.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		return job.Status
	}
	start := func(device string) string {
		job, err := store.Create(device, "sleep 60", "")
		if err != nil {
			t.Fatal(err)
		}
		if err := store.MarkRunning(job.ID); err != nil {
			t.Fatal(err)
		}
		return job.ID
	}

	// A cancelled job is stopped, not failed, even though the shell was killed.
	stopped := start("device-1")
	if err := store.Finish(stopped, -1, "signal: terminated", true); err != nil {
		t.Fatal(err)
	}
	if got := status(stopped); got != JobStopped {
		t.Fatalf("cancelled job status = %s, want %s", got, JobStopped)
	}

	// Jobs still running when the master starts are abandoned, and a result
	// arriving afterwards does not rewrite that.
	orphan := start("device-1")
	if err := store.AbortAll("master restarted"); err != nil {
		t.Fatal(err)
	}
	if err := store.Finish(orphan, 0, "", false); err != nil {
		t.Fatal(err)
	}
	if got := status(orphan); got != JobAbandoned {
		t.Fatalf("orphaned job status = %s, want %s", got, JobAbandoned)
	}
	if got := status(stopped); got != JobStopped {
		t.Fatalf("AbortAll changed a finished job to %s", got)
	}

	// Forgetting a device removes its jobs and logs but not another device's.
	other := start("device-2")
	if err := store.AppendLog(orphan, []byte("x\n"), false); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteDevice("device-1"); err != nil {
		t.Fatal(err)
	}
	if jobs, _ := store.List("device-1"); len(jobs) != 0 {
		t.Fatalf("device-1 still has %d jobs", len(jobs))
	}
	if logs, _ := store.Logs(orphan); len(logs) != 0 {
		t.Fatalf("device-1 logs survived: %v", logs)
	}
	if got := status(other); got != JobRunning {
		t.Fatalf("device-2 job status = %s, want %s", got, JobRunning)
	}
}
