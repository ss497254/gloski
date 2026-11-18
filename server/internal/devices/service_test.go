package devices

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/system"
)

func newTestService(t *testing.T) (*Service, *database.Database) {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "master.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	svc, err := NewService(db, config.MasterConfig{
		JoinToken:         "test-join",
		DeviceTimeout:     45,
		HeartbeatInterval: 10,
	}, "test", t.TempDir())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc, db
}

func TestRegisterHeartbeatAndDelete(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Register(RegisterRequest{JoinToken: "wrong", Name: "alpha"}, "100.64.0.5"); !errors.Is(err, ErrInvalidJoinToken) {
		t.Fatalf("expected invalid join token, got %v", err)
	}

	resp, err := svc.Register(RegisterRequest{
		JoinToken: "test-join",
		Name:      "alpha",
		Port:      9000,
		Hostname:  "alpha-host",
		Platform:  "linux/amd64",
	}, "100.64.0.5")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if resp.DeviceID == "" || resp.DeviceToken == "" {
		t.Fatal("registration response is missing credentials")
	}
	if resp.HeartbeatInterval != 10 {
		t.Fatalf("heartbeat interval = %d, want 10", resp.HeartbeatInterval)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("device count = %d, want 1", len(list))
	}
	if list[0].URL != "http://100.64.0.5:9000" {
		t.Fatalf("device URL = %q", list[0].URL)
	}
	if list[0].Status != StatusOnline {
		t.Fatalf("device status = %q, want online", list[0].Status)
	}

	if _, err := svc.Heartbeat("bad-token", HeartbeatRequest{}); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("expected unknown device, got %v", err)
	}

	updated, err := svc.Heartbeat(resp.DeviceToken, HeartbeatRequest{
		Stats: &system.Stats{Hostname: "alpha-host"},
	})
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if updated.Stats == nil || updated.Stats.Hostname != "alpha-host" {
		t.Fatalf("cached stats were not updated: %#v", updated.Stats)
	}

	// Re-enrolling the same name keeps the ID but rotates the token.
	rotated, err := svc.Register(RegisterRequest{
		JoinToken: "test-join",
		Name:      "alpha",
		URL:       "http://alpha.example:9100",
	}, "100.64.0.5")
	if err != nil {
		t.Fatalf("re-register: %v", err)
	}
	if rotated.DeviceID != resp.DeviceID {
		t.Fatalf("device ID changed: %q -> %q", resp.DeviceID, rotated.DeviceID)
	}
	if rotated.DeviceToken == resp.DeviceToken {
		t.Fatal("device token was not rotated")
	}

	if err := svc.Delete(rotated.DeviceID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(rotated.DeviceID); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("expected device to be gone, got %v", err)
	}
}
