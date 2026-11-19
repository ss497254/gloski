package agents

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
	JobStopped   = "stopped"
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
	// last_seq is the newest output chunk stored for a job. These tables are
	// created here rather than by a migration, so the column is added the
	// same way for databases that predate it.
	var hasSeq int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('agent_jobs') WHERE name = 'last_seq'`).Scan(&hasSeq); err != nil {
		return nil, err
	}
	if hasSeq == 0 {
		if _, err := store.db.Exec(`ALTER TABLE agent_jobs ADD COLUMN last_seq INTEGER NOT NULL DEFAULT 0`); err != nil {
			return nil, err
		}
	}
	return store, nil
}

// Create stores a new pending job.
func (t *JobStore) Create(deviceID, command, cwd string) (*Job, error) {
	job := &Job{
		ID:        uuid.NewString(),
		DeviceID:  deviceID,
		Command:   command,
		CWD:       cwd,
		Status:    JobPending,
		CreatedAt: time.Now().UTC(),
	}
	_, err := t.db.Exec(`
		INSERT INTO agent_jobs (id, device_id, command, cwd, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, job.ID, deviceID, command, cwd, JobPending, job.CreatedAt)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// MarkRunning transitions a job to running.
func (t *JobStore) MarkRunning(id string) error {
	_, err := t.db.Exec(`UPDATE agent_jobs SET status = ?, started_at = ? WHERE id = ?`, JobRunning, time.Now().UTC(), id)
	return err
}

// gapNotice is written into a job's log where output was lost.
const gapNotice = "\n[gloski: some output was lost while the agent was disconnected]\n"

// AppendLog stores one streamed output chunk. seq orders a job's chunks:
// one already stored is dropped (a replay after reconnect), and a jump is
// recorded as a gap. seq 0 means the agent does not number its output.
func (t *JobStore) AppendLog(id string, seq uint64, chunk []byte, stderr bool) error {
	if len(chunk) == 0 {
		return nil
	}
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var last uint64
	if err := tx.QueryRow(`SELECT last_seq FROM agent_jobs WHERE id = ?`, id).Scan(&last); err != nil {
		if err == sql.ErrNoRows {
			return nil // job was deleted; discard its output
		}
		return err
	}
	now := time.Now().UTC()
	if seq != 0 {
		if seq <= last {
			return nil
		}
		if seq > last+1 {
			if _, err := tx.Exec(`
				INSERT INTO agent_job_logs (job_id, chunk, stderr, created_at) VALUES (?, ?, 1, ?)`,
				id, []byte(gapNotice), now); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`UPDATE agent_jobs SET last_seq = ? WHERE id = ?`, seq, id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`
		INSERT INTO agent_job_logs (job_id, chunk, stderr, created_at) VALUES (?, ?, ?, ?)`,
		id, chunk, stderr, now); err != nil {
		return err
	}
	return tx.Commit()
}

// Resume reports how to pick up a job an agent says it still holds: the
// newest chunk stored, and whether the master is still waiting on the job.
func (t *JobStore) Resume(deviceID, id string) (ackedSeq uint64, active bool) {
	var status string
	err := t.db.QueryRow(`SELECT last_seq, status FROM agent_jobs WHERE id = ? AND device_id = ?`, id, deviceID).Scan(&ackedSeq, &status)
	if err != nil {
		return 0, false
	}
	return ackedSeq, status == JobPending || status == JobRunning
}

// AbandonMissing marks a device's unfinished jobs as abandoned unless the
// agent still holds them. It is the reckoning after a reconnect: a job the
// agent no longer knows about died with a previous agent process.
func (t *JobStore) AbandonMissing(deviceID string, held map[string]bool, reason string) error {
	rows, err := t.db.Query(`SELECT id FROM agent_jobs WHERE device_id = ? AND status IN (?, ?)`, deviceID, JobPending, JobRunning)
	if err != nil {
		return err
	}
	var missing []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		if !held[id] {
			missing = append(missing, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range missing {
		if err := t.Abandon(id, reason); err != nil {
			return err
		}
	}
	return nil
}

// Abandon gives up on one unfinished job.
func (t *JobStore) Abandon(id, reason string) error {
	_, err := t.db.Exec(`
		UPDATE agent_jobs SET status = ?, error = ?, finished_at = ?
		WHERE id = ? AND status IN (?, ?)`,
		JobAbandoned, reason, time.Now().UTC(), id, JobPending, JobRunning)
	return err
}

// Logs returns the complete log as lines.
func (t *JobStore) Logs(id string) ([]string, error) {
	rows, err := t.db.Query(`SELECT chunk FROM agent_job_logs WHERE job_id = ? ORDER BY id ASC`, id)
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

// Finish records a job's terminal result. stopped marks a job the user
// cancelled. A job that already reached a terminal state is left alone, so a
// late result cannot overwrite it.
func (t *JobStore) Finish(id string, exitCode int, errMsg string, stopped bool) error {
	status := JobFinished
	switch {
	case stopped:
		status = JobStopped
		errMsg = ""
	case errMsg != "" || exitCode != 0:
		status = JobFailed
	}
	_, err := t.db.Exec(`
		UPDATE agent_jobs SET status = ?, exit_code = ?, error = ?, finished_at = ?
		WHERE id = ? AND status IN (?, ?)`,
		status, exitCode, nullable(errMsg), time.Now().UTC(), id, JobPending, JobRunning)
	return err
}

// DeleteDevice removes every job and log belonging to a device.
func (t *JobStore) DeleteDevice(deviceID string) error {
	if _, err := t.db.Exec(`
		DELETE FROM agent_job_logs
		WHERE job_id IN (SELECT id FROM agent_jobs WHERE device_id = ?)`, deviceID); err != nil {
		return err
	}
	_, err := t.db.Exec(`DELETE FROM agent_jobs WHERE device_id = ?`, deviceID)
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
func (t *JobStore) List(deviceID string) ([]Job, error) {
	rows, err := t.db.Query(`
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
func (t *JobStore) Get(id string) (*Job, error) {
	row := t.db.QueryRow(`
		SELECT `+jobColumns+`
		FROM agent_jobs WHERE id = ?`, id)
	return scanJob(row)
}

// Recent returns the newest jobs across all fleet.
func (t *JobStore) Recent(limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := t.db.Query(`
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

// ProblemJobs returns failed or abandoned jobs across all fleet.
func (t *JobStore) ProblemJobs(limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := t.db.Query(`
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
func (t *JobStore) Delete(id string) error {
	if _, err := t.db.Exec(`DELETE FROM agent_job_logs WHERE job_id = ?`, id); err != nil {
		return err
	}
	_, err := t.db.Exec(`DELETE FROM agent_jobs WHERE id = ?`, id)
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

// Active counts a device's jobs that are still pending or running.
func (t *JobStore) Active(deviceID string) int {
	var count int
	_ = t.db.QueryRow(`
		SELECT COUNT(*) FROM agent_jobs
		WHERE device_id = ? AND status IN ('pending', 'running')`, deviceID).Scan(&count)
	return count
}
