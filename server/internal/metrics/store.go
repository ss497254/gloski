// Package metrics stores a small, bounded history of device metrics on the
// master. Samples are written from the heartbeat path, at most once per sample
// interval, and thinned by the daily maintenance pass.
package metrics

import (
	"context"
	"database/sql"
	"time"

	"github.com/ss497254/gloski/internal/database"
)

const (
	// SampleInterval is the shortest gap between two samples of one device.
	// Heartbeats are much more frequent; only one in every few is persisted.
	SampleInterval = 5 * time.Minute

	// FullDays is how long samples stay at full resolution. Older samples are
	// reduced to one per day, up to MaxDays.
	FullDays = 7
	MaxDays  = 365

	defaultRange = 24 * time.Hour
	defaultLimit = 2000

	// MaxHistoryLimit is the largest number of samples a history query may
	// request. The HTTP handler rejects anything above it.
	MaxHistoryLimit = 20000
)

// Sample is one metric value for one device at one instant.
type Sample struct {
	DeviceID string    `json:"device_id"`
	Metric   string    `json:"metric"`
	TS       time.Time `json:"ts"`
	Value    float64   `json:"value"`
}

// Store persists metric samples in the master database.
type Store struct {
	db *sql.DB
}

// NewStore creates a metric history store backed by the master database.
func NewStore(db *database.Database) *Store {
	return &Store{db: db.DB()}
}

// dayKey is the UTC day a sample belongs to. It is stored explicitly because
// the driver writes time.Time in a format SQLite's date() cannot parse.
func dayKey(ts time.Time) string {
	return ts.UTC().Format("2006-01-02")
}

// recentMetrics returns the metrics sampled within the interval. The query is
// covered by idx_metric_samples_latest.
func (s *Store) recentMetrics(ctx context.Context, deviceID string, since time.Time) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT metric FROM metric_samples WHERE device_id = ? AND ts > ?`, deviceID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recent := make(map[string]bool)
	for rows.Next() {
		var metric string
		if err := rows.Scan(&metric); err != nil {
			return nil, err
		}
		recent[metric] = true
	}
	return recent, rows.Err()
}

// Record saves the samples that are due for a device. Each (device, metric)
// is throttled independently: a metric that first appears mid-interval must
// not be dropped just because its peers were sampled a moment earlier. The
// check reads stored timestamps, so a master restart does not shorten the
// interval. It returns whether anything was written.
func (s *Store) Record(ctx context.Context, deviceID string, ts time.Time, samples []Sample) (bool, error) {
	if len(samples) == 0 || deviceID == "" {
		return false, nil
	}
	ts = ts.UTC()
	cutoff := ts.Add(-SampleInterval)

	recent, err := s.recentMetrics(ctx, deviceID, cutoff)
	if err != nil {
		return false, err
	}
	due := make([]Sample, 0, len(samples))
	for _, sample := range samples {
		if sample.Metric == "" || recent[sample.Metric] {
			continue
		}
		due = append(due, sample)
		recent[sample.Metric] = true // also dedupes within one batch
	}
	if len(due) == 0 {
		return false, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metric_samples (device_id, metric, ts, day, value) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	for _, sample := range due {
		if _, err := stmt.ExecContext(ctx, deviceID, sample.Metric, ts, dayKey(ts), sample.Value); err != nil {
			return false, err
		}
	}
	if err := stmt.Close(); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// History returns samples in chronological order, ready for a chart. An empty
// metric returns every metric for the device. The range and count are bounded
// so a long-lived device cannot make an unbounded query.
func (s *Store) History(ctx context.Context, deviceID, metric string, within time.Duration, limit int) ([]Sample, error) {
	if within <= 0 {
		within = defaultRange
	}
	if limit <= 0 || limit > MaxHistoryLimit {
		limit = defaultLimit
	}
	since := time.Now().UTC().Add(-within)

	query := `
		SELECT device_id, metric, ts, value FROM metric_samples
		WHERE device_id = ? AND ts >= ?`
	args := []interface{}{deviceID, since}
	if metric != "" {
		query += " AND metric = ?"
		args = append(args, metric)
	}
	// Take the newest rows, then reverse in Go. Ordering oldest-first with a
	// LIMIT would return the start of a long history and leave charts stale.
	query += " ORDER BY ts DESC, metric DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := []Sample{}
	for rows.Next() {
		var sample Sample
		if err := rows.Scan(&sample.DeviceID, &sample.Metric, &sample.TS, &sample.Value); err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(samples)-1; i < j; i, j = i+1, j-1 {
		samples[i], samples[j] = samples[j], samples[i]
	}
	return samples, nil
}

// Thin keeps full resolution for FullDays, then one sample per (device, metric,
// day), and drops everything older than MaxDays. It is meant to be called by
// the daily maintenance pass, not after every write.
func (s *Store) Thin(ctx context.Context, now time.Time) (int64, error) {
	now = now.UTC()
	fullCutoff := now.Add(-FullDays * 24 * time.Hour)
	maxCutoff := now.Add(-MaxDays * 24 * time.Hour)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Keep the earliest sample of each old day. Comparing the stored day key,
	// rather than calling SQLite's date(), keeps this independent of how the
	// driver formats time.Time.
	thinned, err := tx.ExecContext(ctx, `
		DELETE FROM metric_samples
		WHERE ts < ? AND EXISTS (
			SELECT 1 FROM metric_samples AS keep
			WHERE keep.device_id = metric_samples.device_id
			  AND keep.metric = metric_samples.metric
			  AND keep.day = metric_samples.day
			  AND keep.ts < metric_samples.ts
		)`, fullCutoff)
	if err != nil {
		return 0, err
	}
	count, err := thinned.RowsAffected()
	if err != nil {
		return 0, err
	}

	expired, err := tx.ExecContext(ctx, `DELETE FROM metric_samples WHERE ts < ?`, maxCutoff)
	if err != nil {
		return 0, err
	}
	expiredCount, err := expired.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count + expiredCount, nil
}
