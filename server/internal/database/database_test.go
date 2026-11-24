package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// The DSN must actually apply its pragmas on every pooled connection.
func TestOpenAppliesPragmas(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dir with space", "master.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var journal string
	var busy, foreignKeys int
	if err := db.DB().QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if err := db.DB().QueryRow("PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatal(err)
	}
	if err := db.DB().QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if journal != "wal" || busy != 5000 || foreignKeys != 1 {
		t.Fatalf("journal_mode=%s busy_timeout=%d foreign_keys=%d", journal, busy, foreignKeys)
	}
}

func TestAgentJobsMigrationUpgradesLegacyTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	if _, err := raw.Exec(`
		CREATE TABLE agent_jobs (
			id TEXT PRIMARY KEY, device_id TEXT NOT NULL, command TEXT NOT NULL,
			cwd TEXT, status TEXT NOT NULL, exit_code INTEGER DEFAULT 0,
			error TEXT, created_at DATETIME NOT NULL, started_at DATETIME,
			finished_at DATETIME
		);
		CREATE TABLE agent_job_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT, job_id TEXT NOT NULL,
			chunk BLOB NOT NULL, stderr INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		);
		INSERT INTO agent_jobs (id, device_id, command, status, created_at)
			VALUES ('job', 'device', 'true', 'finished', CURRENT_TIMESTAMP);
		INSERT INTO agent_job_logs (job_id, chunk, stderr, created_at)
			VALUES ('job', X'616263', 0, CURRENT_TIMESTAMP);
	`); err != nil {
		t.Fatal(err)
	}

	tx, err := raw.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateAgentJobs(tx); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	var bytes int
	if err := raw.QueryRow(`SELECT log_bytes FROM agent_jobs WHERE id = 'job'`).Scan(&bytes); err != nil {
		t.Fatal(err)
	}
	if bytes != 3 {
		t.Errorf("migrated log_bytes = %d, want 3", bytes)
	}
	for _, column := range []string{"last_seq", "log_bytes", "log_truncated"} {
		var found int
		if err := raw.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('agent_jobs') WHERE name = ?`, column).Scan(&found); err != nil {
			t.Fatal(err)
		}
		if found != 1 {
			t.Errorf("migration did not add %s", column)
		}
	}
}
