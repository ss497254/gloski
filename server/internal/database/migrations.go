package database

import (
	"fmt"
)

// migrate runs all database migrations.
func (d *Database) migrate() error {
	// Create migrations table if not exists
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	row := d.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	// Run pending migrations
	for _, m := range migrations {
		if m.version > currentVersion {
			if err := d.runMigration(m); err != nil {
				return fmt.Errorf("migration %d failed: %w", m.version, err)
			}
		}
	}

	return nil
}

type migration struct {
	version int
	name    string
	sql     string
}

func (d *Database) runMigration(m migration) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(m.sql); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// migrations is the list of all database migrations.
var migrations = []migration{
	{
		version: 1,
		name:    "create_downloads_table",
		sql: `
			CREATE TABLE downloads (
				id TEXT PRIMARY KEY,
				url TEXT NOT NULL,
				destination TEXT,
				filename TEXT,
				file_path TEXT,
				status TEXT NOT NULL DEFAULT 'pending',
				progress INTEGER DEFAULT 0,
				total INTEGER DEFAULT -1,
				speed INTEGER DEFAULT 0,
				error TEXT,
				retries INTEGER DEFAULT 0,
				max_retries INTEGER DEFAULT 3,
				created_at DATETIME NOT NULL,
				started_at DATETIME,
				completed_at DATETIME
			);

			CREATE TABLE download_share_links (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				download_id TEXT NOT NULL REFERENCES downloads(id) ON DELETE CASCADE,
				token TEXT UNIQUE NOT NULL,
				url TEXT NOT NULL,
				created_at DATETIME NOT NULL,
				expires_at DATETIME
			);

			CREATE INDEX idx_downloads_status ON downloads(status);
			CREATE INDEX idx_downloads_created_at ON downloads(created_at);
			CREATE INDEX idx_share_links_token ON download_share_links(token);
			CREATE INDEX idx_share_links_download_id ON download_share_links(download_id);
		`,
	},
	{
		version: 2,
		name:    "create_jobs_table",
		sql: `
			CREATE TABLE jobs (
				id TEXT PRIMARY KEY,
				command TEXT NOT NULL,
				cwd TEXT,
				status TEXT NOT NULL DEFAULT 'pending',
				pid INTEGER,
				exit_code INTEGER,
				log_file TEXT,
				created_at DATETIME NOT NULL,
				started_at DATETIME,
				finished_at DATETIME
			);

			CREATE INDEX idx_jobs_status ON jobs(status);
			CREATE INDEX idx_jobs_created_at ON jobs(created_at);
		`,
	},
	{
		version: 3,
		name:    "create_pinned_folders_table",
		sql: `
			CREATE TABLE pinned_folders (
				id TEXT PRIMARY KEY,
				path TEXT NOT NULL,
				name TEXT NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(path)
			);

			CREATE INDEX idx_pinned_folders_path ON pinned_folders(path);
		`,
	},
}
