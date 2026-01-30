package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ss497254/gloski/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %s, want 127.0.0.1", cfg.Host)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %s, want info", cfg.LogLevel)
	}

	if cfg.Shell == "" {
		t.Error("Shell should not be empty")
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		envVars   map[string]string
		wantErr   bool
		checkFunc func(*testing.T, *config.Config)
	}{
		{
			name: "valid config with API key",
			config: `{
				"host": "0.0.0.0",
				"port": 9090,
				"api_key": "test-key",
				"log_level": "debug"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, cfg *config.Config) {
				if cfg.Host != "0.0.0.0" {
					t.Errorf("Host = %s, want 0.0.0.0", cfg.Host)
				}
				if cfg.Port != 9090 {
					t.Errorf("Port = %d, want 9090", cfg.Port)
				}
				if cfg.APIKey != "test-key" {
					t.Errorf("APIKey = %s, want test-key", cfg.APIKey)
				}
				if cfg.LogLevel != "debug" {
					t.Errorf("LogLevel = %s, want debug", cfg.LogLevel)
				}
			},
		},
		{
			name: "environment variables override config",
			config: `{
				"host": "0.0.0.0",
				"port": 9090,
				"api_key": "config-key"
			}`,
			envVars: map[string]string{
				"GLOSKI_HOST":    "127.0.0.1",
				"GLOSKI_PORT":    "3000",
				"GLOSKI_API_KEY": "env-key",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, cfg *config.Config) {
				if cfg.Host != "127.0.0.1" {
					t.Errorf("Host = %s, want 127.0.0.1 (from env)", cfg.Host)
				}
				if cfg.Port != 3000 {
					t.Errorf("Port = %d, want 3000 (from env)", cfg.Port)
				}
				if cfg.APIKey != "env-key" {
					t.Errorf("APIKey = %s, want env-key (from env)", cfg.APIKey)
				}
			},
		},
		{
			name:    "missing authentication",
			config:  `{"host": "127.0.0.1", "port": 8080}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to create test config: %v", err)
			}

			// Set environment variables
			for key, value := range tt.envVars {
				oldValue := os.Getenv(key)
				os.Setenv(key, value)
				defer os.Setenv(key, oldValue)
			}

			// Load config
			cfg, err := config.Load(configPath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, cfg)
			}
		})
	}
}

func TestIsPathAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	os.MkdirAll(testDir, 0755)

	tests := []struct {
		name         string
		allowedPaths []string
		testPath     string
		want         bool
	}{
		{
			name:         "empty allowed paths allows all",
			allowedPaths: []string{},
			testPath:     "/any/path",
			want:         true,
		},
		{
			name:         "exact match",
			allowedPaths: []string{testDir},
			testPath:     testDir,
			want:         true,
		},
		{
			name:         "subdirectory allowed",
			allowedPaths: []string{tmpDir},
			testPath:     testDir,
			want:         true,
		},
		{
			name:         "parent directory not allowed",
			allowedPaths: []string{testDir},
			testPath:     tmpDir,
			want:         false,
		},
		{
			name:         "unrelated path not allowed",
			allowedPaths: []string{tmpDir},
			testPath:     "/etc/passwd",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AllowedPaths: tt.allowedPaths,
				APIKey:       "test",
			}

			got := cfg.IsPathAllowed(tt.testPath)
			if got != tt.want {
				t.Errorf("IsPathAllowed(%s) = %v, want %v", tt.testPath, got, tt.want)
			}
		})
	}
}

func TestConfigAddr(t *testing.T) {
	cfg := &config.Config{
		Host: "127.0.0.1",
		Port: 8080,
	}

	got := cfg.Addr()
	want := "127.0.0.1:8080"

	if got != want {
		t.Errorf("Addr() = %s, want %s", got, want)
	}
}

func TestConfigSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := &config.Config{
		Host:   "127.0.0.1",
		Port:   8080,
		APIKey: "test-key",
	}

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Load and verify
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	if len(data) == 0 {
		t.Error("config file is empty")
	}
}
