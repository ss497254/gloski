// Package devices manages the master's fleet of enrolled agents.
package devices

import (
	"time"

	"github.com/ss497254/gloski/internal/system"
)

// Status values reported for a device.
const (
	StatusOnline  = "online"
	StatusOffline = "offline"
)

// Device is a machine enrolled with the control tower.
type Device struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	URL       string        `json:"url"`
	Hostname  string        `json:"hostname"`
	Platform  string        `json:"platform"`
	Version   string        `json:"version"`
	Status    string        `json:"status"`
	Stats     *system.Stats `json:"stats,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	LastSeen  time.Time     `json:"last_seen"`
	Token     string        `json:"-"`

	// DeployID links the device to a master-managed SSH deployment.
	DeployID string      `json:"deploy_id,omitempty"`
	Managed  bool        `json:"managed"`
	Master   bool        `json:"master"`
	Deploy   *Deployment `json:"deployment,omitempty"`
}

// RegisterRequest is sent by an agent the first time it starts.
type RegisterRequest struct {
	JoinToken string        `json:"join_token"`
	Name      string        `json:"name"`
	URL       string        `json:"url"`
	Hostname  string        `json:"hostname"`
	Platform  string        `json:"platform"`
	Version   string        `json:"version"`
	DeployID  string        `json:"deploy_id,omitempty"`
	Master    bool          `json:"master"`
	Stats     *system.Stats `json:"stats,omitempty"`
}

// RegisterResponse is returned to a newly enrolled agent.
type RegisterResponse struct {
	DeviceID          string `json:"device_id"`
	DeviceToken       string `json:"device_token"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
	MasterVersion     string `json:"master_version,omitempty"`
}

// HeartbeatRequest is sent periodically by an agent.
type HeartbeatRequest struct {
	Stats *system.Stats `json:"stats,omitempty"`
}

// Event is broadcast to fleet WebSocket subscribers.
type Event struct {
	Type     string   `json:"type"`
	Device   *Device  `json:"device,omitempty"`
	DeviceID string   `json:"device_id,omitempty"`
	Devices  []Device `json:"devices,omitempty"`
}
