package devices

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ss497254/gloski/internal/database"
)

// Deployment status values.
const (
	DeployPending   = "pending"
	DeployUploading = "uploading"
	DeployStarting  = "starting"
	DeployRunning   = "running"
	DeployStopped   = "stopped"
	DeployFailed    = "failed"
)

// Deployment records how a device was provisioned over SSH. Tokens are never
// persisted here.
type Deployment struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	User      string    `json:"user"`
	SSHPort   int       `json:"ssh_port"`
	AgentPort int       `json:"agent_port"`
	MasterURL string    `json:"master_url"`
	Status    string    `json:"status"`
	RemotePID string    `json:"remote_pid,omitempty"`
	DeviceID  string    `json:"device_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeploymentStore persists SSH deployment records.
type DeploymentStore struct {
	db *sql.DB
}

// NewDeploymentStore creates a deployment store.
func NewDeploymentStore(db *database.Database) *DeploymentStore {
	return &DeploymentStore{db: db.DB()}
}

// Create inserts a new deployment.
func (s *DeploymentStore) Create(deployment *Deployment) error {
	now := time.Now().UTC()
	if deployment.CreatedAt.IsZero() {
		deployment.CreatedAt = now
	}
	if deployment.UpdatedAt.IsZero() {
		deployment.UpdatedAt = now
	}
	_, err := s.db.Exec(`
		INSERT INTO deployments
			(id, name, host, ssh_user, ssh_port, agent_port, master_url, status,
			 remote_pid, device_id, error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		deployment.ID, deployment.Name, deployment.Host, deployment.User,
		deployment.SSHPort, deployment.AgentPort, deployment.MasterURL,
		deployment.Status, deployment.RemotePID, deployment.DeviceID,
		deployment.Error, deployment.CreatedAt, deployment.UpdatedAt,
	)
	return err
}

// Get returns a deployment by ID.
func (s *DeploymentStore) Get(id string) (*Deployment, error) {
	row := s.db.QueryRow(`
		SELECT id, name, host, COALESCE(ssh_user, ''), ssh_port, agent_port,
		       master_url, status, COALESCE(remote_pid, ''), COALESCE(device_id, ''),
		       COALESCE(error, ''), created_at, updated_at
		FROM deployments WHERE id = ?`, id)
	return scanDeployment(row)
}

// List returns deployments, newest first.
func (s *DeploymentStore) List() ([]Deployment, error) {
	rows, err := s.db.Query(`
		SELECT id, name, host, COALESCE(ssh_user, ''), ssh_port, agent_port,
		       master_url, status, COALESCE(remote_pid, ''), COALESCE(device_id, ''),
		       COALESCE(error, ''), created_at, updated_at
		FROM deployments ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Deployment
	for rows.Next() {
		deployment, err := scanDeployment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *deployment)
	}
	return result, rows.Err()
}

// SetStatus updates a deployment's status and optional error message.
func (s *DeploymentStore) SetStatus(id, status, errMsg string) error {
	result, err := s.db.Exec(`
		UPDATE deployments SET status = ?, error = ?, updated_at = ? WHERE id = ?`,
		status, nullable(errMsg), time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	return rowsAffected(result, id)
}

// SetRunning marks a deployment as running with its remote PID.
func (s *DeploymentStore) SetRunning(id, pid string) error {
	result, err := s.db.Exec(`
		UPDATE deployments
		SET status = ?, remote_pid = ?, error = NULL, updated_at = ?
		WHERE id = ?`, DeployRunning, nullable(pid), time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	return rowsAffected(result, id)
}

// SetStopped marks a deployment as stopped.
func (s *DeploymentStore) SetStopped(id string) error {
	result, err := s.db.Exec(`
		UPDATE deployments
		SET status = ?, remote_pid = NULL, error = NULL, updated_at = ?
		WHERE id = ?`, DeployStopped, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	return rowsAffected(result, id)
}

// LinkDevice associates the deployment with the device that enrolled.
func (s *DeploymentStore) LinkDevice(id, deviceID string) error {
	result, err := s.db.Exec(`
		UPDATE deployments SET device_id = ?, updated_at = ? WHERE id = ?`,
		deviceID, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	return rowsAffected(result, id)
}

// Delete removes a deployment record.
func (s *DeploymentStore) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM deployments WHERE id = ?", id)
	if err != nil {
		return err
	}
	return rowsAffected(result, id)
}

type deploymentScanner interface {
	Scan(dest ...interface{}) error
}

func scanDeployment(row deploymentScanner) (*Deployment, error) {
	var deployment Deployment
	err := row.Scan(
		&deployment.ID, &deployment.Name, &deployment.Host, &deployment.User,
		&deployment.SSHPort, &deployment.AgentPort, &deployment.MasterURL,
		&deployment.Status, &deployment.RemotePID, &deployment.DeviceID,
		&deployment.Error, &deployment.CreatedAt, &deployment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

func nullable(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func rowsAffected(result sql.Result, id string) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%w: %s", ErrDeploymentNotFound, id)
	}
	return nil
}

// ErrDeploymentNotFound is returned when a deployment ID does not exist.
var ErrDeploymentNotFound = errors.New("deployment not found")
