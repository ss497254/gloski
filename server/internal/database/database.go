// Package database provides SQLite database operations for Gloski.
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Database wraps the SQL database connection.
type Database struct {
	db   *sql.DB
	path string
}

// Open opens or creates a SQLite database at the specified path.
func Open(path string) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// modernc.org/sqlite takes pragmas as _pragma=name(value); the mattn-style
	// _journal_mode/_busy_timeout parameters are silently ignored. WAL lets
	// readers run alongside the single writer, and the busy timeout makes a
	// second writer wait instead of failing with "database is locked".
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)&_txlock=immediate", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings for SQLite with WAL mode
	// WAL mode allows concurrent reads with a single writer
	// We allow multiple connections for reads, SQLite will serialize writes
	db.SetMaxOpenConns(10) // Allow concurrent reads
	db.SetMaxIdleConns(5)  // Keep some connections ready

	d := &Database{
		db:   db,
		path: path,
	}

	// Run migrations
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return d, nil
}

// DB returns the underlying sql.DB instance.
func (d *Database) DB() *sql.DB {
	return d.db
}

// Close closes the database connection.
func (d *Database) Close() error {
	return d.db.Close()
}

// Path returns the database file path.
func (d *Database) Path() string {
	return d.path
}
