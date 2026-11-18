package database

import (
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
