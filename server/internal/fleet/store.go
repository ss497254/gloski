package fleet

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
func (r *Store) Upsert(dev *Device) error {
	tx, err := r.db.Begin()
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
				(id, name, url, token, hostname, platform, version, stats_json,
				 deploy_id, master, machine_id, created_at, last_seen, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			dev.ID, dev.Name, dev.URL, dev.Token, dev.Hostname, dev.Platform, dev.Version,
			marshalStats(dev.Stats), nullable(dev.DeployID), dev.Master, nullable(dev.MachineID),
			dev.CreatedAt, dev.LastSeen, dev.LastSeen,
		)
	case err != nil:
		return err
	default:
		dev.ID = existingID
		_, err = tx.Exec(`
			UPDATE devices
			SET url = ?, token = ?, hostname = ?, platform = ?, version = ?,
			    stats_json = ?, deploy_id = COALESCE(NULLIF(?, ''), deploy_id),
			    master = ?, machine_id = COALESCE(NULLIF(?, ''), machine_id),
			    last_seen = ?, updated_at = ?
			WHERE id = ?`,
			dev.URL, dev.Token, dev.Hostname, dev.Platform, dev.Version,
			marshalStats(dev.Stats), dev.DeployID, dev.Master, dev.MachineID,
			dev.LastSeen, dev.LastSeen, dev.ID,
		)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Get returns a device by ID.
func (r *Store) Get(id string) (*Device, error) {
	row := r.db.QueryRow(`
		SELECT id, name, url, token, COALESCE(hostname, ''), COALESCE(platform, ''),
		       COALESCE(version, ''), stats_json, COALESCE(deploy_id, ''), master, created_at, last_seen
		FROM devices WHERE id = ?`, id)
	return scanDevice(row)
}

// GetByToken returns a device by its private token.
func (r *Store) GetByToken(token string) (*Device, error) {
	row := r.db.QueryRow(`
		SELECT id, name, url, token, COALESCE(hostname, ''), COALESCE(platform, ''),
		       COALESCE(version, ''), stats_json, COALESCE(deploy_id, ''), master, created_at, last_seen
		FROM devices WHERE token = ?`, token)
	return scanDevice(row)
}

// List returns all devices, oldest first.
func (r *Store) List() ([]Device, error) {
	rows, err := r.db.Query(`
		SELECT id, name, url, token, COALESCE(hostname, ''), COALESCE(platform, ''),
		       COALESCE(version, ''), stats_json, COALESCE(deploy_id, ''), master, created_at, last_seen
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
func (r *Store) UpdateHeartbeat(id string, stats *system.Stats, at time.Time) error {
	_, err := r.db.Exec(`
		UPDATE devices
		SET last_seen = ?, updated_at = ?, stats_json = ?
		WHERE id = ?`, at, at, marshalStats(stats), id,
	)
	return err
}

// Delete removes a device.
func (r *Store) Delete(id string) error {
	res, err := r.db.Exec("DELETE FROM devices WHERE id = ?", id)
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
		&dev.Version, &statsJSON, &dev.DeployID, &dev.Master, &dev.CreatedAt, &lastSeen,
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

// String renders a device as "name (id)" for logs.
func (d *Device) String() string {
	return fmt.Sprintf("%s (%s)", d.Name, d.ID)
}

// Identity returns the ID and machine ID of the device with this name.
func (r *Store) Identity(name string) (id, machineID string, err error) {
	var machine sql.NullString
	err = r.db.QueryRow("SELECT id, machine_id FROM devices WHERE name = ?", name).Scan(&id, &machine)
	return id, machine.String, err
}
