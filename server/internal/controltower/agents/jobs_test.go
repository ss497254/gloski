package agents

import (
	"path/filepath"
	"strings"
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
	if err := store.AppendLog(job.ID, 1, []byte("hello\nworld\n"), false); err != nil {
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

	// After a reconnect, a job the agent no longer holds is abandoned while
	// one it still holds keeps running. A late result does not rewrite the
	// abandoned one.
	orphan := start("device-1")
	kept := start("device-1")
	if err := store.AbandonMissing("device-1", map[string]bool{kept: true}, "agent restarted"); err != nil {
		t.Fatal(err)
	}
	if got := status(kept); got != JobRunning {
		t.Fatalf("held job status = %s, want %s", got, JobRunning)
	}
	if err := store.Finish(orphan, 0, "", false); err != nil {
		t.Fatal(err)
	}
	if got := status(orphan); got != JobAbandoned {
		t.Fatalf("orphaned job status = %s, want %s", got, JobAbandoned)
	}
	if got := status(stopped); got != JobStopped {
		t.Fatalf("reconciliation changed a finished job to %s", got)
	}

	// Forgetting a device removes its jobs and logs but not another device's.
	other := start("device-2")
	if err := store.AppendLog(orphan, 1, []byte("x\n"), false); err != nil {
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

func TestJobStoreOutputSequencing(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "master.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err := NewJobStore(db)
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.Create("device-1", "long task", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkRunning(job.ID); err != nil {
		t.Fatal(err)
	}
	add := func(seq uint64, text string) {
		t.Helper()
		if err := store.AppendLog(job.ID, seq, []byte(text), false); err != nil {
			t.Fatal(err)
		}
	}

	add(1, "one\n")
	add(2, "two\n")
	// A reconnect replays from the last acknowledged chunk; repeats are dropped.
	add(1, "one\n")
	add(2, "two\n")
	add(3, "three\n")
	if acked, active := store.Resume("device-1", job.ID); acked != 3 || !active {
		t.Fatalf("Resume = (%d, %v), want (3, true)", acked, active)
	}
	logs, err := store.Logs(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(logs, "|"); got != "one|two|three" {
		t.Fatalf("logs after replay = %q", got)
	}

	// The agent's replay buffer overflowed during a long outage: chunks 4-6
	// are gone, and the log says so instead of silently joining the pieces.
	add(7, "seven\n")
	logs, _ = store.Logs(job.ID)
	if got := strings.Join(logs, "|"); !strings.Contains(got, "output was lost") || !strings.HasSuffix(got, "seven") {
		t.Fatalf("logs after gap = %q", got)
	}

	// Another device cannot resume this job, and a finished job is not active.
	if _, active := store.Resume("device-2", job.ID); active {
		t.Fatal("job resumed for the wrong device")
	}
	if err := store.Finish(job.ID, 0, "", false); err != nil {
		t.Fatal(err)
	}
	if _, active := store.Resume("device-1", job.ID); active {
		t.Fatal("finished job reported as active")
	}
	// Output for a deleted job is discarded rather than stored as an orphan.
	if err := store.Delete(job.ID); err != nil {
		t.Fatal(err)
	}
	add(8, "late\n")
	if logs, _ := store.Logs(job.ID); len(logs) != 0 {
		t.Fatalf("output stored for a deleted job: %v", logs)
	}
}
