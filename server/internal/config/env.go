package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// loadFromEnv applies environment overrides on top of the config file.
func (c *Config) loadFromEnv() {
	c.loadCommonEnv()
	c.loadFeatureEnv()
	c.loadMasterEnv()
	c.loadAgentEnv()
}

func (c *Config) loadCommonEnv() {
	envString("GLOSKI_MODE", &c.Mode)
	envString("GLOSKI_HOST", &c.Host)
	envInt("GLOSKI_PORT", &c.Port)
	envString("GLOSKI_BASE_URL", &c.BaseURL)
	envString("GLOSKI_API_PREFIX", &c.APIPrefix)
	envString("GLOSKI_DATA_DIR", &c.DataDir)
	envString("GLOSKI_SHELL", &c.Shell)
	envString("GLOSKI_LOG_LEVEL", &c.LogLevel)
	envInt("GLOSKI_SHUTDOWN_TIMEOUT", &c.ShutdownTimeout)
	envBool("GLOSKI_DETAILED_ERRORS", &c.DetailedErrors)
	envInt64("GLOSKI_MAX_JSON_BODY_SIZE", &c.MaxJSONBodySize)
}

func (c *Config) loadFeatureEnv() {
	envList("GLOSKI_ALLOWED_ORIGINS", &c.AllowedOrigins)
	envString("GLOSKI_API_KEY", &c.APIKey)
	envString("GLOSKI_JWT_PUBLIC_KEY", &c.JWTPublicKey)
	envString("GLOSKI_JWT_PUBLIC_KEY_FILE", &c.JWTPublicKeyFile)
	envBool("GLOSKI_DOWNLOADS_ENABLED", &c.Downloads.Enabled)
	envInt("GLOSKI_DOWNLOADS_MAX_CONCURRENT", &c.Downloads.MaxConcurrent)
	envBool("GLOSKI_JOBS_ENABLED", &c.Jobs.Enabled)
	envInt("GLOSKI_JOBS_MAX_JOBS", &c.Jobs.MaxJobs)

	if raw := strings.TrimSpace(os.Getenv("GLOSKI_PROBES")); raw != "" {
		var probes []ProbeDefinition
		if err := json.Unmarshal([]byte(raw), &probes); err != nil {
			fmt.Fprintf(os.Stderr, "gloski: ignoring GLOSKI_PROBES: %v\n", err)
		} else {
			c.Probes = probes
		}
	}
}

func (c *Config) loadMasterEnv() {
	envString("GLOSKI_MASTER_WEB_DIR", &c.Master.WebDir)
	envString("GLOSKI_MASTER_JOIN_TOKEN", &c.Master.JoinToken)
	envInt("GLOSKI_MASTER_DEVICE_TIMEOUT", &c.Master.DeviceTimeout)
	envInt("GLOSKI_MASTER_HEARTBEAT_INTERVAL", &c.Master.HeartbeatInterval)
	envString("GLOSKI_MASTER_AGENT_BINARY", &c.Master.AgentBinary)
	envString("GLOSKI_MASTER_SSH_KEY_FILE", &c.Master.SSHKeyFile)
	envInt("GLOSKI_MASTER_DEPLOY_TIMEOUT", &c.Master.DeployTimeout)
	envBool("GLOSKI_MASTER_LOCAL_AGENT", &c.Master.LocalAgent)
	envString("GLOSKI_MASTER_LOCAL_AGENT_NAME", &c.Master.LocalAgentName)
	envString("GLOSKI_MASTER_AGENT_GRPC_HOST", &c.Master.AgentGRPCHost)
	envInt("GLOSKI_MASTER_AGENT_GRPC_PORT", &c.Master.AgentGRPCPort)
	envString("GLOSKI_MASTER_AGENT_GRPC_URL", &c.Master.AgentGRPCURL)
}

func (c *Config) loadAgentEnv() {
	envString("GLOSKI_AGENT_MASTER_URL", &c.Agent.MasterURL)
	envString("GLOSKI_AGENT_JOIN_TOKEN", &c.Agent.JoinToken)
	envString("GLOSKI_AGENT_NAME", &c.Agent.Name)
	envInt("GLOSKI_AGENT_HEARTBEAT_INTERVAL", &c.Agent.HeartbeatInterval)
	envString("GLOSKI_AGENT_DEPLOY_ID", &c.Agent.DeployID)
	envBool("GLOSKI_AGENT_IS_MASTER", &c.Agent.Master)
	envString("GLOSKI_TAILSCALE_SOCKET", &c.Tailscale.Socket)
}

func envString(key string, target *string) {
	if value := os.Getenv(key); value != "" {
		*target = value
	}
}

func envInt(key string, target *int) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		warnInvalidEnv(key, value, err)
		return
	}
	*target = parsed
}

func envInt64(key string, target *int64) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		warnInvalidEnv(key, value, err)
		return
	}
	*target = parsed
}

func envBool(key string, target *bool) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	*target = value == "true" || value == "1"
}

func warnInvalidEnv(key, value string, err error) {
	fmt.Fprintf(os.Stderr, "warning: invalid %s value %q: %v\n", key, strings.TrimSpace(value), err)
}

// envList reads a comma-separated list.
func envList(key string, target *[]string) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	list := []string{}
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			list = append(list, item)
		}
	}
	*target = list
}
