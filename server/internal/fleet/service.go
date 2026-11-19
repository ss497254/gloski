package fleet

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/logger"
)

var (
	ErrInvalidJoinToken = errors.New("invalid join token")
	ErrDeviceNotFound   = errors.New("device not found")
)

// Service implements registration, heartbeats, and fleet queries.
type Registry struct {
	store       *Store
	deployments *DeploymentStore
	join        string
	timeout     time.Duration
	interval    int
	version     string
	onChange    func()
	online      func(deviceID string) bool
}

// NewService creates the fleet service.
func NewRegistry(db *database.Database, cfg config.MasterConfig, version, dataDir string) (*Registry, error) {
	join, err := resolveJoinToken(cfg, dataDir)
	if err != nil {
		return nil, err
	}
	return &Registry{
		store:       NewStore(db),
		deployments: NewDeploymentStore(db),
		join:        join,
		timeout:     time.Duration(cfg.DeviceTimeout) * time.Second,
		interval:    cfg.HeartbeatInterval,
		version:     version,
	}, nil
}

// SetOnChange registers a callback invoked after a device changes.
func (r *Registry) SetOnChange(fn func()) {
	r.onChange = fn
}

// SetOnline supplies the live connection check. Once set, a device is online
// exactly while its agent stream is connected; without it, status falls back
// to how recently the device was seen.
func (r *Registry) SetOnline(fn func(deviceID string) bool) {
	r.online = fn
}

// Changed tells subscribers that device state changed outside the registry,
// such as an agent connecting or disconnecting.
func (r *Registry) Changed() {
	r.changed()
}

func (r *Registry) changed() {
	if r.onChange != nil {
		r.onChange()
	}
}

