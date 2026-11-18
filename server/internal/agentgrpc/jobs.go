package agentgrpc

import (
	"bytes"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/database"
)

// Job status values stored by the master.
const (
	JobPending   = "pending"
	JobRunning   = "running"
	JobFinished  = "finished"
	JobFailed    = "failed"
	JobAbandoned = "abandoned"
)

// Job is a master-owned task executed by an agent.
type Job struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"device_id,omitempty"`
	Command    string    `json:"command"`
	CWD        string    `json:"cwd,omitempty"`
	Status     string    `json:"status"`
	ExitCode   int       `json:"exit_code,omitempty"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// JobStore persists jobs and their streamed logs in the master database.
type JobStore struct {
	db *sql.DB
}

// NewJobStore creates the master-side job tables if needed.
func NewJobStore(db *database.Database) (*JobStore, error) {
	store := &JobStore{db: db.DB()}
	_, err := store.db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_jobs (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			command TEXT NOT NULL,
			cwd TEXT,
			status TEXT NOT NULL,
			exit_code INTEGER DEFAULT 0,
			error TEXT,
			created_at DATETIME NOT NULL,
			started_at DATETIME,
			finished_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_agent_jobs_device ON agent_jobs(device_id);
		CREATE TABLE IF NOT EXISTS agent_job_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id TEXT NOT NULL,
			chunk BLOB NOT NULL,
			stderr INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_agent_job_logs_job ON agent_job_logs(job_id);
	`)
	if err != nil {
		return nil, err
	}
	return store, nil
}

// Create stores a new pending job.
func (s *JobStore) Create(deviceID, command, cwd string) (*Job, error) {
	job := &Job{
		ID:        uuid.NewString(),
		DeviceID:  deviceID,
		Command:   command,
		CWD:       cwd,
		Status:    JobPending,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.db.Exec(`
		INSERT INTO agent_jobs (id, device_id, command, cwd, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, job.ID, deviceID, command, cwd, JobPending, job.CreatedAt)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// MarkRunning transitions a job to running.
func (s *JobStore) MarkRunning(id string) error {
	_, err := s.db.Exec(`UPDATE agent_jobs SET status = ?, started_at = ? WHERE id = ?`, JobRunning, time.Now().UTC(), id)
	return err
}

// AppendLog stores one streamed output chunk.
func (s *JobStore) AppendLog(id string, chunk []byte, stderr bool) error {
	if len(chunk) == 0 {
		return nil
	}
	_, err := s.db.Exec(`
		INSERT INTO agent_job_logs (job_id, chunk, stderr, created_at) VALUES (?, ?, ?, ?)`,
		id, chunk, stderr, time.Now().UTC())
	return err
}

// Logs returns the complete log as lines.
func (s *JobStore) Logs(id string) ([]string, error) {
	rows, err := s.db.Query(`SELECT chunk FROM agent_job_logs WHERE job_id = ? ORDER BY id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buffer bytes.Buffer
	lines := []string{}
	for rows.Next() {
		var chunk []byte
		if err := rows.Scan(&chunk); err != nil {
			return nil, err
		}
		for _, b := range chunk {
			if b == '\n' {
				lines = append(lines, strings.TrimSuffix(buffer.String(), "\r"))
				buffer.Reset()
				continue
			}
			buffer.WriteByte(b)
		}
	}
	if buffer.Len() > 0 {
		lines = append(lines, buffer.String())
	}
	return lines, rows.Err()
}

// Finish records a job's terminal result.
func (s *JobStore) Finish(id string, exitCode int, errMsg string) error {
	status := JobFinished
	if errMsg != "" || exitCode != 0 {
		status = JobFailed
	}
	_, err := s.db.Exec(`
		UPDATE agent_jobs SET status = ?, exit_code = ?, error = ?, finished_at = ? WHERE id = ?`,
		status, exitCode, nullable(errMsg), time.Now().UTC(), id)
	return err
}

// AbortDevice marks unfinished jobs for a device as abandoned.
func (s *JobStore) AbortDevice(deviceID, reason string) error {
	_, err := s.db.Exec(`
		UPDATE agent_jobs
		SET status = ?, error = ?, finished_at = ?
		WHERE device_id = ? AND status IN (?, ?)`,
		JobAbandoned, reason, time.Now().UTC(), deviceID, JobPending, JobRunning)
	return err
}

const jobColumns = `id, device_id, command, COALESCE(cwd, ''), status, exit_code,
	COALESCE(error, ''), created_at, COALESCE(started_at, created_at),
	COALESCE(finished_at, created_at)`

type jobScanner interface {
	Scan(dest ...interface{}) error
}

func scanJob(row jobScanner) (*Job, error) {
	var job Job
	var created, started, finished interface{}
	if err := row.Scan(&job.ID, &job.DeviceID, &job.Command, &job.CWD, &job.Status,
		&job.ExitCode, &job.Error, &created, &started, &finished); err != nil {
		return nil, err
	}
	job.CreatedAt = scanJobTime(created)
	job.StartedAt = scanJobTime(started)
	job.FinishedAt = scanJobTime(finished)
	return &job, nil
}

// List returns jobs for a device, newest first.
func (s *JobStore) List(deviceID string) ([]Job, error) {
	rows, err := s.db.Query(`
		SELECT `+jobColumns+`
		FROM agent_jobs WHERE device_id = ? ORDER BY created_at DESC LIMIT 200`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	return jobs, rows.Err()
}

// Get returns one job.
func (s *JobStore) Get(id string) (*Job, error) {
	row := s.db.QueryRow(`
		SELECT `+jobColumns+`
		FROM agent_jobs WHERE id = ?`, id)
	return scanJob(row)
}

// Recent returns the newest jobs across all devices.
func (s *JobStore) Recent(limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`
		SELECT `+jobColumns+`
		FROM agent_jobs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	return jobs, rows.Err()
}

// ProblemJobs returns failed or abandoned jobs across all devices.
func (s *JobStore) ProblemJobs(limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`
		SELECT `+jobColumns+`
		FROM agent_jobs WHERE status IN (?, ?) ORDER BY created_at DESC LIMIT ?`,
		JobFailed, JobAbandoned, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	return jobs, rows.Err()
}

// Delete removes a job and its logs.
func (s *JobStore) Delete(id string) error {
	if _, err := s.db.Exec(`DELETE FROM agent_job_logs WHERE job_id = ?`, id); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM agent_jobs WHERE id = ?`, id)
	return err
}

// scanJobTime accepts the shapes SQLite returns: a time.Time for plain columns,
// but strings or bytes once a SQL function such as COALESCE is involved.
func scanJobTime(value interface{}) time.Time {
	switch typed := value.(type) {
	case time.Time:
		return typed
	case string:
		return parseJobTime(typed)
	case []byte:
		return parseJobTime(string(typed))
	default:
		return time.Time{}
	}
}

func parseJobTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05+00:00",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func nullable(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
