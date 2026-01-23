package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	prevCPU     CPUTimes
	prevCPUTime time.Time
}

func NewService() *Service {
	s := &Service{}
	// Initialize CPU times for accurate first reading
	s.prevCPU, _ = s.readCPUTimes()
	s.prevCPUTime = time.Now()
	return s
}

// Stats represents complete system statistics
type Stats struct {
	Hostname   string      `json:"hostname"`
	Platform   string      `json:"platform"`
	Uptime     int64       `json:"uptime"`
	BootTime   int64       `json:"boot_time"`
	CPU        CPUStats    `json:"cpu"`
	Memory     MemoryStats `json:"memory"`
	Swap       SwapStats   `json:"swap"`
	Disks      []DiskStats `json:"disks"`
	Network    NetStats    `json:"network"`
	LoadAvg    LoadAvg     `json:"load_avg"`
	Processes  int         `json:"processes"`
	GoRoutines int         `json:"goroutines"`
}

type CPUStats struct {
	Cores        int     `json:"cores"`
	ModelName    string  `json:"model_name"`
	UsagePercent float64 `json:"usage_percent"`
	User         float64 `json:"user"`
	System       float64 `json:"system"`
	Idle         float64 `json:"idle"`
	IOWait       float64 `json:"iowait"`
}

type CPUTimes struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Steal   uint64
	Total   uint64
}

type MemoryStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	Available   uint64  `json:"available"`
	Cached      uint64  `json:"cached"`
	Buffers     uint64  `json:"buffers"`
	UsedPercent float64 `json:"used_percent"`
}

type SwapStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type DiskStats struct {
	Device      string  `json:"device"`
	MountPoint  string  `json:"mount_point"`
	FSType      string  `json:"fs_type"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type NetStats struct {
	Interfaces []NetInterface `json:"interfaces"`
	TotalRx    uint64         `json:"total_rx"`
	TotalTx    uint64         `json:"total_tx"`
}

type NetInterface struct {
	Name    string `json:"name"`
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
	RxPkts  uint64 `json:"rx_packets"`
	TxPkts  uint64 `json:"tx_packets"`
	Up      bool   `json:"up"`
}

type LoadAvg struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// GetStats returns comprehensive system statistics
func (s *Service) GetStats() (*Stats, error) {
	stats := &Stats{
		Platform:   runtime.GOOS + "/" + runtime.GOARCH,
		GoRoutines: runtime.NumGoroutine(),
	}

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		stats.Hostname = hostname
	}

	// CPU info
	stats.CPU = s.getCPUStats()

	// Memory
	stats.Memory = s.getMemoryStats()
	stats.Swap = s.getSwapStats()

	// Disks
	stats.Disks = s.getDiskStats()

	// Network
	stats.Network = s.getNetworkStats()

	// Load average
	stats.LoadAvg = s.getLoadAvg()

	// Uptime and boot time
	stats.Uptime, stats.BootTime = s.getUptime()

	// Process count
	stats.Processes = s.getProcessCount()

	return stats, nil
}

func (s *Service) getCPUStats() CPUStats {
	stats := CPUStats{
		Cores: runtime.NumCPU(),
	}

	// Read CPU model
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					stats.ModelName = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	// Calculate CPU usage
	current, err := s.readCPUTimes()
	if err != nil {
		return stats
	}

	// Calculate deltas
	totalDelta := float64(current.Total - s.prevCPU.Total)
	if totalDelta > 0 {
		stats.User = float64(current.User-s.prevCPU.User) / totalDelta * 100
		stats.System = float64(current.System-s.prevCPU.System) / totalDelta * 100
		stats.Idle = float64(current.Idle-s.prevCPU.Idle) / totalDelta * 100
		stats.IOWait = float64(current.IOWait-s.prevCPU.IOWait) / totalDelta * 100
		stats.UsagePercent = 100 - stats.Idle
	}

	// Update previous values
	s.prevCPU = current
	s.prevCPUTime = time.Now()

	return stats
}

func (s *Service) readCPUTimes() (CPUTimes, error) {
	var times CPUTimes

	file, err := os.Open("/proc/stat")
	if err != nil {
		return times, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 8 {
				times.User, _ = strconv.ParseUint(fields[1], 10, 64)
				times.Nice, _ = strconv.ParseUint(fields[2], 10, 64)
				times.System, _ = strconv.ParseUint(fields[3], 10, 64)
				times.Idle, _ = strconv.ParseUint(fields[4], 10, 64)
				times.IOWait, _ = strconv.ParseUint(fields[5], 10, 64)
				times.IRQ, _ = strconv.ParseUint(fields[6], 10, 64)
				times.SoftIRQ, _ = strconv.ParseUint(fields[7], 10, 64)
				if len(fields) >= 9 {
					times.Steal, _ = strconv.ParseUint(fields[8], 10, 64)
				}
				times.Total = times.User + times.Nice + times.System + times.Idle +
					times.IOWait + times.IRQ + times.SoftIRQ + times.Steal
			}
			break
		}
	}

	return times, nil
}

func (s *Service) getMemoryStats() MemoryStats {
	stats := MemoryStats{}

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return stats
	}
	defer file.Close()

	memInfo := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			memInfo[key] = value * 1024 // Convert KB to bytes
		}
	}

	stats.Total = memInfo["MemTotal"]
	stats.Free = memInfo["MemFree"]
	stats.Available = memInfo["MemAvailable"]
	stats.Buffers = memInfo["Buffers"]
	stats.Cached = memInfo["Cached"]
	stats.Used = stats.Total - stats.Available

	if stats.Total > 0 {
		stats.UsedPercent = float64(stats.Used) / float64(stats.Total) * 100
	}

	return stats
}

func (s *Service) getSwapStats() SwapStats {
	stats := SwapStats{}

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return stats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			switch key {
			case "SwapTotal":
				stats.Total = value * 1024
			case "SwapFree":
				stats.Free = value * 1024
			}
		}
	}

	stats.Used = stats.Total - stats.Free
	if stats.Total > 0 {
		stats.UsedPercent = float64(stats.Used) / float64(stats.Total) * 100
	}

	return stats
}

func (s *Service) getDiskStats() []DiskStats {
	var disks []DiskStats

	file, err := os.Open("/proc/mounts")
	if err != nil {
		return disks
	}
	defer file.Close()

	seen := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		// Skip virtual filesystems
		if !strings.HasPrefix(device, "/dev/") {
			continue
		}

		// Skip duplicates
		if seen[device] {
			continue
		}
		seen[device] = true

		// Get disk usage via statfs
		var stat StatFS
		if err := Statfs(mountPoint, &stat); err != nil {
			continue
		}

		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		used := total - free

		usedPercent := float64(0)
		if total > 0 {
			usedPercent = float64(used) / float64(total) * 100
		}

		disks = append(disks, DiskStats{
			Device:      device,
			MountPoint:  mountPoint,
			FSType:      fsType,
			Total:       total,
			Used:        used,
			Free:        free,
			UsedPercent: usedPercent,
		})
	}

	return disks
}

func (s *Service) getNetworkStats() NetStats {
	stats := NetStats{
		Interfaces: []NetInterface{},
	}

	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return stats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue // Skip header lines
		}

		line := scanner.Text()
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}

		name := strings.TrimSpace(line[:colonIdx])
		fields := strings.Fields(line[colonIdx+1:])

		if len(fields) < 10 {
			continue
		}

		// Skip loopback
		if name == "lo" {
			continue
		}

		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		rxPkts, _ := strconv.ParseUint(fields[1], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)
		txPkts, _ := strconv.ParseUint(fields[9], 10, 64)

		// Check if interface is up
		up := false
		if flagsData, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/operstate", name)); err == nil {
			up = strings.TrimSpace(string(flagsData)) == "up"
		}

		iface := NetInterface{
			Name:    name,
			RxBytes: rxBytes,
			TxBytes: txBytes,
			RxPkts:  rxPkts,
			TxPkts:  txPkts,
			Up:      up,
		}

		stats.Interfaces = append(stats.Interfaces, iface)
		stats.TotalRx += rxBytes
		stats.TotalTx += txBytes
	}

	return stats
}

func (s *Service) getLoadAvg() LoadAvg {
	var load LoadAvg

	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return load
	}

	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		load.Load1, _ = strconv.ParseFloat(fields[0], 64)
		load.Load5, _ = strconv.ParseFloat(fields[1], 64)
		load.Load15, _ = strconv.ParseFloat(fields[2], 64)
	}

	return load
}

func (s *Service) getUptime() (int64, int64) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, 0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, 0
	}

	uptime, _ := strconv.ParseFloat(fields[0], 64)
	bootTime := time.Now().Unix() - int64(uptime)

	return int64(uptime), bootTime
}

func (s *Service) getProcessCount() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := strconv.Atoi(entry.Name()); err == nil {
				count++
			}
		}
	}

	return count
}

// GetProcesses returns a list of running processes
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

	// Memory info
	mem := s.getMemoryStats()
	info.TotalMemory = mem.Total
	swap := s.getSwapStats()
	info.TotalSwap = swap.Total

	// Timezone
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		info.Timezone = strings.TrimSpace(string(data))
	} else if link, err := os.Readlink("/etc/localtime"); err == nil {
		// Extract timezone from symlink like /usr/share/zoneinfo/America/New_York
		if idx := strings.Index(link, "zoneinfo/"); idx >= 0 {
			info.Timezone = link[idx+9:]
		}
	}

	// Uptime
	info.Uptime, info.BootTime = s.getUptime()

	// Detect available features
	info.Features["systemd"] = s.HasSystemd()
	info.Features["docker"] = s.hasCommand("docker")
	info.Features["podman"] = s.hasCommand("podman")
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
			// Remove quotes
			value = strings.Trim(value, "\"'")
			result[key] = value
		}
	}
	return result
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