// Register enrolls an agent. Re-registering an existing name rotates its token.
func (r *Registry) Register(req RegisterRequest) (*RegisterResponse, error) {
	if !r.validJoinToken(req.JoinToken) {
		return nil, ErrInvalidJoinToken
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(req.Hostname)
	}
	if name == "" {
		name = "unnamed-device"
	}

	now := time.Now().UTC()
	dev := &Device{
		ID:        uuid.NewString(),
		Name:      name,
		URL:       deriveURL(req.URL),
		Token:     randomToken(),
		Hostname:  strings.TrimSpace(req.Hostname),
		Platform:  strings.TrimSpace(req.Platform),
		Version:   strings.TrimSpace(req.Version),
		DeployID:  strings.TrimSpace(req.DeployID),
		Master:    req.Master,
		MachineID: strings.TrimSpace(req.MachineID),
		Stats:     req.Stats,
		CreatedAt: now,
		LastSeen:  now,
	}

	if err := r.store.Upsert(dev); err != nil {
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	// Read back the canonical record (the ID may belong to an existing device).
	saved, err := r.store.Get(dev.ID)
	if err != nil {
		return nil, err
	}

	if saved.DeployID != "" {
		if err := r.deployments.LinkDevice(saved.DeployID, saved.ID); err != nil {
			logger.Warn("Failed to link deployment %s to device %s: %v", saved.DeployID, saved.ID, err)
		}
	}

	logger.Info("Device enrolled: %s (%s) at %s", saved.Name, saved.ID, saved.URL)
	r.changed()

	return &RegisterResponse{
		DeviceID:          saved.ID,
		DeviceToken:       saved.Token,
		HeartbeatInterval: r.interval,
		MasterVersion:     r.version,
	}, nil
}

// Heartbeat authenticates a device by token and stores its latest stats.
func (r *Registry) Heartbeat(token string, req HeartbeatRequest) (*Device, error) {
	dev, err := r.store.GetByToken(token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	now := time.Now().UTC()
	if err := r.store.UpdateHeartbeat(dev.ID, req.Stats, now); err != nil {
		return nil, err
	}

	updated, err := r.store.Get(dev.ID)
	if err != nil {
		return nil, err
	}
	r.decorate(updated)
	r.changed()
	return updated, nil
}

// List returns all devices with online/offline status applied.
func (r *Registry) List() ([]Device, error) {
	list, err := r.store.List()
	if err != nil {
		return nil, err
	}
	for i := range list {
		r.decorate(&list[i])
	}
	return list, nil
}

// Get returns one device with status applied.
func (r *Registry) Get(id string) (*Device, error) {
	dev, err := r.store.Get(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	r.decorate(dev)
	return dev, nil
}

// Delete removes a device from the fleet.
func (r *Registry) Delete(id string) error {
	if err := r.store.Delete(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDeviceNotFound
		}
		return err
	}
	logger.Info("Device removed: %s", id)
	r.changed()
	return nil
}

// CreateDeployment records a new SSH deployment.
func (r *Registry) CreateDeployment(deployment *Deployment) error {
	return r.deployments.Create(deployment)
}

// GetDeployment returns a deployment by ID.
func (r *Registry) GetDeployment(id string) (*Deployment, error) {
	return r.deployments.Get(id)
}

// ListDeployments returns all deployments, newest first.
func (r *Registry) ListDeployments() ([]Deployment, error) {
	return r.deployments.List()
}

// SetDeploymentStatus updates a deployment's status and optional error.
func (r *Registry) SetDeploymentStatus(id, status, errMsg string) error {
	err := r.deployments.SetStatus(id, status, errMsg)
	if err == nil {
		r.changed()
	}
	return err
}

// SetDeploymentRunning marks a deployment as running with its remote PID.
func (r *Registry) SetDeploymentRunning(id, pid string) error {
	err := r.deployments.SetRunning(id, pid)
	if err == nil {
		r.changed()
	}
	return err
}

// SetDeploymentStopped marks a deployment as stopped.
func (r *Registry) SetDeploymentStopped(id string) error {
	err := r.deployments.SetStopped(id)
	if err == nil {
		r.changed()
	}
	return err
}

// DeleteDeployment removes a deployment record.
func (r *Registry) DeleteDeployment(id string) error {
	return r.deployments.Delete(id)
}

// HeartbeatInterval returns the advertised interval in seconds.
func (r *Registry) HeartbeatInterval() int {
	return r.interval
}

// JoinToken returns the enrollment token agents must present.
func (r *Registry) JoinToken() string {
	return r.join
}

func (r *Registry) decorate(dev *Device) {
	switch {
	case r.online != nil && r.online(dev.ID):
		dev.Status = StatusOnline
	case r.online != nil:
		dev.Status = StatusOffline
	case dev.LastSeen.IsZero() || time.Since(dev.LastSeen) > r.timeout:
		dev.Status = StatusOffline
	default:
		dev.Status = StatusOnline
	}
	if dev.DeployID != "" {
		if deployment, err := r.deployments.Get(dev.DeployID); err == nil {
			dev.Managed = true
			dev.Deploy = deployment
		}
	}
}

func (r *Registry) validJoinToken(token string) bool {
	if r.join == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(r.join)) == 1
}

// deriveURL keeps the agent's advertised label. Agents no longer expose HTTP.
func deriveURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "grpc://agent"
	}
	return strings.TrimRight(raw, "/")
}

func randomToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failure is unrecoverable for token generation.
		panic("failed to generate device token: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func resolveJoinToken(cfg config.MasterConfig, dataDir string) (string, error) {
	if cfg.JoinToken != "" {
		return cfg.JoinToken, nil
	}

	path := filepath.Join(dataDir, "master-join-token")
	data, err := os.ReadFile(path)
	if err == nil {
		token := strings.TrimSpace(string(data))
		if token != "" {
			return token, nil
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read join token: %w", err)
	}

	token := randomToken()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0600); err != nil {
		return "", err
	}
	logger.Info("Generated master join token at %s", path)
	return token, nil
}

// Identity returns the ID and machine ID of an enrolled device by name. The
// boolean is false when no device has that name.
func (r *Registry) Identity(name string) (id, machineID string, found bool) {
	id, machineID, err := r.store.Identity(strings.TrimSpace(name))
	if err != nil {
		return "", "", false
	}
	return id, machineID, true
}
