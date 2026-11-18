package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ModeStandalone = "standalone"
	ModeMaster     = "master"
	ModeAgent      = "agent"
)

type Config struct {
	// Operating mode: standalone (default), master, or agent
	Mode string `json:"mode"`

	// Server settings
	Host            string `json:"host"`
	Port            int    `json:"port"`
	BaseURL         string `json:"base_url"`         // Public URL for share links (e.g., https://example.com)
	APIPrefix       string `json:"api_prefix"`       // API prefix for all routes (e.g., /my-prefix)
	ShutdownTimeout int    `json:"shutdown_timeout"` // Shutdown timeout in seconds (default: 5)

	// Data directory
	DataDir string `json:"data_dir"` // Directory for database and logs (default: ~/.gloski/data)

	// Authentication
	APIKey string `json:"api_key"` // API key for authentication

	// JWT Authentication (asymmetric - public key only for verification)
	JWTPublicKey     string `json:"jwt_public_key"`      // PEM-encoded RSA public key
	JWTPublicKeyFile string `json:"jwt_public_key_file"` // Path to PEM file (alternative to inline key)

	// Security
	AllowedOrigins []string `json:"allowed_origins"`
	AllowedPaths   []string `json:"allowed_paths"` // empty = allow all

	// Shell settings
	Shell string `json:"shell"`

	// Logging
	LogLevel string `json:"log_level"` // debug, info, warn, error

	// API settings
	DetailedErrors  bool  `json:"detailed_errors"`    // Include detailed error messages in API responses (useful for development)
	MaxJSONBodySize int64 `json:"max_json_body_size"` // Maximum size for JSON request bodies in bytes (default: 1MB)

	// Downloads
	Downloads DownloadsConfig `json:"downloads"`

	// Jobs
	Jobs JobsConfig `json:"jobs"`

	// Control tower (master) settings
	Master MasterConfig `json:"master"`

	// Device agent settings
	Agent AgentConfig `json:"agent"`
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
}

// AgentConfig holds settings for a device agent.
type AgentConfig struct {
	// MasterURL is the base URL of the control tower (e.g. https://gloski.tailnet.ts.net).
	MasterURL string `json:"master_url"`
	// JoinToken is the enrollment token issued by the master.
	JoinToken string `json:"join_token"`
	// Name is the display name for this device (default: hostname).
	Name string `json:"name"`
	// AdvertiseURL is the URL the master should use to reach this agent.
	// If empty, the master derives it from the connection and the listen port.
	AdvertiseURL string `json:"advertise_url"`
	// HeartbeatInterval is how often to report to the master (seconds).
	HeartbeatInterval int `json:"heartbeat_interval"`
	// StateFile stores the device credentials (default: <data_dir>/agent.json).
	StateFile string `json:"state_file"`
}

// DownloadsConfig holds configuration for the download manager
type DownloadsConfig struct {
	Enabled       bool `json:"enabled"`
	MaxConcurrent int  `json:"max_concurrent"` // Maximum concurrent downloads (default: 3)
	MaxRetries    int  `json:"max_retries"`    // Maximum retry attempts (default: 3)
}

// JobsConfig holds configuration for the jobs manager
type JobsConfig struct {
	Enabled bool `json:"enabled"`
	MaxJobs int  `json:"max_jobs"` // Maximum number of jobs to keep (default: 100)
}

func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".gloski", "data")

	return &Config{
		Mode:            ModeStandalone,
		Host:            "127.0.0.1",
		Port:            8080,
		ShutdownTimeout: 5,
		DataDir:         dataDir,
		AllowedOrigins:  []string{"*"},
		AllowedPaths:    []string{},
		Shell:           getDefaultShell(),
		LogLevel:        "info",
		Downloads: DownloadsConfig{
			Enabled:       true,
			MaxConcurrent: 3,
			MaxRetries:    3,
		},
		Jobs: JobsConfig{
			Enabled: true,
			MaxJobs: 100,
		},
		Master: MasterConfig{
			WebDir:            "web/dist",
			DeviceTimeout:     45,
			HeartbeatInterval: 10,
		},
		Agent: AgentConfig{
			HeartbeatInterval: 10,
		},
	}
}

// MasterDBPath returns the path to the master's SQLite database.
func (c *Config) MasterDBPath() string {
	return filepath.Join(c.DataDir, "master.db")
}

// AgentStatePath returns the path to the agent credential file.
func (c *Config) AgentStatePath() string {
	if c.Agent.StateFile != "" {
		return c.Agent.StateFile
	}
	return filepath.Join(c.DataDir, "agent.json")
}

// DatabasePath returns the path to the SQLite database
func (c *Config) DatabasePath() string {
	return filepath.Join(c.DataDir, "gloski.db")
}

// LogsDir returns the path to the logs directory
func (c *Config) LogsDir() string {
	return filepath.Join(c.DataDir, "logs")
}

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
			return nil, fmt.Errorf("Failed to parse config file: %w\n", err)
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

