package system

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/logger"
)

// Collector periodically collects system stats and updates the store.
type Collector struct {
	store    *Store
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}

	// For CPU delta calculation
	prevCPU     CPUTimes
	prevCPUTime time.Time
}

// NewCollector creates a new stats collector.
func NewCollector(store *Store, interval time.Duration) *Collector {
	return &Collector{
		store:    store,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins collecting stats in the background.
func (c *Collector) Start() {
	// Initial CPU read for delta calculation
	c.prevCPU, _ = c.readCPUTimes()
	c.prevCPUTime = time.Now()

	// Collect immediately on start
	c.collect()

	go c.run()
	logger.Info("System stats collector started (interval: %s)", c.interval)
}

// Stop stops the collector.
func (c *Collector) Stop() {
	close(c.stopCh)
	<-c.doneCh
	logger.Info("System stats collector stopped")
}

func (c *Collector) run() {
	defer close(c.doneCh)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.collect()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Collector) collect() {
	stats := &Stats{
		Platform:   runtime.GOOS + "/" + runtime.GOARCH,
		GoRoutines: runtime.NumGoroutine(),
	}

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		stats.Hostname = hostname
	}

	// CPU info
	stats.CPU = c.getCPUStats()

	// Memory
	stats.Memory = c.getMemoryStats()
	stats.Swap = c.getSwapStats()

	// Disks
	stats.Disks = c.getDiskStats()

	// Network
	stats.Network = c.getNetworkStats()

	// Load average
	stats.LoadAvg = c.getLoadAvg()

	// Uptime and boot time
	stats.Uptime, stats.BootTime = c.getUptime()

	// Process count
	stats.Processes = c.getProcessCount()

	c.store.Push(stats)
}

func (c *Collector) getCPUStats() CPUStats {
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
	current, err := c.readCPUTimes()
	if err != nil {
		return stats
	}

	// Calculate deltas
	totalDelta := float64(current.Total - c.prevCPU.Total)
	if totalDelta > 0 {
		stats.User = float64(current.User-c.prevCPU.User) / totalDelta * 100
		stats.System = float64(current.System-c.prevCPU.System) / totalDelta * 100
		stats.Idle = float64(current.Idle-c.prevCPU.Idle) / totalDelta * 100
		stats.IOWait = float64(current.IOWait-c.prevCPU.IOWait) / totalDelta * 100
		stats.UsagePercent = 100 - stats.Idle
	}

	// Update previous values
	c.prevCPU = current
	c.prevCPUTime = time.Now()

	return stats
}

func (c *Collector) readCPUTimes() (CPUTimes, error) {
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

func (c *Collector) getMemoryStats() MemoryStats {
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

func (c *Collector) getSwapStats() SwapStats {
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

func (c *Collector) getDiskStats() []DiskStats {
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

func (c *Collector) getNetworkStats() NetStats {
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

func (c *Collector) getLoadAvg() LoadAvg {
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

func (c *Collector) getUptime() (int64, int64) {
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

func (c *Collector) getProcessCount() int {
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
