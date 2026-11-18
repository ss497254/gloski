package devices

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/system"
)

// Store persists devices in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a device store.
func NewStore(db *database.Database) *Store {
	return &Store{db: db.DB()}
}

// Upsert inserts a new device or refreshes an existing one with the same name.
func (s *Store) Upsert(dev *Device) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingID string
	err = tx.QueryRow("SELECT id FROM devices WHERE name = ?", dev.Name).Scan(&existingID)
	switch {
	case err == sql.ErrNoRows:
		_, err = tx.Exec(`
			INSERT INTO devices
				(id, name, url, token, hostname, platform, version, stats_json, created_at, last_seen, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			dev.ID, dev.Name, dev.URL, dev.Token, dev.Hostname, dev.Platform, dev.Version,
			marshalStats(dev.Stats), dev.CreatedAt, dev.LastSeen, dev.LastSeen,
		)
	case err != nil:
		return err
	default:
		dev.ID = existingID
		_, err = tx.Exec(`
			UPDATE devices
			SET url = ?, token = ?, hostname = ?, platform = ?, version = ?,
			    stats_json = ?, last_seen = ?, updated_at = ?
			WHERE id = ?`,
			dev.URL, dev.Token, dev.Hostname, dev.Platform, dev.Version,
			marshalStats(dev.Stats), dev.LastSeen, dev.LastSeen, dev.ID,
		)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Get returns a device by ID.
func (s *Store) Get(id string) (*Device, error) {
	row := s.db.QueryRow(`
		SELECT id, name, url, token, COALESCE(hostname, ''), COALESCE(platform, ''),
		       COALESCE(version, ''), stats_json, created_at, last_seen
		FROM devices WHERE id = ?`, id)
	return scanDevice(row)
}

// GetByToken returns a device by its private token.
func (s *Store) GetByToken(token string) (*Device, error) {
	row := s.db.QueryRow(`
		SELECT id, name, url, token, COALESCE(hostname, ''), COALESCE(platform, ''),
		       COALESCE(version, ''), stats_json, created_at, last_seen
		FROM devices WHERE token = ?`, token)
	return scanDevice(row)
}

// List returns all devices, oldest first.
func (s *Store) List() ([]Device, error) {
	rows, err := s.db.Query(`
		SELECT id, name, url, token, COALESCE(hostname, ''), COALESCE(platform, ''),
		       COALESCE(version, ''), stats_json, created_at, last_seen
		FROM devices ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Device
	for rows.Next() {
		dev, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *dev)
	}
	return result, rows.Err()
}

// UpdateHeartbeat refreshes liveness and cached stats for a device.
func (s *Store) UpdateHeartbeat(id, url, version string, stats *system.Stats, at time.Time) error {
	_, err := s.db.Exec(`
		UPDATE devices
		SET last_seen = ?, updated_at = ?,
		    url = CASE WHEN ? <> '' THEN ? ELSE url END,
		    version = CASE WHEN ? <> '' THEN ? ELSE version END,
		    stats_json = ?
		WHERE id = ?`,
		at, at, url, url, version, version, marshalStats(stats), id,
	)
	return err
}

// Delete removes a device.
func (s *Store) Delete(id string) error {
	res, err := s.db.Exec("DELETE FROM devices WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanDevice(row scanner) (*Device, error) {
	var dev Device
	var statsJSON sql.NullString
	var lastSeen sql.NullTime
	err := row.Scan(
		&dev.ID, &dev.Name, &dev.URL, &dev.Token, &dev.Hostname, &dev.Platform,
		&dev.Version, &statsJSON, &dev.CreatedAt, &lastSeen,
	)
	if err != nil {
		return nil, err
	}
	if lastSeen.Valid {
		dev.LastSeen = lastSeen.Time
	} else {
		dev.LastSeen = dev.CreatedAt
	}
	if statsJSON.Valid && statsJSON.String != "" {
		var stats system.Stats
		if err := json.Unmarshal([]byte(statsJSON.String), &stats); err == nil {
			dev.Stats = &stats
		}
	}
	return &dev, nil
}

func marshalStats(stats *system.Stats) string {
	if stats == nil {
		return ""
	}
	data, err := json.Marshal(stats)
	if err != nil {
		return ""
	}
	return string(data)
}

func (d *Device) String() string {
	return fmt.Sprintf("%s (%s)", d.Name, d.ID)
}
