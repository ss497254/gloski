package database

import (
	"database/sql"
	"time"
)

// NullString converts a string to sql.NullString.
// Empty strings are stored as NULL.
func NullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// NullInt converts an int to sql.NullInt64.
// The valid flag controls whether the value is NULL.
func NullInt(i int, valid bool) sql.NullInt64 {
	if !valid {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}

// NullTime converts a *time.Time to sql.NullTime.
// Nil pointers are stored as NULL.
func NullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
