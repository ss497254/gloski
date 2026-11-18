package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Credentials are persisted locally after enrollment.
type Credentials struct {
	DeviceID          string `json:"device_id"`
	DeviceToken       string `json:"device_token"`
	MasterURL         string `json:"master_url"`
	Name              string `json:"name"`
	HeartbeatInterval int    `json:"heartbeat_interval,omitempty"`
}

func loadCredentials(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse agent state: %w", err)
	}
	if creds.DeviceID == "" || creds.DeviceToken == "" {
		return nil, nil
	}
	return &creds, nil
}

func saveCredentials(path string, creds *Credentials) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
