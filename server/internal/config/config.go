// Package config defines Gloski's configuration, defaults and environment
// overrides. The same types hold master and agent settings; each binary reads
// the fields for its side.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Operating modes accepted in Config.Mode. The gloski binary only runs
// ModeMaster; gloski-agent forces ModeAgent.
const (
	ModeMaster = "master"
	ModeAgent  = "agent"
)

// ProbeDefinition is one native probe supplied by the master. Kind is tcp,
// http, dns or tls; interval and timeout are duration strings such as "5m".
type ProbeDefinition struct {
	Kind       string `json:"kind"`
	Target     string `json:"target"`
	Accepted   []int  `json:"accepted,omitempty"`
	Interval   string `json:"interval,omitempty"`
	Timeout    string `json:"timeout,omitempty"`
	SkipVerify bool   `json:"skip_verify,omitempty"`
}

// Config is the complete configuration for a master or agent process. Values
// come from defaults, an optional JSON file, then environment variables.
type Config struct {
	// Operating mode: master (default) or agent.
	Mode string `json:"mode"`

	// Server settings
	Host            string `json:"host"`
	Port            int    `json:"port"`
	ShutdownTimeout int    `json:"shutdown_timeout"` // Shutdown timeout in seconds (default: 5)

	// Data directory
	DataDir string `json:"data_dir"` // Directory for database and logs (default: ~/.gloski/data)

	// Authentication
	APIKey string `json:"api_key"` // API key for authentication

	// JWT Authentication (asymmetric - public key only for verification)
	JWTPublicKey     string `json:"jwt_public_key"`      // PEM-encoded RSA public key
	JWTPublicKeyFile string `json:"jwt_public_key_file"` // Path to PEM file (alternative to inline key)

	// Security
	// AllowedOrigins lists extra browser origins that may call the API. Empty
	// means same-origin only; "*" disables the check.
	AllowedOrigins []string `json:"allowed_origins"`
	AllowedPaths   []string `json:"allowed_paths"` // empty = allow all

	// Probes are optional native checks run on devices. Empty means no native
	// checks are scheduled. Set with GLOSKI_PROBES as a JSON array.
	Probes []ProbeDefinition `json:"probes"`

	// Shell settings
	Shell string `json:"shell"`

	// Logging
	LogLevel string `json:"log_level"` // debug, info, warn, error

	// API settings
	DetailedErrors bool `json:"detailed_errors"` // Include detailed error messages in API responses (useful for development)

	// Master settings
	Master MasterConfig `json:"master"`

	// Device agent settings
	Agent AgentConfig `json:"agent"`

	// Tailscale LocalAPI settings
	Tailscale TailscaleConfig `json:"tailscale"`
}

// TailscaleConfig controls access to the local tailscaled API.
type TailscaleConfig struct {
	// Socket is the tailscaled Unix socket path. Auto-discovered when empty.
	Socket string `json:"socket"`
}

// MasterConfig holds settings for the control tower.
type MasterConfig struct {
	// WebDir is the directory containing the built SPA (default: web/dist)
	WebDir string `json:"web_dir"`
	// JoinToken is the one-time token agents use to enroll. If empty, one is generated.
	JoinToken string `json:"join_token"`
	// DeviceTimeout is how long without a heartbeat before a device is shown offline (seconds).
	DeviceTimeout int `json:"device_timeout"`
	// HeartbeatInterval is the interval advertised to agents (seconds).
	HeartbeatInterval int `json:"heartbeat_interval"`
	// AgentBinary is the path to gloski-agent used for SSH deployments.
	// If empty, the master looks next to its own binary.
	AgentBinary string `json:"agent_binary"`
	// SSHKeyFile is an optional private key passed to ssh for deployments.
	SSHKeyFile string `json:"ssh_key_file"`
	// DeployTimeout is the maximum time for an SSH deployment (seconds).
	DeployTimeout int `json:"deploy_timeout"`
	// LocalAgent controls whether the master also runs an agent for itself.
	LocalAgent bool `json:"local_agent"`
	// LocalAgentName overrides the display name of the master device.
	LocalAgentName string `json:"local_agent_name"`
	// AgentGRPCHost/Port is where remote agents connect to the master. An empty
	// host binds only to this machine's Tailscale addresses.
	AgentGRPCHost string `json:"agent_grpc_host"`
	AgentGRPCPort int    `json:"agent_grpc_port"`
	// AgentGRPCURL overrides the address given to deployed agents.
	AgentGRPCURL string `json:"agent_grpc_url"`
}

