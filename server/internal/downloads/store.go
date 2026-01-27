package downloads

import (
	"database/sql"
	"time"

	"github.com/ss497254/gloski/internal/database"
)

// Store handles persistence of downloads to SQLite database
type Store struct {
	db *sql.DB
}

// NewStore creates a new store with the given database
func NewStore(database *database.Database) *Store {
	return &Store{
		db: database.DB(),
	}
}

// Load reads all downloads from the database
func (s *Store) Load() ([]*Download, error) {
	rows, err := s.db.Query(`
		SELECT id, url, destination, filename, file_path, status, progress, total, speed,
		       error, retries, max_retries, created_at, started_at, completed_at
		FROM downloads
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []*Download
	for rows.Next() {
		d := &Download{}
		var errorStr sql.NullString
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&d.ID, &d.URL, &d.Destination, &d.Filename, &d.FilePath,
			&d.Status, &d.Progress, &d.Total, &d.Speed,
			&errorStr, &d.Retries, &d.MaxRetries,
			&d.CreatedAt, &startedAt, &completedAt,
		)
		if err != nil {
			return nil, err
		}

		if errorStr.Valid {
			d.Error = errorStr.String
		}
		if startedAt.Valid {
			d.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			d.CompletedAt = &completedAt.Time
		}

		// Load share links for this download
		d.ShareLinks, err = s.loadShareLinks(d.ID)
		if err != nil {
			return nil, err
		}

		downloads = append(downloads, d)
	}

	return downloads, rows.Err()
}

// Get retrieves a single download by ID
func (s *Store) Get(id string) (*Download, error) {
	d := &Download{}
	var errorStr sql.NullString
	var startedAt, completedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, url, destination, filename, file_path, status, progress, total, speed,
		       error, retries, max_retries, created_at, started_at, completed_at
		FROM downloads WHERE id = ?
	`, id).Scan(
		&d.ID, &d.URL, &d.Destination, &d.Filename, &d.FilePath,
		&d.Status, &d.Progress, &d.Total, &d.Speed,
		&errorStr, &d.Retries, &d.MaxRetries,
		&d.CreatedAt, &startedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	if errorStr.Valid {
		d.Error = errorStr.String
	}
	if startedAt.Valid {
		d.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		d.CompletedAt = &completedAt.Time
	}

	// Load share links
	d.ShareLinks, err = s.loadShareLinks(d.ID)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Insert adds a new download to the database
func (s *Store) Insert(d *Download) error {
	_, err := s.db.Exec(`
		INSERT INTO downloads (id, url, destination, filename, file_path, status, progress, total, speed,
		                       error, retries, max_retries, created_at, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		d.ID, d.URL, d.Destination, d.Filename, d.FilePath,
		d.Status, d.Progress, d.Total, d.Speed,
		nullString(d.Error), d.Retries, d.MaxRetries,
		d.CreatedAt, nullTime(d.StartedAt), nullTime(d.CompletedAt),
	)
	return err
}

// Update updates an existing download in the database
func (s *Store) Update(d *Download) error {
	_, err := s.db.Exec(`
		UPDATE downloads SET
			url = ?, destination = ?, filename = ?, file_path = ?, status = ?,
			progress = ?, total = ?, speed = ?, error = ?, retries = ?, max_retries = ?,
			started_at = ?, completed_at = ?
		WHERE id = ?
	`,
		d.URL, d.Destination, d.Filename, d.FilePath, d.Status,
		d.Progress, d.Total, d.Speed, nullString(d.Error), d.Retries, d.MaxRetries,
		nullTime(d.StartedAt), nullTime(d.CompletedAt),
		d.ID,
	)
	return err
}

// Delete removes a download from the database
func (s *Store) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM downloads WHERE id = ?", id)
	return err
}

// UpdateStatus updates just the status and related fields
func (s *Store) UpdateStatus(id string, status DownloadStatus, errorMsg string) error {
	_, err := s.db.Exec(`
		UPDATE downloads SET status = ?, error = ? WHERE id = ?
	`, status, nullString(errorMsg), id)
	return err
}

// UpdateProgress updates progress and speed for a download
func (s *Store) UpdateProgress(id string, progress, total, speed int64) error {
	_, err := s.db.Exec(`
		UPDATE downloads SET progress = ?, total = ?, speed = ? WHERE id = ?
	`, progress, total, speed, id)
	return err
}

// loadShareLinks loads all share links for a download
func (s *Store) loadShareLinks(downloadID string) ([]ShareLink, error) {
	rows, err := s.db.Query(`
		SELECT token, url, created_at, expires_at
		FROM download_share_links
		WHERE download_id = ?
		ORDER BY created_at DESC
	`, downloadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []ShareLink
	for rows.Next() {
		var link ShareLink
		var expiresAt sql.NullTime

		if err := rows.Scan(&link.Token, &link.URL, &link.CreatedAt, &expiresAt); err != nil {
			return nil, err
		}

		if expiresAt.Valid {
			link.ExpiresAt = &expiresAt.Time
		}

		links = append(links, link)
	}

	if links == nil {
		links = []ShareLink{}
	}

	return links, rows.Err()
}

// InsertShareLink adds a new share link for a download
func (s *Store) InsertShareLink(downloadID string, link *ShareLink) error {
	_, err := s.db.Exec(`
		INSERT INTO download_share_links (download_id, token, url, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, downloadID, link.Token, link.URL, link.CreatedAt, nullTime(link.ExpiresAt))
	return err
}

// DeleteShareLink removes a share link by token
func (s *Store) DeleteShareLink(downloadID, token string) error {
	result, err := s.db.Exec(`
		DELETE FROM download_share_links WHERE download_id = ? AND token = ?
	`, downloadID, token)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetByShareToken finds a download by share token
func (s *Store) GetByShareToken(token string) (*Download, *ShareLink, error) {
	var downloadID string
	var link ShareLink
	var expiresAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT download_id, token, url, created_at, expires_at
		FROM download_share_links
		WHERE token = ?
	`, token).Scan(&downloadID, &link.Token, &link.URL, &link.CreatedAt, &expiresAt)
	if err != nil {
		return nil, nil, err
	}

	if expiresAt.Valid {
		link.ExpiresAt = &expiresAt.Time
	}

	// Check if expired
	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
		return nil, nil, sql.ErrNoRows
	}

	download, err := s.Get(downloadID)
	if err != nil {
		return nil, nil, err
	}

	return download, &link, nil
}

// Helper functions for nullable fields
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