func (c *Config) loadFromEnv() {
	if v := os.Getenv("GLOSKI_MODE"); v != "" {
		c.Mode = v
	}
	if v := os.Getenv("GLOSKI_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("GLOSKI_PORT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &c.Port); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid GLOSKI_PORT value %q: %v\n", v, err)
		}
	}
	if v := os.Getenv("GLOSKI_BASE_URL"); v != "" {
		c.BaseURL = v
	}
	if v := os.Getenv("GLOSKI_API_PREFIX"); v != "" {
		c.APIPrefix = v
	}
	if v := os.Getenv("GLOSKI_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := os.Getenv("GLOSKI_SHELL"); v != "" {
		c.Shell = v
	}
	if v := os.Getenv("GLOSKI_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("GLOSKI_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("GLOSKI_JWT_PUBLIC_KEY"); v != "" {
		c.JWTPublicKey = v
	}
	if v := os.Getenv("GLOSKI_JWT_PUBLIC_KEY_FILE"); v != "" {
		c.JWTPublicKeyFile = v
	}
	if v := os.Getenv("GLOSKI_DOWNLOADS_ENABLED"); v != "" {
		c.Downloads.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("GLOSKI_DOWNLOADS_MAX_CONCURRENT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &c.Downloads.MaxConcurrent); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid GLOSKI_DOWNLOADS_MAX_CONCURRENT value %q: %v\n", v, err)
		}
	}
	if v := os.Getenv("GLOSKI_JOBS_ENABLED"); v != "" {
		c.Jobs.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("GLOSKI_JOBS_MAX_JOBS"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &c.Jobs.MaxJobs); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid GLOSKI_JOBS_MAX_JOBS value %q: %v\n", v, err)
		}
	}
	if v := os.Getenv("GLOSKI_SHUTDOWN_TIMEOUT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &c.ShutdownTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid GLOSKI_SHUTDOWN_TIMEOUT value %q: %v\n", v, err)
		}
	}
	if v := os.Getenv("GLOSKI_DETAILED_ERRORS"); v != "" {
		c.DetailedErrors = v == "true" || v == "1"
	}
	if v := os.Getenv("GLOSKI_MAX_JSON_BODY_SIZE"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &c.MaxJSONBodySize); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid GLOSKI_MAX_JSON_BODY_SIZE value %q: %v\n", v, err)
		}
	}
	if v := os.Getenv("GLOSKI_MASTER_WEB_DIR"); v != "" {
		c.Master.WebDir = v
	}
	if v := os.Getenv("GLOSKI_MASTER_JOIN_TOKEN"); v != "" {
		c.Master.JoinToken = v
	}
	if v := os.Getenv("GLOSKI_MASTER_DEVICE_TIMEOUT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &c.Master.DeviceTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid GLOSKI_MASTER_DEVICE_TIMEOUT value %q: %v\n", v, err)
		}
	}
	if v := os.Getenv("GLOSKI_MASTER_HEARTBEAT_INTERVAL"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &c.Master.HeartbeatInterval); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid GLOSKI_MASTER_HEARTBEAT_INTERVAL value %q: %v\n", v, err)
		}
	}
	if v := os.Getenv("GLOSKI_AGENT_MASTER_URL"); v != "" {
		c.Agent.MasterURL = v
	}
	if v := os.Getenv("GLOSKI_AGENT_JOIN_TOKEN"); v != "" {
		c.Agent.JoinToken = v
	}
	if v := os.Getenv("GLOSKI_AGENT_NAME"); v != "" {
		c.Agent.Name = v
	}
	if v := os.Getenv("GLOSKI_AGENT_ADVERTISE_URL"); v != "" {
		c.Agent.AdvertiseURL = v
	}
	if v := os.Getenv("GLOSKI_AGENT_HEARTBEAT_INTERVAL"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &c.Agent.HeartbeatInterval); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid GLOSKI_AGENT_HEARTBEAT_INTERVAL value %q: %v\n", v, err)
		}
	}
	if v := os.Getenv("GLOSKI_AGENT_STATE_FILE"); v != "" {
		c.Agent.StateFile = v
	}
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
	case ModeAgent:
		// The device token is assigned during enrollment, so API auth is not
		// required in the local config.
		if c.Agent.HeartbeatInterval <= 0 {
			c.Agent.HeartbeatInterval = 10
		}
	case ModeStandalone:
		// At least one auth method is required
		hasAPIKey := c.APIKey != ""
		hasJWT := c.JWTPublicKey != "" || c.JWTPublicKeyFile != ""

		if !hasAPIKey && !hasJWT {
			return fmt.Errorf("at least one authentication method is required (set GLOSKI_API_KEY or GLOSKI_JWT_PUBLIC_KEY)")
		}
	default:
		return fmt.Errorf("invalid mode %q (expected standalone, master, or agent)", c.Mode)
	}

	// Load JWT public key from file if specified
	if c.JWTPublicKeyFile != "" && c.JWTPublicKey == "" {
		data, err := os.ReadFile(c.JWTPublicKeyFile)
		if err != nil {
			return fmt.Errorf("Failed to read JWT public key file: %w", err)
		}
		c.JWTPublicKey = string(data)
	}

	return nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

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
