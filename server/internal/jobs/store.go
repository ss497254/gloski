package jobs

import (
	"database/sql"
	"time"

	"github.com/ss497254/gloski/internal/database"
)

// Store handles persistence of jobs to SQLite database
type Store struct {
	db *sql.DB
}

// NewStore creates a new store with the given database
func NewStore(database *database.Database) *Store {
	return &Store{
		db: database.DB(),
	}
}

// Load reads all jobs from the database
func (s *Store) Load() ([]*Job, error) {
	rows, err := s.db.Query(`
		SELECT id, command, cwd, status, pid, exit_code, log_file,
		       created_at, started_at, finished_at
		FROM jobs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		j := &Job{}
		var pid, exitCode sql.NullInt64
		var logFile sql.NullString
		var startedAt, finishedAt sql.NullTime

		err := rows.Scan(
			&j.ID, &j.Command, &j.Cwd, &j.Status,
			&pid, &exitCode, &logFile,
			&j.CreatedAt, &startedAt, &finishedAt,
		)
		if err != nil {
			return nil, err
		}

		if pid.Valid {
			j.PID = int(pid.Int64)
		}
		if exitCode.Valid {
			j.ExitCode = int(exitCode.Int64)
		}
		if logFile.Valid {
			j.LogFile = logFile.String
		}
		if startedAt.Valid {
			j.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			j.FinishedAt = &finishedAt.Time
		}

		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// Get retrieves a single job by ID
func (s *Store) Get(id string) (*Job, error) {
	j := &Job{}
	var pid, exitCode sql.NullInt64
	var logFile sql.NullString
	var startedAt, finishedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, command, cwd, status, pid, exit_code, log_file,
		       created_at, started_at, finished_at
		FROM jobs WHERE id = ?
	`, id).Scan(
		&j.ID, &j.Command, &j.Cwd, &j.Status,
		&pid, &exitCode, &logFile,
		&j.CreatedAt, &startedAt, &finishedAt,
	)
	if err != nil {
		return nil, err
	}

	if pid.Valid {
		j.PID = int(pid.Int64)
	}
	if exitCode.Valid {
		j.ExitCode = int(exitCode.Int64)
	}
	if logFile.Valid {
		j.LogFile = logFile.String
	}
	if startedAt.Valid {
		j.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		j.FinishedAt = &finishedAt.Time
	}

	return j, nil
}

// Insert adds a new job to the database
func (s *Store) Insert(j *Job) error {
	_, err := s.db.Exec(`
		INSERT INTO jobs (id, command, cwd, status, pid, exit_code, log_file,
		                  created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		j.ID, j.Command, j.Cwd, j.Status,
		nullInt(j.PID), nullInt(j.ExitCode), nullString(j.LogFile),
		j.CreatedAt, nullTime(j.StartedAt), nullTime(j.FinishedAt),
	)
	return err
}

// Update updates an existing job in the database
func (s *Store) Update(j *Job) error {
	_, err := s.db.Exec(`
		UPDATE jobs SET
			command = ?, cwd = ?, status = ?, pid = ?, exit_code = ?,
			log_file = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`,
		j.Command, j.Cwd, j.Status, nullInt(j.PID), nullInt(j.ExitCode),
		nullString(j.LogFile), nullTime(j.StartedAt), nullTime(j.FinishedAt),
		j.ID,
	)
	return err
}

// Delete removes a job from the database
func (s *Store) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM jobs WHERE id = ?", id)
	return err
}

// Count returns the total number of jobs
func (s *Store) Count() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM jobs").Scan(&count)
	return count, err
}

// GetOldestJobs returns the oldest jobs beyond the limit
func (s *Store) GetOldestJobs(keepCount int) ([]*Job, error) {
	rows, err := s.db.Query(`
		SELECT id, command, cwd, status, pid, exit_code, log_file,
		       created_at, started_at, finished_at
		FROM jobs
		WHERE status NOT IN ('running', 'pending')
		ORDER BY created_at DESC
		LIMIT -1 OFFSET ?
	`, keepCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		j := &Job{}
		var pid, exitCode sql.NullInt64
		var logFile sql.NullString
		var startedAt, finishedAt sql.NullTime

		err := rows.Scan(
			&j.ID, &j.Command, &j.Cwd, &j.Status,
			&pid, &exitCode, &logFile,
			&j.CreatedAt, &startedAt, &finishedAt,
		)
		if err != nil {
			return nil, err
		}

		if pid.Valid {
			j.PID = int(pid.Int64)
		}
		if exitCode.Valid {
			j.ExitCode = int(exitCode.Int64)
		}
		if logFile.Valid {
			j.LogFile = logFile.String
		}
		if startedAt.Valid {
			j.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			j.FinishedAt = &finishedAt.Time
		}

		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// Helper functions for nullable fields
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
