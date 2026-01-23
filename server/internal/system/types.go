package system

// Stats represents complete system statistics.
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

// CPUStats represents CPU usage statistics.
type CPUStats struct {
	Cores        int     `json:"cores"`
	ModelName    string  `json:"model_name"`
	UsagePercent float64 `json:"usage_percent"`
	User         float64 `json:"user"`
	System       float64 `json:"system"`
	Idle         float64 `json:"idle"`
	IOWait       float64 `json:"iowait"`
}

// CPUTimes represents raw CPU time values for delta calculation.
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

// MemoryStats represents memory usage statistics.
type MemoryStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	Available   uint64  `json:"available"`
	Cached      uint64  `json:"cached"`
	Buffers     uint64  `json:"buffers"`
	UsedPercent float64 `json:"used_percent"`
}

// SwapStats represents swap usage statistics.
type SwapStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// DiskStats represents disk usage statistics.
type DiskStats struct {
	Device      string  `json:"device"`
	MountPoint  string  `json:"mount_point"`
	FSType      string  `json:"fs_type"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// NetStats represents network statistics.
type NetStats struct {
	Interfaces []NetInterface `json:"interfaces"`
	TotalRx    uint64         `json:"total_rx"`
	TotalTx    uint64         `json:"total_tx"`
}

// NetInterface represents a network interface.
type NetInterface struct {
	Name    string `json:"name"`
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
	RxPkts  uint64 `json:"rx_packets"`
	TxPkts  uint64 `json:"tx_packets"`
	Up      bool   `json:"up"`
}

// LoadAvg represents system load averages.
type LoadAvg struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}
