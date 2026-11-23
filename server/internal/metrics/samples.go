package metrics

import (
	"strings"

	"github.com/ss497254/gloski/internal/system"
)

// Metric names are stable strings: a chart asks for one by name, and renaming
// one would break its stored history.
const (
	MetricCPU       = "cpu"
	MetricMemory    = "memory"
	MetricSwap      = "swap"
	MetricLoad      = "load1"
	MetricNetworkRX = "network_rx"
	MetricNetworkTX = "network_tx"

	diskMetricPrefix = "disk:"
)

// DiskMetric is the metric name for one mount point. Mount points are part of
// the name so two disks on one device keep separate histories.
func DiskMetric(mountPoint string) string {
	return diskMetricPrefix + mountPoint
}

// DiskMountPoint reverses DiskMetric. It returns false for other metrics.
func DiskMountPoint(metric string) (string, bool) {
	if !strings.HasPrefix(metric, diskMetricPrefix) {
		return "", false
	}
	mount := strings.TrimPrefix(metric, diskMetricPrefix)
	return mount, mount != ""
}

// SamplesFromStats turns one stats snapshot into the metric series we keep.
// It is pure so the heartbeat path only does a small, obvious conversion.
func SamplesFromStats(stats *system.Stats) []Sample {
	if stats == nil {
		return nil
	}
	samples := []Sample{
		{Metric: MetricCPU, Value: stats.CPU.UsagePercent},
		{Metric: MetricMemory, Value: stats.Memory.UsedPercent},
		{Metric: MetricLoad, Value: stats.LoadAvg.Load1},
		{Metric: MetricNetworkRX, Value: stats.Network.RxRate},
		{Metric: MetricNetworkTX, Value: stats.Network.TxRate},
	}
	if stats.Swap.Total > 0 {
		samples = append(samples, Sample{Metric: MetricSwap, Value: stats.Swap.UsedPercent})
	}
	for _, disk := range stats.Disks {
		if disk.Total == 0 {
			continue
		}
		samples = append(samples, Sample{Metric: DiskMetric(disk.MountPoint), Value: disk.UsedPercent})
	}
	return samples
}
