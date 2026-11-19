// Package supervisor keeps one agent alive on a device. It is the same binary
// as the agent: "gloski-agent start" launches a detached supervisor, which
// runs the agent as a child and restarts it whenever it dies.
package supervisor

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// ExitRestart is the agent's way of asking to be started again from the
	// binary on disk, used after it has replaced that binary with an update.
	ExitRestart = 75

	maxLogBytes  = 5 << 20
	maxBackoff   = 30 * time.Second
	stableRun    = time.Minute
	stopGrace    = 15 * time.Second
	lockFileName = "agent.lock"
	pidFileName  = "agent.pid"
	envFileName  = "agent.env"
	logFileName  = "agent.log"
)

// Start launches a detached supervisor for the install in dir. It succeeds
// without doing anything when one is already running.
func Start(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, envFileName)); err != nil {
		return fmt.Errorf("no agent is installed in %s: %w", dir, err)
	}
	lock, held, err := tryLock(dir)
	if err != nil {
		return err
	}
	if held {
		return nil
	}
	_ = lock.Close()

	self, err := os.Executable()
	if err != nil {
		return err
	}
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer null.Close()

	command := exec.Command(self, "supervise", "-dir", dir)
	command.Dir = dir
	command.Stdin, command.Stdout, command.Stderr = null, null, null
	// A new session, so the supervisor survives the shell or SSH session that
	// started it.
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		return err
	}
	pid := command.Process.Pid
	_ = command.Process.Release()

	// Wait until it owns the lock, so the caller can read the pid file.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if lock, held, err := tryLock(dir); err == nil {
			if held {
				return nil
			}
			_ = lock.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("supervisor (pid %d) did not start", pid)
}

// Run supervises the agent until it is told to stop. It returns the process
// exit code.
func Run(dir string) int {
	lock, held, err := tryLock(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if held {
		return 0 // another supervisor owns this install
	}
	defer lock.Close()

	log := newCappedLog(filepath.Join(dir, logFileName))
	defer log.Close()
	say := func(format string, args ...any) {
		fmt.Fprintf(log, "%s [SUPERVISOR] %s\n", time.Now().Format("2006/01/02 15:04:05"), fmt.Sprintf(format, args...))
	}

	pidFile := filepath.Join(dir, pidFileName)
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0644); err != nil {
		say("cannot write %s: %v", pidFile, err)
	}
	defer func() {
		// Only remove the pid file if it is still ours.
		if data, err := os.ReadFile(pidFile); err == nil && strings.TrimSpace(string(data)) == strconv.Itoa(os.Getpid()) {
			_ = os.Remove(pidFile)
		}
	}()

	self, err := os.Executable()
	if err != nil {
		say("cannot find own binary: %v", err)
		return 1
	}

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	// Pdeathsig is tied to the thread that started the child, so that thread
	// must live as long as the supervisor does.
	runtime.LockOSThread()

	backoff := time.Second
	restarts := 0
	lastExit := ""
	for {
		env, err := readEnvFile(filepath.Join(dir, envFileName))
		if err != nil {
			say("cannot read agent settings: %v", err)
			return 1
		}
		command := exec.Command(self)
		command.Dir = dir
		command.Env = append(os.Environ(), env...)
		command.Env = append(command.Env,
			"GLOSKI_AGENT_INSTALL_DIR="+dir,
			"GLOSKI_AGENT_RESTARTS="+strconv.Itoa(restarts),
			"GLOSKI_AGENT_LAST_EXIT="+lastExit,
		)
		command.Stdout, command.Stderr = log, log
		// If the supervisor is killed outright the agent must go too, or the
		// next supervisor would start a second agent beside it.
		command.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}

		started := time.Now()
		if err := command.Start(); err != nil {
			say("cannot start agent: %v", err)
			lastExit = "could not start: " + err.Error()
		} else {
			exited := make(chan error, 1)
			go func() { exited <- command.Wait() }()

			var waitErr error
			stopping := false
			select {
			case waitErr = <-exited:
			case sig := <-signals:
				stopping = true
				say("received %s, stopping agent", sig)
				_ = command.Process.Signal(syscall.SIGTERM)
				select {
				case waitErr = <-exited:
				case <-time.After(stopGrace):
					_ = command.Process.Kill()
					waitErr = <-exited
				}
			}
			if stopping {
				return 0
			}

			code := exitCode(waitErr)
			switch {
			case code == 0:
				say("agent stopped on request; supervisor exiting")
				return 0
			case code == ExitRestart:
				say("agent asked for a restart from the binary on disk")
				log.Close()
				_ = lock.Close()
				err := syscall.Exec(self, []string{self, "supervise", "-dir", dir}, os.Environ())
				// Exec only returns on failure; carry on with the old binary.
				lock, _, _ = tryLock(dir)
				say("could not re-exec: %v", err)
				lastExit = "update restart failed"
			default:
				lastExit = describeExit(waitErr)
				say("agent exited: %s", lastExit)
			}
		}

		if time.Since(started) >= stableRun {
			backoff = time.Second
		}
		restarts++
		say("restarting agent in %s", backoff)
		select {
		case <-signals:
			return 0
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
		}
	}
}

// tryLock takes the install's lock. held is true when another process has it.
func tryLock(dir string) (file *os.File, held bool, err error) {
	file, err = os.OpenFile(filepath.Join(dir, lockFileName), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, true, nil
		}
		return nil, false, err
	}
	return file, false, nil
}

// readEnvFile reads KEY=value lines. Values run to the end of the line and
// are taken literally; there is no quoting.
func readEnvFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var env []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		env = append(env, line)
	}
	return env, scanner.Err()
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func describeExit(err error) string {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			if status.Signal() == syscall.SIGKILL {
				return "killed (out of memory, or killed by hand)"
			}
			return "killed by signal " + status.Signal().String()
		}
		return fmt.Sprintf("exit code %d", exitErr.ExitCode())
	}
	if err == nil {
		return "exit code 0"
	}
	return err.Error()
}

// cappedLog appends to a file and starts a fresh one once it grows past
// maxLogBytes, keeping a single previous file.
type cappedLog struct {
	mu   sync.Mutex
	path string
	file *os.File
	size int64
}

func newCappedLog(path string) *cappedLog {
	return &cappedLog{path: path}
}

func (l *cappedLog) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil && l.size+int64(len(data)) > maxLogBytes {
		_ = l.file.Close()
		l.file = nil
		_ = os.Rename(l.path, l.path+".1")
	}
	if l.file == nil {
		file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return len(data), nil // never block the agent on its own log
		}
		l.file = file
		l.size = 0
		if info, err := file.Stat(); err == nil {
			l.size = info.Size()
		}
	}
	n, _ := l.file.Write(data)
	l.size += int64(n)
	return len(data), nil
}

func (l *cappedLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
	return nil
}
