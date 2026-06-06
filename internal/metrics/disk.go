package metrics

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"math"
	"strings"
)

const (
	diskAwaitHistoryCapMS = 50
	diskQueueHistoryCap   = 8
)

// DiskUtilPercent returns the busiest sampled disk utilization.
func DiskUtilPercent(s core.Sample) int {
	util := s.DiskUtil
	for _, disk := range s.Disks {
		util = maxKnownDiskPercent(util, disk.Util)
	}

	return util
}

func maxKnownDiskPercent(a, b int) int {
	switch {
	case a < 0:
		return b
	case b < 0:
		return a
	default:
		return max(a, b)
	}
}

func diskAwaitHistoryPercent(awaitMS float64) int {
	if awaitMS < 0 {
		return -1
	}
	capped := math.Min(awaitMS, diskAwaitHistoryCapMS)

	return Clamp(int(math.Round((capped/diskAwaitHistoryCapMS)*percentScale)), percentMin, percentMax)
}

func diskQueueHistoryPercent(queueDepth float64) int {
	if queueDepth < 0 {
		return -1
	}
	capped := math.Min(queueDepth, diskQueueHistoryCap)

	return Clamp(int(math.Round((capped/diskQueueHistoryCap)*percentScale)), percentMin, percentMax)
}

// DiskLatencyHistoryPercent folds disk await and queue depth into one history value.
func DiskLatencyHistoryPercent(s core.Sample) int {
	historyPct := diskLatencyPairHistoryPercent(s.DiskAwaitMS, s.DiskQueueDepth)
	for _, disk := range s.Disks {
		historyPct = maxKnownDiskPercent(historyPct, diskLatencyPairHistoryPercent(disk.AwaitMS, disk.QueueDepth))
	}

	return historyPct
}

func diskLatencyPairHistoryPercent(awaitMS, queueDepth float64) int {
	awaitPct := diskAwaitHistoryPercent(awaitMS)
	queuePct := diskQueueHistoryPercent(queueDepth)
	switch {
	case awaitPct < 0:
		return queuePct
	case queuePct < 0:
		return awaitPct
	default:
		return max(awaitPct, queuePct)
	}
}

// DiskSourceText returns the best display label for the root disk source.
func DiskSourceText(s core.Sample) string {
	if source := strings.TrimSpace(s.RootSource); source != "" {
		return source
	}
	if device := strings.TrimSpace(s.DiskDevice); device != "" {
		return "/dev/" + device
	}

	return "n/a"
}

// DiskFreeKiB returns available root filesystem space in KiB.
func DiskFreeKiB(s core.Sample) int64 {
	if s.RootTotalKiB <= 0 || s.RootUsedKiB < 0 {
		return -1
	}
	free := s.RootTotalKiB - s.RootUsedKiB
	if free < 0 {
		return 0
	}

	return free
}

// DiskFreePercent returns available root filesystem space as a percent.
func DiskFreePercent(s core.Sample) int {
	return PercentOf(DiskFreeKiB(s), s.RootTotalKiB)
}

// RootFilesystem returns the sampled root filesystem when present.
func RootFilesystem(s core.Sample) (core.FilesystemStat, bool) {
	for _, fs := range s.Filesystems {
		if strings.TrimSpace(fs.Mount) == "/" {
			return fs, true
		}
	}

	return core.EmptyFilesystemStat(), false
}

// ExtraFilesystems returns sampled filesystems except the root mount.
func ExtraFilesystems(s core.Sample) []core.FilesystemStat {
	rows := make([]core.FilesystemStat, 0, len(s.Filesystems))
	for _, fs := range s.Filesystems {
		if strings.TrimSpace(fs.Mount) == "/" {
			continue
		}
		rows = append(rows, fs)
	}

	return rows
}
