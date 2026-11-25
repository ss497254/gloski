package database

import (
	"database/sql"
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

// LatestVersion is the highest migration the master knows about.
func LatestVersion() int {
	if len(migrations) == 0 {
		return 0
	}
	return migrations[len(migrations)-1].version
}

type migration struct {
	version int
	name    string
	sql     string
	apply   func(*sql.Tx) error
}

func (d *Database) runMigration(m migration) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if m.apply != nil {
		if err := m.apply(tx); err != nil {
			return fmt.Errorf("failed to apply migration: %w", err)
		}
	} else if _, err := tx.Exec(m.sql); err != nil {
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
	{
		version: 4,
		name:    "create_devices_table",
		sql: `
			CREATE TABLE devices (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL UNIQUE,
				url TEXT NOT NULL,
				token TEXT NOT NULL UNIQUE,
				hostname TEXT,
				platform TEXT,
				version TEXT,
				stats_json TEXT,
				created_at DATETIME NOT NULL,
				last_seen DATETIME,
				updated_at DATETIME NOT NULL
			);

			CREATE INDEX idx_devices_last_seen ON devices(last_seen);
			CREATE INDEX idx_devices_token ON devices(token);
		`,
	},
	{
		version: 5,
		name:    "add_device_deployments",
		sql: `
			ALTER TABLE devices ADD COLUMN deploy_id TEXT;
			CREATE INDEX idx_devices_deploy_id ON devices(deploy_id);

			CREATE TABLE deployments (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				host TEXT NOT NULL,
				ssh_user TEXT,
				ssh_port INTEGER NOT NULL,
				agent_port INTEGER NOT NULL,
				master_url TEXT NOT NULL,
				status TEXT NOT NULL,
				remote_pid TEXT,
				device_id TEXT,
				error TEXT,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			);

			CREATE INDEX idx_deployments_status ON deployments(status);
			CREATE INDEX idx_deployments_device_id ON deployments(device_id);
		`,
	},
	{
		version: 7,
		name:    "mark_master_device",
		sql: `
			ALTER TABLE devices ADD COLUMN master INTEGER NOT NULL DEFAULT 0;
			CREATE INDEX idx_devices_master ON devices(master);
		`,
	},
	{
		version: 8,
		name:    "add_device_machine_id",
		sql: `
			ALTER TABLE devices ADD COLUMN machine_id TEXT;
		`,
	},
	{
		version: 9,
		name:    "add_check_results",
		sql: `
			CREATE TABLE check_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				check_id TEXT NOT NULL,
				device_id TEXT NOT NULL,
				status TEXT NOT NULL,
				summary TEXT NOT NULL,
				details_json TEXT NOT NULL DEFAULT '[]',
				ran_at DATETIME NOT NULL,
				duration_ms INTEGER NOT NULL DEFAULT 0,
				interval_ns INTEGER NOT NULL DEFAULT 0
			);
			CREATE INDEX idx_check_results_latest ON check_results(check_id, device_id, ran_at DESC, id DESC);
		`,
	},
	{
		version: 10,
		name:    "add_device_actions",
		sql: `
			CREATE TABLE device_actions (
				id TEXT PRIMARY KEY,
				device_id TEXT NOT NULL,
				kind TEXT NOT NULL,
				target TEXT NOT NULL DEFAULT '',
				from_value TEXT NOT NULL DEFAULT '',
				to_value TEXT NOT NULL DEFAULT '',
				result TEXT NOT NULL,
				error TEXT NOT NULL DEFAULT '',
				undoable INTEGER NOT NULL DEFAULT 0,
				undone_at DATETIME,
				created_at DATETIME NOT NULL
			);
			CREATE INDEX idx_device_actions_recent ON device_actions(device_id, created_at DESC);
		`,
	},
	{
		version: 11,
		name:    "add_metric_samples",
		sql: `
			CREATE TABLE metric_samples (
				device_id TEXT NOT NULL,
				metric TEXT NOT NULL,
				ts DATETIME NOT NULL,
				day TEXT NOT NULL,
				value REAL NOT NULL
			);
			CREATE INDEX idx_metric_samples_series ON metric_samples(device_id, metric, ts DESC);
			-- The sample throttle looks up the newest ts for a device on every
			-- heartbeat, without knowing the metric yet.
			CREATE INDEX idx_metric_samples_latest ON metric_samples(device_id, ts DESC);
		`,
	},
	{
		version: 12,
		name:    "create_agent_jobs",
		apply:   migrateAgentJobs,
	},
	{
		version: 13,
		name:    "drop_legacy_standalone_tables",
		sql: `
			DROP TABLE IF EXISTS download_share_links;
			DROP TABLE IF EXISTS downloads;
			DROP TABLE IF EXISTS jobs;
			DROP TABLE IF EXISTS pinned_folders;
		`,
	},
}

// migrateAgentJobs creates job storage and upgrades databases where the tables
// were previously created by the job store at startup.
func migrateAgentJobs(tx *sql.Tx) error {
	if _, err := tx.Exec(`
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
			finished_at DATETIME,
			last_seq INTEGER NOT NULL DEFAULT 0,
			log_bytes INTEGER NOT NULL DEFAULT 0,
			log_truncated INTEGER NOT NULL DEFAULT 0
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
	`); err != nil {
		return err
	}

	for _, column := range []struct{ name, definition string }{
		{"last_seq", "INTEGER NOT NULL DEFAULT 0"},
		{"log_bytes", "INTEGER NOT NULL DEFAULT 0"},
		{"log_truncated", "INTEGER NOT NULL DEFAULT 0"},
	} {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('agent_jobs') WHERE name = ?`, column.name).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			continue
		}
		if _, err := tx.Exec(`ALTER TABLE agent_jobs ADD COLUMN ` + column.name + ` ` + column.definition); err != nil {
			return err
		}
		if column.name == "log_bytes" {
			if _, err := tx.Exec(`
				UPDATE agent_jobs SET log_bytes = (
					SELECT COALESCE(SUM(LENGTH(chunk)), 0)
					FROM agent_job_logs WHERE job_id = agent_jobs.id)`); err != nil {
				return err
			}
		}
	}
	return nil
}
