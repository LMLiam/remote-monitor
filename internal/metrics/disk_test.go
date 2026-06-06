package metrics_test

import (
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
)

func TestDiskUtilPercentUsesBusiestKnownDisk(t *testing.T) {
	t.Parallel()

	smp := core.EmptySample()
	smp.DiskUtil = 12
	smp.Disks = []core.DiskStat{
		testDiskStat("sda", 3, 1.2, 0.1),
		testDiskStat("nvme0n1", 63, 2.4, 0.4),
	}

	if got := metrics.DiskUtilPercent(smp); got != 63 {
		t.Fatalf("DiskUtilPercent = %d, want 63", got)
	}
}

func TestDiskLatencyHistoryPercentUsesBusiestKnownDisk(t *testing.T) {
	t.Parallel()

	smp := core.EmptySample()
	smp.DiskAwaitMS = 1
	smp.DiskQueueDepth = 0.1
	smp.Disks = []core.DiskStat{
		testDiskStat("sda", 3, 1.2, 0.1),
		testDiskStat("nvme0n1", 63, 25, 0.4),
	}

	if got := metrics.DiskLatencyHistoryPercent(smp); got != 50 {
		t.Fatalf("DiskLatencyHistoryPercent = %d, want 50", got)
	}
}

func testDiskStat(device string, util int, awaitMS, queueDepth float64) core.DiskStat {
	return core.DiskStat{
		Device:            device,
		ReadBps:           0,
		WriteBps:          0,
		ReadMergedPerSec:  0,
		WriteMergedPerSec: 0,
		Util:              util,
		AwaitMS:           awaitMS,
		QueueDepth:        queueDepth,
		Inflight:          0,
	}
}
