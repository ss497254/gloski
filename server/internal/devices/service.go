package devices

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
type Service struct {
	store    *Store
	join     string
	timeout  time.Duration
	interval int
	version  string
	onChange func()
}

// NewService creates the fleet service.
func NewService(db *database.Database, cfg config.MasterConfig, version, dataDir string) (*Service, error) {
	join, err := resolveJoinToken(cfg, dataDir)
	if err != nil {
		return nil, err
	}
	return &Service{
		store:    NewStore(db),
		join:     join,
		timeout:  time.Duration(cfg.DeviceTimeout) * time.Second,
		interval: cfg.HeartbeatInterval,
		version:  version,
	}, nil
}

// SetOnChange registers a callback invoked after a device changes.
func (s *Service) SetOnChange(fn func()) {
	s.onChange = fn
}

func (s *Service) changed() {
	if s.onChange != nil {
		s.onChange()
	}
}

// Register enrolls an agent. Re-registering an existing name rotates its token.
func (s *Service) Register(req RegisterRequest, remoteIP string) (*RegisterResponse, error) {
	if !s.validJoinToken(req.JoinToken) {
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
		URL:       deriveURL(req, remoteIP),
		Token:     randomToken(),
		Hostname:  strings.TrimSpace(req.Hostname),
		Platform:  strings.TrimSpace(req.Platform),
		Version:   strings.TrimSpace(req.Version),
		Stats:     req.Stats,
		CreatedAt: now,
		LastSeen:  now,
	}

	if err := s.store.Upsert(dev); err != nil {
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	// Read back the canonical record (the ID may belong to an existing device).
	saved, err := s.store.Get(dev.ID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device enrolled: %s (%s) at %s", saved.Name, saved.ID, saved.URL)
	s.changed()

	return &RegisterResponse{
		DeviceID:          saved.ID,
		DeviceToken:       saved.Token,
		HeartbeatInterval: s.interval,
		MasterVersion:     s.version,
	}, nil
}

// Heartbeat authenticates a device by token and stores its latest stats.
func (s *Service) Heartbeat(token string, req HeartbeatRequest) (*Device, error) {
	dev, err := s.store.GetByToken(token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.store.UpdateHeartbeat(dev.ID, strings.TrimSpace(req.URL), req.Version, req.Stats, now); err != nil {
		return nil, err
	}

	updated, err := s.store.Get(dev.ID)
	if err != nil {
		return nil, err
	}
	s.decorate(updated)
	s.changed()
	return updated, nil
}

// List returns all devices with online/offline status applied.
func (s *Service) List() ([]Device, error) {
	list, err := s.store.List()
	if err != nil {
		return nil, err
	}
	for i := range list {
		s.decorate(&list[i])
	}
	return list, nil
}

// Get returns one device with status applied.
func (s *Service) Get(id string) (*Device, error) {
	dev, err := s.store.Get(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	s.decorate(dev)
	return dev, nil
}

// Delete removes a device from the fleet.
func (s *Service) Delete(id string) error {
	if err := s.store.Delete(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDeviceNotFound
		}
		return err
	}
	logger.Info("Device removed: %s", id)
	s.changed()
	return nil
}

// HeartbeatInterval returns the advertised interval in seconds.
func (s *Service) HeartbeatInterval() int {
	return s.interval
}

// JoinToken returns the enrollment token agents must present.
func (s *Service) JoinToken() string {
	return s.join
}

func (s *Service) decorate(dev *Device) {
	if dev.LastSeen.IsZero() || time.Since(dev.LastSeen) > s.timeout {
		dev.Status = StatusOffline
	} else {
		dev.Status = StatusOnline
	}
}

func (s *Service) validJoinToken(token string) bool {
	if s.join == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.join)) == 1
}

func deriveURL(req RegisterRequest, remoteIP string) string {
	raw := strings.TrimSpace(req.URL)
	port := req.Port
	if port <= 0 {
		port = 8080
	}

	if raw != "" {
		if u, err := url.Parse(raw); err == nil {
			if p := u.Port(); p != "" {
				if n, err := strconv.Atoi(p); err == nil {
					port = n
				}
			}
			host := u.Hostname()
			if host != "" && host != "0.0.0.0" && host != "::" {
				return strings.TrimRight(raw, "/")
			}
		}
	}

	host := remoteIP
	if host == "" {
		host = strings.TrimSpace(req.Hostname)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
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
