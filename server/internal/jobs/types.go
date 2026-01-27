package jobs

import (
	"time"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	StatusPending  JobStatus = "pending"
	StatusRunning  JobStatus = "running"
	StatusStopped  JobStatus = "stopped"
	StatusFailed   JobStatus = "failed"
	StatusFinished JobStatus = "finished"
)

// Job represents an executed command/job
type Job struct {
	ID         string     `json:"id"`
	Command    string     `json:"command"`
	Cwd        string     `json:"cwd"`
	Status     JobStatus  `json:"status"`
	PID        int        `json:"pid,omitempty"`
	ExitCode   int        `json:"exit_code,omitempty"`
	LogFile    string     `json:"log_file,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// IsActive returns true if the job is currently running
func (j *Job) IsActive() bool {
	return j.Status == StatusRunning || j.Status == StatusPending
}

// StartJobRequest represents a request to start a new job
type StartJobRequest struct {
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
}
