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

	// Open database with WAL mode for better concurrency
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite doesn't handle multiple writers well
	db.SetMaxIdleConns(1)

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
