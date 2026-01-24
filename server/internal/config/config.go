package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	// Server settings
	Host string `json:"host"`
	Port int    `json:"port"`

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
}

func DefaultConfig() *Config {
	return &Config{
		Host:           "127.0.0.1",
		Port:           8080,
		AllowedOrigins: []string{"*"},
		AllowedPaths:   []string{},
		Shell:          getDefaultShell(),
		LogLevel:       "info",
	}
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
		} else {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
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
			return fmt.Errorf("failed to read JWT public key file: %w", err)
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
		// Check if path is under allowed path
		rel, err := filepath.Rel(allowedAbs, absPath)
		if err != nil {
			continue
		}
		if len(rel) >= 2 && rel[:2] == ".." {
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
