package metrics_test

import (
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
)

func TestOverallGPUUtilVRAMAndTemperatureSummaries(t *testing.T) {
	t.Parallel()

	empty := core.EmptySample()
	if got := metrics.OverallGPUUtil(empty); got != 0 {
		t.Fatalf("OverallGPUUtil empty = %d, want 0", got)
	}
	if got := metrics.OverallVRAMPct(empty); got != 0 {
		t.Fatalf("OverallVRAMPct empty = %d, want 0", got)
	}
	if got := metrics.OverallTempValue(empty); got != -1 {
		t.Fatalf("OverallTempValue empty = %d, want -1", got)
	}
	if got := metrics.OverallTempPct(empty); got != 0 {
		t.Fatalf("OverallTempPct empty = %d, want 0", got)
	}

	smp := core.EmptySample()
	smp.GPUs = []core.GPUStat{
		testGPUStat(0, 7, 4204, 12288, 64, 0, 0),
		testGPUStat(1, 82, 2003, 12288, 55, 0, 0),
		testGPUStat(2, 10, -1, 12288, 150, 0, 0),
	}

	if got := metrics.OverallGPUUtil(smp); got != 82 {
		t.Fatalf("OverallGPUUtil = %d, want 82", got)
	}
	if got := metrics.OverallVRAMPct(smp); got != 34 {
		t.Fatalf("OverallVRAMPct = %d, want 34", got)
	}
	if got := metrics.OverallTempValue(smp); got != 150 {
		t.Fatalf("OverallTempValue = %d, want 150", got)
	}
	if got := metrics.OverallTempPct(smp); got != 100 {
		t.Fatalf("OverallTempPct = %d, want 100", got)
	}
}

func TestOverallPowerSummariesIgnoreUnknownAndAggregateKnown(t *testing.T) {
	t.Parallel()

	smp := core.EmptySample()
	smp.GPUs = []core.GPUStat{
		testGPUStat(0, 0, 0, 0, -1, 50, 100),
		testGPUStat(1, 0, 0, 0, -1, -1, 0),
		testGPUStat(2, 0, 0, 0, -1, 25, 50),
	}

	if got := metrics.OverallPowerDraw(smp); got != 75 {
		t.Fatalf("OverallPowerDraw = %.1f, want 75.0", got)
	}
	if got := metrics.OverallPowerLimit(smp); got != 150 {
		t.Fatalf("OverallPowerLimit = %.1f, want 150.0", got)
	}
	if got := metrics.OverallPowerPct(smp); got != 50 {
		t.Fatalf("OverallPowerPct = %d, want 50", got)
	}

	noLimit := core.EmptySample()
	noLimit.GPUs = []core.GPUStat{
		testGPUStat(0, 0, 0, 0, -1, 25, 0),
	}
	if got := metrics.OverallPowerPct(noLimit); got != 0 {
		t.Fatalf("OverallPowerPct without limit = %d, want 0", got)
	}
}

func TestRAMAvailablePercentUsesPercentOf(t *testing.T) {
	t.Parallel()

	smp := core.EmptySample()
	smp.RAMAvailableMiB = 25
	smp.RAMTotalMiB = 100
	if got := metrics.RAMAvailablePercent(smp); got != 25 {
		t.Fatalf("RAMAvailablePercent = %d, want 25", got)
	}

	smp.RAMTotalMiB = 0
	if got := metrics.RAMAvailablePercent(smp); got != 0 {
		t.Fatalf("RAMAvailablePercent with zero total = %d, want 0", got)
	}
}

func testGPUStat(index, util int, memUsed, memTotal int64, temp int, powerDraw, powerLimit float64) core.GPUStat {
	return core.GPUStat{
		Index:            index,
		UUID:             "",
		Name:             "",
		Util:             util,
		MemUtil:          0,
		EncoderUtil:      0,
		DecoderUtil:      0,
		MemUsed:          memUsed,
		MemTotal:         memTotal,
		Temp:             temp,
		PowerDraw:        powerDraw,
		PowerLimit:       powerLimit,
		Fan:              0,
		SMClock:          0,
		MaxSMClock:       0,
		MemClock:         0,
		MaxMemClock:      0,
		GraphicsClock:    0,
		VideoClock:       0,
		PCIeGenCurrent:   0,
		PCIeGenMax:       0,
		PCIeWidthCurrent: 0,
		PCIeWidthMax:     0,
		ThrottleReasons:  "",
		PState:           "",
	}
}
