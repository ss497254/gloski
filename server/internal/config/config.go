package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
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
	}
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
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("Failed to read config file: %w\n", err)
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
	if v := os.Getenv("GLOSKI_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("GLOSKI_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Port)
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
		fmt.Sscanf(v, "%d", &c.Downloads.MaxConcurrent)
	}
	if v := os.Getenv("GLOSKI_JOBS_ENABLED"); v != "" {
		c.Jobs.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("GLOSKI_JOBS_MAX_JOBS"); v != "" {
		fmt.Sscanf(v, "%d", &c.Jobs.MaxJobs)
	}
	if v := os.Getenv("GLOSKI_SHUTDOWN_TIMEOUT"); v != "" {
		fmt.Sscanf(v, "%d", &c.ShutdownTimeout)
	}
	if v := os.Getenv("GLOSKI_DETAILED_ERRORS"); v != "" {
		c.DetailedErrors = v == "true" || v == "1"
	}
	if v := os.Getenv("GLOSKI_MAX_JSON_BODY_SIZE"); v != "" {
		fmt.Sscanf(v, "%d", &c.MaxJSONBodySize)
	}
}

func (c *Config) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	// At least one auth method is required
	hasAPIKey := c.APIKey != ""
	hasJWT := c.JWTPublicKey != "" || c.JWTPublicKeyFile != ""

	if !hasAPIKey && !hasJWT {
		return fmt.Errorf("at least one authentication method is required (set GLOSKI_API_KEY or GLOSKI_JWT_PUBLIC_KEY)")
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
