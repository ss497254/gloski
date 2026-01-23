package system

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Service provides system information by reading from the store.
type Service struct {
	store *Store
}

// NewService creates a new system service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// GetStats returns the latest system stats from the store.
func (s *Service) GetStats() (*Stats, error) {
	stats := s.store.Get()
	if stats == nil {
		return &Stats{}, nil
	}
	return stats, nil
}

// HasSystemd checks if systemd is available on the system.
func (s *Service) HasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// ServerInfo represents detailed server information.
type ServerInfo struct {
	Hostname     string            `json:"hostname"`
	OS           string            `json:"os"`
	OSVersion    string            `json:"os_version"`
	Kernel       string            `json:"kernel"`
	Architecture string            `json:"architecture"`
	CPUModel     string            `json:"cpu_model"`
	CPUCores     int               `json:"cpu_cores"`
	TotalMemory  uint64            `json:"total_memory"`
	TotalSwap    uint64            `json:"total_swap"`
	Timezone     string            `json:"timezone"`
	Uptime       int64             `json:"uptime"`
	BootTime     int64             `json:"boot_time"`
	Features     map[string]bool   `json:"features"`
	Environment  map[string]string `json:"environment,omitempty"`
}

// GetServerInfo returns detailed server information.
func (s *Service) GetServerInfo() (*ServerInfo, error) {
	info := &ServerInfo{
		Architecture: runtime.GOARCH,
		CPUCores:     runtime.NumCPU(),
		Features:     make(map[string]bool),
	}

	// Hostname
	info.Hostname, _ = os.Hostname()

	// OS information from /etc/os-release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		osInfo := parseOSRelease(string(data))
		info.OS = osInfo["NAME"]
		info.OSVersion = osInfo["VERSION"]
		if info.OSVersion == "" {
			info.OSVersion = osInfo["VERSION_ID"]
		}
	}

	// Kernel version
	if data, err := os.ReadFile("/proc/version"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			info.Kernel = parts[2]
		}
	}

	// CPU model
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.CPUModel = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	// Get memory/swap from store if available, otherwise read directly
	if stats := s.store.Get(); stats != nil {
		info.TotalMemory = stats.Memory.Total
		info.TotalSwap = stats.Swap.Total
		info.Uptime = stats.Uptime
		info.BootTime = stats.BootTime
	}

	// Timezone
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		info.Timezone = strings.TrimSpace(string(data))
	} else if link, err := os.Readlink("/etc/localtime"); err == nil {
		if idx := strings.Index(link, "zoneinfo/"); idx >= 0 {
			info.Timezone = link[idx+9:]
		}
	}

	// Detect available features
	info.Features["systemd"] = s.HasSystemd()
	info.Features["apt"] = s.hasCommand("apt")
	info.Features["dnf"] = s.hasCommand("dnf")
	info.Features["yum"] = s.hasCommand("yum")
	info.Features["apk"] = s.hasCommand("apk")
	info.Features["cron"] = s.hasCommand("crontab")
	info.Features["ufw"] = s.hasCommand("ufw")
	info.Features["firewalld"] = s.hasCommand("firewall-cmd")

	return info, nil
}

func (s *Service) hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func parseOSRelease(content string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "="); idx > 0 {
			key := line[:idx]
			value := line[idx+1:]
			value = strings.Trim(value, "\"'")
			result[key] = value
		}
	}
	return result
}

// Process represents a running process.
type Process struct {
	PID     int     `json:"pid"`
	Name    string  `json:"name"`
	State   string  `json:"state"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	VSZ     uint64  `json:"vsz"`
	RSS     uint64  `json:"rss"`
	Command string  `json:"command"`
}

// GetProcesses returns a list of running processes.
func (s *Service) GetProcesses(limit int) ([]Process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var processes []Process

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		proc, err := s.readProcess(pid)
		if err != nil {
			continue
		}

		processes = append(processes, proc)
		if limit > 0 && len(processes) >= limit {
			break
		}
	}

	return processes, nil
}

func (s *Service) readProcess(pid int) (Process, error) {
	proc := Process{PID: pid}
	procDir := filepath.Join("/proc", strconv.Itoa(pid))

	// Read stat file
	statData, err := os.ReadFile(filepath.Join(procDir, "stat"))
	if err != nil {
		return proc, err
	}

	// Parse stat - format: pid (comm) state ...
	statStr := string(statData)

	// Find command name between parentheses
	start := strings.Index(statStr, "(")
	end := strings.LastIndex(statStr, ")")
	if start >= 0 && end > start {
		proc.Name = statStr[start+1 : end]
		statStr = statStr[end+2:] // Skip ") "
	}

	fields := strings.Fields(statStr)
	if len(fields) > 0 {
		proc.State = fields[0]
	}

	// Read RSS from statm
	if statmData, err := os.ReadFile(filepath.Join(procDir, "statm")); err == nil {
		statmFields := strings.Fields(string(statmData))
		if len(statmFields) >= 2 {
			vsz, _ := strconv.ParseUint(statmFields[0], 10, 64)
			rss, _ := strconv.ParseUint(statmFields[1], 10, 64)
			pageSize := uint64(os.Getpagesize())
			proc.VSZ = vsz * pageSize
			proc.RSS = rss * pageSize
		}
	}

	// Read cmdline
	if cmdData, err := os.ReadFile(filepath.Join(procDir, "cmdline")); err == nil {
		proc.Command = strings.ReplaceAll(string(cmdData), "\x00", " ")
		proc.Command = strings.TrimSpace(proc.Command)
		if proc.Command == "" {
			proc.Command = "[" + proc.Name + "]"
		}
	}

	return proc, nil
}
