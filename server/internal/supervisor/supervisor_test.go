package supervisor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The supervisor runs its own binary as the agent. Under test that binary is
// this test program, so TestMain plays the agent when asked to: it fails on
// its first run and stops cleanly on its second.
func TestMain(m *testing.M) {
	if dir := os.Getenv("SUPERVISOR_TEST_CHILD"); dir != "" {
		runs := filepath.Join(dir, "runs")
		previous, _ := os.ReadFile(runs)
		_ = os.WriteFile(runs, append(previous, []byte(os.Getenv("GLOSKI_AGENT_RESTARTS")+"\n")...), 0600)
		if len(previous) == 0 {
			os.Exit(3)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestRunRestartsUntilCleanExit(t *testing.T) {
	dir := t.TempDir()
	env := "SUPERVISOR_TEST_CHILD=" + dir + "\n"
	if err := os.WriteFile(filepath.Join(dir, envFileName), []byte(env), 0600); err != nil {
		t.Fatal(err)
	}

	if code := Run(dir); code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}

	runs, _ := os.ReadFile(filepath.Join(dir, "runs"))
	if string(runs) != "0\n1\n" {
		t.Fatalf("agent runs = %q, want two runs with restart counts 0 and 1", runs)
	}
	log, _ := os.ReadFile(filepath.Join(dir, logFileName))
	if !strings.Contains(string(log), "agent exited: exit code 3") {
		t.Fatalf("log does not record the failed run:\n%s", log)
	}
	if _, err := os.Stat(filepath.Join(dir, pidFileName)); !os.IsNotExist(err) {
		t.Fatal("pid file was left behind")
	}
}

func TestSecondSupervisorBacksOff(t *testing.T) {
	dir := t.TempDir()
	lock, held, err := tryLock(dir)
	if err != nil || held {
		t.Fatalf("tryLock() held=%v err=%v", held, err)
	}
	defer lock.Close()

	if _, held, _ := tryLock(dir); !held {
		t.Fatal("second tryLock() should report the lock as held")
	}
	// No settings file exists, so anything but an immediate, quiet return
	// would be an error exit.
	if code := Run(dir); code != 0 {
		t.Fatalf("Run() with the lock held = %d, want 0", code)
	}
}

func TestStartNeedsAnInstall(t *testing.T) {
	if err := Start(t.TempDir()); err == nil {
		t.Fatal("Start() in an empty directory should fail so cron falls back to reinstalling")
	}
}

func TestCappedLogRotates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.log")
	log := newCappedLog(path)
	defer log.Close()

	chunk := make([]byte, 1<<20)
	for i := 0; i < 7; i++ {
		if _, err := log.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	current, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := os.Stat(path + ".1")
	if err != nil {
		t.Fatal("no rotated log file")
	}
	if current.Size() > maxLogBytes || previous.Size() > maxLogBytes {
		t.Fatalf("log sizes %d and %d exceed the cap", current.Size(), previous.Size())
	}
}
