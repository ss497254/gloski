package checks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/ss497254/gloski/internal/database"
)

// HistoryLimit is how many results are kept per (check, device) pair.
const HistoryLimit = 20

// Store persists check results in the master database.
type Store struct {
	db *sql.DB
}

// NewStore creates a check result store backed by the master database.
func NewStore(db *database.Database) *Store {
	return &Store{db: db.DB()}
}

// Save records one run of a check. It keeps the whole history; call Prune from
// the maintenance pass to bound it.
func (s *Store) Save(ctx context.Context, result Result) error {
	details := result.Details
	if details == nil {
		details = []Detail{}
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO check_results
			(check_id, device_id, status, summary, details_json, ran_at, duration_ms, interval_ns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		result.CheckID, result.DeviceID, string(result.Status), result.Summary,
		string(encoded), result.RanAt.UTC(), result.DurationMS, result.Interval.Nanoseconds(),
	)
	return err
}

// Latest returns the most recent result for one check on one device.
func (s *Store) Latest(ctx context.Context, checkID, deviceID string) (Result, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT check_id, device_id, status, summary, details_json, ran_at, duration_ms, interval_ns
		FROM check_results
		WHERE check_id = ? AND device_id = ?
		ORDER BY ran_at DESC, id DESC LIMIT 1`, checkID, deviceID)
	result, err := scanResult(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, false, nil
	}
	if err != nil {
		return Result{}, false, err
	}
	result.RefreshStale(time.Now())
	return result, true, nil
}

// LatestForDevice returns one result per check, newest first by check ID.
func (s *Store) LatestForDevice(ctx context.Context, deviceID string) ([]Result, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.check_id, r.device_id, r.status, r.summary, r.details_json, r.ran_at, r.duration_ms, r.interval_ns
		FROM check_results r
		WHERE r.device_id = ? AND r.id = (
			SELECT id FROM check_results
			WHERE check_id = r.check_id AND device_id = r.device_id
			ORDER BY ran_at DESC, id DESC LIMIT 1
		)
		ORDER BY r.check_id`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResults(rows)
}

// LatestForAllDevices returns one result per (check, device) pair across the
// fleet, sorted by device then check. Staleness is recomputed on read like
// every other result query.
func (s *Store) LatestForAllDevices(ctx context.Context) ([]Result, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.check_id, r.device_id, r.status, r.summary, r.details_json, r.ran_at, r.duration_ms, r.interval_ns
		FROM check_results r
		WHERE r.id = (
			SELECT id FROM check_results
			WHERE check_id = r.check_id AND device_id = r.device_id
			ORDER BY ran_at DESC, id DESC LIMIT 1
		)
		ORDER BY r.device_id, r.check_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResults(rows)
}

// History returns the last limit results for one check on one device.
func (s *Store) History(ctx context.Context, checkID, deviceID string, limit int) ([]Result, error) {
	if limit <= 0 {
		limit = HistoryLimit
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT check_id, device_id, status, summary, details_json, ran_at, duration_ms, interval_ns
		FROM check_results
		WHERE check_id = ? AND device_id = ?
		ORDER BY ran_at DESC, id DESC LIMIT ?`, checkID, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResults(rows)
}

// Prune keeps only the newest keep results per (check, device) pair. It is
// meant to be called by the daily maintenance pass, not after every save.
func (s *Store) Prune(ctx context.Context, keep int) (int64, error) {
	if keep < 1 {
		keep = HistoryLimit
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, check_id, device_id FROM check_results
		ORDER BY check_id, device_id, ran_at DESC, id DESC`)
	if err != nil {
		return 0, err
	}
	var stale []int64
	seen := make(map[[2]string]int)
	for rows.Next() {
		var id int64
		var checkID, deviceID string
		if err := rows.Scan(&id, &checkID, &deviceID); err != nil {
			rows.Close()
			return 0, err
		}
		key := [2]string{checkID, deviceID}
		seen[key]++
		if seen[key] > keep {
			stale = append(stale, id)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	for _, id := range stale {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM check_results WHERE id = ?", id); err != nil {
			return 0, err
		}
	}
	return int64(len(stale)), nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanResult(row scanner) (Result, error) {
	var result Result
	var status, detailsJSON string
	var durationMS, intervalNS int64
	err := row.Scan(
		&result.CheckID, &result.DeviceID, &status, &result.Summary,
		&detailsJSON, &result.RanAt, &durationMS, &intervalNS,
	)
	if err != nil {
		return Result{}, err
	}
	result.Status = Status(status)
	result.DurationMS = durationMS
	result.Interval = time.Duration(intervalNS)
	// A malformed details blob must not hide the summary and verdict.
	_ = json.Unmarshal([]byte(detailsJSON), &result.Details)
	return result, nil
}

func scanResults(rows *sql.Rows) ([]Result, error) {
	results := []Result{}
	for rows.Next() {
		result, err := scanResult(rows)
		if err != nil {
			return nil, err
		}
		result.RefreshStale(time.Now())
		results = append(results, result)
	}
	return results, rows.Err()
}
