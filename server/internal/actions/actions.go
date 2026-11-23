// Package actions records what the master changed on a device. Some changes
// can be undone because the master knew the previous wanted state.
package actions

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/database"
)

// Kinds recorded today. File operations and services will add their own.
const (
	KindAgentStart   = "agent_start"
	KindAgentStop    = "agent_stop"
	KindAgentRestart = "agent_restart"
	KindTrustHost    = "trust_host"
	KindUndo         = "undo"
)

// Results recorded in Action.Result.
const (
	ResultOK     = "ok"
	ResultFailed = "failed"
)

// Action is one master-owned change to a device.
type Action struct {
	ID        string     `json:"id"`
	DeviceID  string     `json:"device_id"`
	Kind      string     `json:"kind"`
	Target    string     `json:"target"`
	From      string     `json:"from,omitempty"`
	To        string     `json:"to,omitempty"`
	Result    string     `json:"result"`
	Error     string     `json:"error,omitempty"`
	Undoable  bool       `json:"undoable"`
	UndoneAt  *time.Time `json:"undone_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Store persists device actions in the master database.
type Store struct {
	db *sql.DB
}

// NewStore creates a device action store backed by the master database.
func NewStore(db *database.Database) *Store {
	return &Store{db: db.DB()}
}

// Record appends one action. It generates the ID and timestamp so callers
// cannot forget them.
func (s *Store) Record(ctx context.Context, action Action) error {
	if action.ID == "" {
		action.ID = uuid.NewString()
	}
	if action.CreatedAt.IsZero() {
		action.CreatedAt = time.Now().UTC()
	}
	undoable := 0
	if action.Undoable {
		undoable = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO device_actions
			(id, device_id, kind, target, from_value, to_value, result, error, undoable, undone_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		action.ID, action.DeviceID, action.Kind, action.Target, action.From, action.To,
		action.Result, action.Error, undoable, action.UndoneAt, action.CreatedAt,
	)
	return err
}

// Recent returns newest actions first.
func (s *Store) Recent(ctx context.Context, deviceID string, limit int) ([]Action, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, device_id, kind, target, from_value, to_value, result, error,
		       undoable, undone_at, created_at
		FROM device_actions WHERE device_id = ?
		ORDER BY created_at DESC, id DESC LIMIT ?`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := []Action{}
	for rows.Next() {
		action, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

// Get returns one action by ID.
func (s *Store) Get(ctx context.Context, id string) (Action, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, device_id, kind, target, from_value, to_value, result, error,
		       undoable, undone_at, created_at
		FROM device_actions WHERE id = ?`, id)
	return scanAction(row)
}

// ClaimUndo atomically marks an action as being undone. Only one caller can
// win the claim, so two concurrent undo requests cannot both run.
func (s *Store) ClaimUndo(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE device_actions SET undone_at = ? WHERE id = ? AND undone_at IS NULL`, time.Now().UTC(), id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

// ReleaseUndo clears a claim after a failed undo, so the change can be tried
// again.
func (s *Store) ReleaseUndo(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE device_actions SET undone_at = NULL WHERE id = ?`, id)
	return err
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanAction(row scanner) (Action, error) {
	var action Action
	var undoable int
	err := row.Scan(
		&action.ID, &action.DeviceID, &action.Kind, &action.Target,
		&action.From, &action.To, &action.Result, &action.Error,
		&undoable, &action.UndoneAt, &action.CreatedAt,
	)
	if err != nil {
		return Action{}, err
	}
	action.Undoable = undoable != 0
	return action, nil
}
