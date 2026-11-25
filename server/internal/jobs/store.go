// Package jobs persists master-owned tasks, statuses, and streamed logs.
package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/database"
)

// ErrJobNotFound means no job exists with that ID.
var ErrJobNotFound = errors.New("job not found")

// Job status values stored by the master.
const (
	JobPending   = "pending"
	JobRunning   = "running"
	JobFinished  = "finished"
	JobFailed    = "failed"
	JobStopped   = "stopped"
	JobAbandoned = "abandoned"
)

// Job log limits. A chatty job must not fill the master's disk, so each job
// keeps at most 10 MB and finished jobs are removed after 30 days.
const (
	maxJobLogBytes = 10 << 20
	JobRetention   = 30 * 24 * time.Hour
)

const truncationNotice = "\n[gloski: job log truncated after 10 MB]\n"

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

// Store persists jobs and their streamed logs in the master database.
type Store struct {
	db *sql.DB
}

// NewStore creates the master-side job store after database migrations run.
func NewStore(db *database.Database) *Store {
	return &Store{db: db.DB()}
}

// Create stores a new pending job.
func (t *Store) Create(deviceID, command, cwd string) (*Job, error) {
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
func (t *Store) MarkRunning(id string) error {
	_, err := t.db.Exec(`UPDATE agent_jobs SET status = ?, started_at = ? WHERE id = ?`, JobRunning, time.Now().UTC(), id)
	return err
}

// gapNotice is written into a job's log where output was lost.
const gapNotice = "\n[gloski: some output was lost while the agent was disconnected]\n"

// AppendLog stores one streamed output chunk. seq orders a job's chunks:
// one already stored is dropped (a replay after reconnect), and a jump is
// recorded as a gap. seq 0 means the agent does not number its output.
func (t *Store) AppendLog(id string, seq uint64, chunk []byte, stderr bool) error {
	if len(chunk) == 0 {
		return nil
	}
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var last uint64
	var logBytes int64
	var truncated bool
	if err := tx.QueryRow(`SELECT last_seq, log_bytes, log_truncated FROM agent_jobs WHERE id = ?`, id).Scan(&last, &logBytes, &truncated); err != nil {
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
			if err := appendChunk(tx, id, []byte(gapNotice), true, now, &logBytes, &truncated); err != nil {
				return err
			}
		}
		// last_seq advances even when the cap drops the chunk, so a reconnect
		// does not replay output the master has already decided to discard.
		if _, err := tx.Exec(`UPDATE agent_jobs SET last_seq = ? WHERE id = ?`, seq, id); err != nil {
			return err
		}
	}
	if err := appendChunk(tx, id, chunk, stderr, now, &logBytes, &truncated); err != nil {
		return err
	}
	return tx.Commit()
}

// appendChunk stores one log chunk unless the job is already at its limit. The
// chunk that crosses the limit is replaced by a single truncation notice.
func appendChunk(tx *sql.Tx, id string, chunk []byte, stderr bool, now time.Time, logBytes *int64, truncated *bool) error {
	if *truncated {
		return nil
	}
	if *logBytes+int64(len(chunk)) > maxJobLogBytes {
		if _, err := tx.Exec(`
			INSERT INTO agent_job_logs (job_id, chunk, stderr, created_at) VALUES (?, ?, 1, ?)`,
			id, []byte(truncationNotice), now); err != nil {
			return err
		}
		*truncated = true
		_, err := tx.Exec(`UPDATE agent_jobs SET log_truncated = 1 WHERE id = ?`, id)
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO agent_job_logs (job_id, chunk, stderr, created_at) VALUES (?, ?, ?, ?)`,
		id, chunk, stderr, now); err != nil {
		return err
	}
	*logBytes += int64(len(chunk))
	_, err := tx.Exec(`UPDATE agent_jobs SET log_bytes = ? WHERE id = ?`, *logBytes, id)
	return err
}

// Resume reports how to pick up a job an agent says it still holds: the
// newest chunk stored, and whether the master is still waiting on the job.
func (t *Store) Resume(deviceID, id string) (ackedSeq uint64, active bool) {
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
func (t *Store) AbandonMissing(deviceID string, held map[string]bool, reason string) error {
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
func (t *Store) Abandon(id, reason string) error {
	_, err := t.db.Exec(`
		UPDATE agent_jobs SET status = ?, error = ?, finished_at = ?
		WHERE id = ? AND status IN (?, ?)`,
		JobAbandoned, reason, time.Now().UTC(), id, JobPending, JobRunning)
	return err
}

// Logs returns the complete log as lines.
func (t *Store) Logs(id string) ([]string, error) {
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
func (t *Store) Finish(id string, exitCode int, errMsg string, stopped bool) error {
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

// PruneFinished deletes terminal jobs older than cutoff along with their
// logs. Running and pending jobs are never touched.
func (t *Store) PruneFinished(cutoff time.Time) (int64, error) {
	if _, err := t.db.Exec(`
		DELETE FROM agent_job_logs WHERE job_id IN (
			SELECT id FROM agent_jobs
			WHERE status NOT IN (?, ?) AND COALESCE(finished_at, created_at) < ?)`,
		JobPending, JobRunning, cutoff); err != nil {
		return 0, err
	}
	result, err := t.db.Exec(`
		DELETE FROM agent_jobs
		WHERE status NOT IN (?, ?) AND COALESCE(finished_at, created_at) < ?`,
		JobPending, JobRunning, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteDevice removes every job and log belonging to a device.
func (t *Store) DeleteDevice(deviceID string) error {
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
func (t *Store) List(deviceID string) ([]Job, error) {
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
func (t *Store) Get(id string) (*Job, error) {
	row := t.db.QueryRow(`
		SELECT `+jobColumns+`
		FROM agent_jobs WHERE id = ?`, id)
	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("load job %s: %w", id, err)
	}
	return job, nil
}

// Recent returns the newest jobs across all fleet.
func (t *Store) Recent(limit int) ([]Job, error) {
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

// ProblemCountsByDevice counts failed and abandoned jobs per device without
// the fleet-wide limit used by ProblemJobs. The matrix needs every device's
// count, not just the ten newest problems in the fleet.
func (t *Store) ProblemCountsByDevice(ctx context.Context) (map[string]int, error) {
	rows, err := t.db.QueryContext(ctx, `
		SELECT device_id, COUNT(*) FROM agent_jobs
		WHERE status IN (?, ?) GROUP BY device_id`, JobFailed, JobAbandoned)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var deviceID string
		var count int
		if err := rows.Scan(&deviceID, &count); err != nil {
			return nil, err
		}
		counts[deviceID] = count
	}
	return counts, rows.Err()
}

// ProblemJobs returns failed or abandoned jobs across all fleet.
func (t *Store) ProblemJobs(limit int) ([]Job, error) {
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
func (t *Store) Delete(id string) error {
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
func (t *Store) Active(deviceID string) int {
	var count int
	_ = t.db.QueryRow(`
		SELECT COUNT(*) FROM agent_jobs
		WHERE device_id = ? AND status IN ('pending', 'running')`, deviceID).Scan(&count)
	return count
}
