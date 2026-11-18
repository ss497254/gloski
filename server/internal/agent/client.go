// Package agent implements the device-side service that enrolls with a master
// and exposes the private Gloski API over the tailnet.
package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/devices"
)

// Client talks to the master control plane.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a master API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// Register enrolls this agent with the master.
func (c *Client) Register(req devices.RegisterRequest) (*devices.RegisterResponse, error) {
	var resp devices.RegisterResponse
	if err := c.post("/api/agents/register", "", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Heartbeat reports liveness and stats to the master.
func (c *Client) Heartbeat(token string, req devices.HeartbeatRequest) error {
	var resp struct {
		DeviceID string `json:"device_id"`
	}
	return c.post("/api/agents/heartbeat", token, req, &resp)
}

func (c *Client) post(path, token string, body, out interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("master request failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}

	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   string          `json:"error"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("master returned an invalid response (HTTP %d)", resp.StatusCode)
	}
	if !envelope.Success {
		if envelope.Error != "" {
			return fmt.Errorf("master: %s", envelope.Error)
		}
		return fmt.Errorf("master returned HTTP %d", resp.StatusCode)
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return fmt.Errorf("failed to decode master response: %w", err)
		}
	}
	return nil
}
