package jobs

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/logger"
)

// Config holds configuration for the jobs service
type Config struct {
	LogsDir string // Directory for job log files
	MaxJobs int    // Maximum number of jobs to keep (default: 100)
}

// Service manages job execution
type Service struct {
	config Config
	store  *Store
	jobs   map[string]*runningJob
	mu     sync.RWMutex
}

// runningJob holds the runtime state of a running job
type runningJob struct {
	*Job
	cmd     *exec.Cmd
	logFile *os.File
}

// NewService creates a new jobs service
func NewService(db *database.Database, config Config) (*Service, error) {
	// Ensure sensible defaults
	if config.MaxJobs <= 0 {
		config.MaxJobs = 100
	}

	// Ensure logs directory exists
	if config.LogsDir != "" {
		if err := os.MkdirAll(config.LogsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create logs directory: %w", err)
		}
	}

	store := NewStore(db)

	s := &Service{
		config: config,
		store:  store,
		jobs:   make(map[string]*runningJob),
	}

	// Load existing jobs and mark running ones as stopped (server restarted)
	jobs, err := store.Load()
	if err != nil {
		logger.Warn("Failed to load jobs: %v", err)
	} else {
		for _, j := range jobs {
			if j.Status == StatusRunning || j.Status == StatusPending {
				j.Status = StatusStopped
				now := time.Now()
				j.FinishedAt = &now
				store.Update(j)
			}
		}
	}

	// Run cleanup on startup
	s.cleanup()

	return s, nil
}

// Start starts a new job
// Commands are executed through the shell to support pipes, redirects, and shell features
func (s *Service) Start(id, command, cwd string) (*Job, error) {
	if command == "" {
		return nil, fmt.Errorf("empty command")
	}

	// Create job record
	now := time.Now()
	job := &Job{
		ID:        id,
		Command:   command,
		Cwd:       cwd,
		Status:    StatusRunning,
		CreatedAt: now,
		StartedAt: &now,
	}

	// Set up log file
	if s.config.LogsDir != "" {
		job.LogFile = filepath.Join(s.config.LogsDir, fmt.Sprintf("%s.log", id))
	}

	// Create the command using shell for proper handling of pipes, redirects, etc.
	// This is intentional - jobs are meant to run arbitrary shell commands
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell, "-c", command)
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	// Set up output capture
	var logFile *os.File
	if job.LogFile != "" {
		var err error
		logFile, err = os.Create(job.LogFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create log file: %w", err)
		}
	}

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if logFile != nil {
			logFile.Close()
		}
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		if logFile != nil {
			logFile.Close()
		}
		return nil, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		if logFile != nil {
			logFile.Close()
		}
		return nil, err
	}

	job.PID = cmd.Process.Pid

	// Save to database
	if err := s.store.Insert(job); err != nil {
		cmd.Process.Kill()
		if logFile != nil {
			logFile.Close()
		}
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	// Track running job
	rj := &runningJob{
		Job:     job,
		cmd:     cmd,
		logFile: logFile,
	}

	s.mu.Lock()
	s.jobs[id] = rj
	s.mu.Unlock()

	// Capture stdout in background
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if logFile != nil {
				fmt.Fprintln(logFile, line)
			}
		}
	}()

	// Capture stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := "[stderr] " + scanner.Text()
			if logFile != nil {
				fmt.Fprintln(logFile, line)
			}
		}
	}()

	// Wait for process in background
	go func() {
		err := cmd.Wait()

		s.mu.Lock()
		if err != nil {
			job.Status = StatusFailed
			if exitErr, ok := err.(*exec.ExitError); ok {
				job.ExitCode = exitErr.ExitCode()
			}
		} else {
			job.Status = StatusFinished
			job.ExitCode = 0
		}
		now := time.Now()
		job.FinishedAt = &now

		// Close log file
		if logFile != nil {
			logFile.Close()
		}

		// Remove from running jobs
		delete(s.jobs, id)
		s.mu.Unlock()

		// Update database
		s.store.Update(job)

		// Run cleanup after job finishes
		s.cleanup()
	}()

	return job, nil
}

// Stop stops a running job
func (s *Service) Stop(id string) error {
	s.mu.Lock()
	rj, ok := s.jobs[id]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("job not found or not running")
	}

	if rj.Status != StatusRunning {
		return fmt.Errorf("job is not running")
	}

	if err := rj.cmd.Process.Kill(); err != nil {
		return err
	}

	return nil
}

// List returns all jobs from the database
func (s *Service) List() ([]*Job, error) {
	return s.store.Load()
}

// Get returns a job by ID
func (s *Service) Get(id string) (*Job, error) {
	// First check running jobs for live status
	s.mu.RLock()
	if rj, ok := s.jobs[id]; ok {
		s.mu.RUnlock()
		return rj.Job, nil
	}
	s.mu.RUnlock()

	// Otherwise get from database
	return s.store.Get(id)
}

// GetLogs reads logs from the job's log file
func (s *Service) GetLogs(id string) ([]string, error) {
	job, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	if job.LogFile == "" {
		return []string{}, nil
	}

	file, err := os.Open(job.LogFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// Delete removes a job and its log file
func (s *Service) Delete(id string) error {
	// Stop if running
	s.mu.Lock()
	if rj, ok := s.jobs[id]; ok {
		if rj.cmd != nil && rj.cmd.Process != nil {
			rj.cmd.Process.Kill()
		}
		delete(s.jobs, id)
	}
	s.mu.Unlock()

	// Get job to find log file
	job, err := s.store.Get(id)
	if err != nil {
		return err
	}

	// Delete log file
	if job.LogFile != "" {
		os.Remove(job.LogFile)
	}

	// Delete from database
	return s.store.Delete(id)
}

// Shutdown stops all running jobs
func (s *Service) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rj := range s.jobs {
		if rj.cmd != nil && rj.cmd.Process != nil {
			rj.cmd.Process.Kill()
		}
		if rj.logFile != nil {
			rj.logFile.Close()
		}
	}

	return nil
}

// cleanup removes old jobs beyond the configured limit
func (s *Service) cleanup() {
	oldJobs, err := s.store.GetOldestJobs(s.config.MaxJobs)
	if err != nil {
		logger.Warn("Failed to get old jobs for cleanup: %v", err)
		return
	}

	for _, job := range oldJobs {
		// Delete log file
		if job.LogFile != "" {
			os.Remove(job.LogFile)
		}
		// Delete from database
		s.store.Delete(job.ID)
	}

	if len(oldJobs) > 0 {
		logger.Debug("Cleaned up %d old jobs", len(oldJobs))
	}
}