// AgentConfig holds settings for a device agent.
type AgentConfig struct {
	// MasterURL is the base URL of the control tower (e.g. https://gloski.tailnet.ts.net).
	MasterURL string `json:"master_url"`
	// JoinToken is the enrollment token issued by the master.
	JoinToken string `json:"join_token"`
	// Name is the display name for this device (default: hostname).
	Name string `json:"name"`
	// HeartbeatInterval is how often to report to the master (seconds).
	HeartbeatInterval int `json:"heartbeat_interval"`
	// DeployID links this agent to the master deployment that started it.
	DeployID string `json:"deploy_id"`
	// Master marks the agent running on the control tower itself.
	Master bool `json:"master"`
}

// DefaultConfig returns the built-in defaults before the config file and
// environment are applied.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".gloski", "data")

	return &Config{
		Mode:            ModeMaster,
		Host:            "127.0.0.1",
		Port:            8080,
		ShutdownTimeout: 5,
		DataDir:         dataDir,
		AllowedOrigins:  []string{},
		AllowedPaths:    []string{},
		Shell:           getDefaultShell(),
		LogLevel:        "info",
		Master: MasterConfig{
			WebDir:            "web/dist",
			DeviceTimeout:     45,
			HeartbeatInterval: 10,
			DeployTimeout:     120,
			LocalAgent:        true,
			AgentGRPCHost:     "",
			AgentGRPCPort:     9090,
		},
		Agent: AgentConfig{
			HeartbeatInterval: 10,
		},
		Tailscale: TailscaleConfig{},
	}
}

// MasterDBPath returns the path to the master's SQLite database.
func (c *Config) MasterDBPath() string {
	return filepath.Join(c.DataDir, "master.db")
}

// Load reads the optional JSON config file, applies environment overrides and
// validates the result.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// If config file exists, load it
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// File doesn't exist, use defaults
		} else if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	cfg.loadFromEnv()

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	switch c.Mode {
	case ModeMaster:
		// Master user auth is optional. Agent endpoints use per-device tokens.
		if c.Master.DeviceTimeout <= 0 {
			c.Master.DeviceTimeout = 45
		}
		if c.Master.HeartbeatInterval <= 0 {
			c.Master.HeartbeatInterval = 10
		}
		if c.Master.DeployTimeout <= 0 {
			c.Master.DeployTimeout = 120
		}
	case ModeAgent:
		// The device token is assigned during enrollment, so API auth is not
		// required in the local config.
		if c.Agent.HeartbeatInterval <= 0 {
			c.Agent.HeartbeatInterval = 10
		}
	default:
		return fmt.Errorf("invalid mode %q (expected master or agent; standalone mode was removed)", c.Mode)
	}

	// Load JWT public key from file if specified
	if c.JWTPublicKeyFile != "" && c.JWTPublicKey == "" {
		data, err := os.ReadFile(c.JWTPublicKeyFile)
		if err != nil {
			return fmt.Errorf("failed to read JWT public key file: %w", err)
		}
		c.JWTPublicKey = string(data)
	}

	return nil
}

// Addr returns the host:port the dashboard listens on.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Save writes the configuration as JSON with owner-only permissions.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// IsPathAllowed reports whether path is inside one of AllowedPaths. An empty
// list allows every path.
func (c *Config) IsPathAllowed(path string) bool {
	if len(c.AllowedPaths) == 0 {
		return true
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, allowed := range c.AllowedPaths {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		// Check if path is under allowed path or is the allowed path itself
		if absPath == allowedAbs {
			return true
		}
		// Check if absPath is a subdirectory of allowedAbs
		rel, err := filepath.Rel(allowedAbs, absPath)
		if err != nil {
			continue
		}
		// If relative path starts with "..", it's outside the allowed path
		if strings.HasPrefix(rel, "..") {
			continue
		}
		return true
	}

	return false
}

func getDefaultShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/bash"
}
