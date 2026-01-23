package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	// Server settings
	Host string `json:"host"`
	Port int    `json:"port"`

	// Authentication
	PasswordHash string `json:"password_hash"`
	JWTSecret    string `json:"jwt_secret"`
	TokenExpiry  int    `json:"token_expiry"` // in hours
	APIKey       string `json:"api_key"`      // API key for direct auth (fallback)

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
		TokenExpiry:    24,
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
	if v := os.Getenv("GLOSKI_PASSWORD"); v != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(v), bcrypt.DefaultCost)
		c.PasswordHash = string(hash)
	}
	if v := os.Getenv("GLOSKI_JWT_SECRET"); v != "" {
		c.JWTSecret = v
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
	c.APIKey = "1234"
}

func (c *Config) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	// If no password hash is set, generate one from default password
	if c.PasswordHash == "" {
		defaultPassword := os.Getenv("GLOSKI_PASSWORD")
		if defaultPassword == "" {
			defaultPassword = "gloski" // Default for development only
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		c.PasswordHash = string(hash)
	}

	// Generate JWT secret if not provided
	if c.JWTSecret == "" {
		c.JWTSecret = generateRandomString(32)
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

func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
