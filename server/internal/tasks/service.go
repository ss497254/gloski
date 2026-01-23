package tasks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type TaskStatus string

const (
	TaskRunning  TaskStatus = "running"
	TaskStopped  TaskStatus = "stopped"
	TaskFailed   TaskStatus = "failed"
	TaskFinished TaskStatus = "finished"
)

type Task struct {
	ID        string     `json:"id"`
	Command   string     `json:"command"`
	Cwd       string     `json:"cwd"`
	Status    TaskStatus `json:"status"`
	PID       int        `json:"pid"`
	StartedAt time.Time  `json:"started_at"`
	ExitCode  int        `json:"exit_code,omitempty"`

	cmd    *exec.Cmd
	output []string
	mu     sync.RWMutex
}

type Service struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

func NewService() *Service {
	return &Service{
		tasks: make(map[string]*Task),
	}
}

func (s *Service) Start(id, command, cwd string) (*Task, error) {
	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	task := &Task{
		ID:        id,
		Command:   command,
		Cwd:       cwd,
		Status:    TaskRunning,
		PID:       cmd.Process.Pid,
		StartedAt: time.Now(),
		cmd:       cmd,
		output:    make([]string, 0),
	}

	// Capture output in background
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			task.mu.Lock()
			task.output = append(task.output, scanner.Text())
			// Keep last 1000 lines
			if len(task.output) > 1000 {
				task.output = task.output[len(task.output)-1000:]
			}
			task.mu.Unlock()
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			task.mu.Lock()
			task.output = append(task.output, "[stderr] "+scanner.Text())
			if len(task.output) > 1000 {
				task.output = task.output[len(task.output)-1000:]
			}
			task.mu.Unlock()
		}
	}()

	// Wait for process in background
	go func() {
		err := cmd.Wait()
		task.mu.Lock()
		if err != nil {
			task.Status = TaskFailed
			if exitErr, ok := err.(*exec.ExitError); ok {
				task.ExitCode = exitErr.ExitCode()
			}
		} else {
			task.Status = TaskFinished
			task.ExitCode = 0
		}
		task.mu.Unlock()
	}()

	s.mu.Lock()
	s.tasks[id] = task
	s.mu.Unlock()

	return task, nil
}

func (s *Service) Stop(id string) error {
	s.mu.RLock()
	task, ok := s.tasks[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found")
	}

	task.mu.Lock()
	defer task.mu.Unlock()

	if task.Status != TaskRunning {
		return fmt.Errorf("task is not running")
	}

	if err := task.cmd.Process.Kill(); err != nil {
		return err
	}

	task.Status = TaskStopped
	return nil
}

func (s *Service) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

func (s *Service) Get(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	return task, ok
}

func (s *Service) GetOutput(id string) ([]string, error) {
	s.mu.RLock()
	task, ok := s.tasks[id]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("task not found")
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	output := make([]string, len(task.output))
	copy(output, task.output)
	return output, nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task not found")
	}

	if task.Status == TaskRunning {
		task.cmd.Process.Kill()
	}

	delete(s.tasks, id)
	return nil
}

// Shutdown stops all running tasks gracefully.
func (s *Service) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range s.tasks {
		if task.Status == TaskRunning && task.cmd != nil && task.cmd.Process != nil {
			task.cmd.Process.Kill()
		}
	}

	return nil
}

// Systemd service management
type SystemdUnit struct {
	Name        string `json:"name"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
}

func (s *Service) ListSystemdUnits(userMode bool) ([]SystemdUnit, error) {
	args := []string{"list-units", "--type=service", "--all", "--output=json"}
	if userMode {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.Command("systemctl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var units []SystemdUnit
	if err := json.Unmarshal(output, &units); err != nil {
		return nil, err
	}

	return units, nil
}

func (s *Service) SystemdAction(unit, action string, userMode bool) error {
	args := []string{action, unit}
	if userMode {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.Command("systemctl", args...)
	return cmd.Run()
}

func (s *Service) SystemdLogs(unit string, userMode bool, lines int) (string, error) {
	args := []string{"-u", unit, "-n", fmt.Sprintf("%d", lines), "--no-pager"}
	if userMode {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.Command("journalctl", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
