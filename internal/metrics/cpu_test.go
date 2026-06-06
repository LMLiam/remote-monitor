package metrics_test

import (
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
)

func TestCPUActiveCoreCountUsesStrictThreshold(t *testing.T) {
	t.Parallel()

	cores := []core.CPUCore{
		{Index: 0, Percent: 0},
		{Index: 1, Percent: 50},
		{Index: 2, Percent: 51},
		{Index: 3, Percent: 100},
	}

	if got := metrics.CPUActiveCoreCount(cores, 50); got != 2 {
		t.Fatalf("CPUActiveCoreCount threshold 50 = %d, want 2", got)
	}
	if got := metrics.CPUActiveCoreCount(cores, -1); got != 3 {
		t.Fatalf("CPUActiveCoreCount negative threshold = %d, want 3", got)
	}
}

func TestCPUAveragePercentAveragesOnlyActiveCores(t *testing.T) {
	t.Parallel()

	if got := metrics.CPUAveragePercent(nil); got != -1 {
		t.Fatalf("CPUAveragePercent empty = %d, want -1", got)
	}
	if got := metrics.CPUAveragePercent([]core.CPUCore{{Index: 0, Percent: 0}}); got != 0 {
		t.Fatalf("CPUAveragePercent idle = %d, want 0", got)
	}

	cores := []core.CPUCore{
		{Index: 0, Percent: 0},
		{Index: 1, Percent: 25},
		{Index: 2, Percent: 50},
	}
	if got := metrics.CPUAveragePercent(cores); got != 38 {
		t.Fatalf("CPUAveragePercent active average = %d, want 38", got)
	}
}

func TestCPUPeakCorePrefersLowestIndexOnTie(t *testing.T) {
	t.Parallel()

	if got := metrics.CPUPeakCore(nil); got != (core.CPUCore{Index: -1, Percent: -1}) {
		t.Fatalf("CPUPeakCore empty = %#v, want sentinel", got)
	}

	cores := []core.CPUCore{
		{Index: 3, Percent: 70},
		{Index: 1, Percent: 70},
		{Index: 2, Percent: 40},
	}
	if got := metrics.CPUPeakCore(cores); got != (core.CPUCore{Index: 1, Percent: 70}) {
		t.Fatalf("CPUPeakCore tie = %#v, want core 1", got)
	}
}

func TestCPUImbalancePercentHandlesSentinelsAndClamp(t *testing.T) {
	t.Parallel()

	if got := metrics.CPUImbalancePercent(nil); got != -1 {
		t.Fatalf("CPUImbalancePercent empty = %d, want -1", got)
	}

	cores := []core.CPUCore{
		{Index: 0, Percent: 20},
		{Index: 1, Percent: 100},
	}
	if got := metrics.CPUImbalancePercent(cores); got != 40 {
		t.Fatalf("CPUImbalancePercent mixed cores = %d, want 40", got)
	}

	overRange := []core.CPUCore{
		{Index: 0, Percent: 0},
		{Index: 1, Percent: 250},
	}
	if got := metrics.CPUImbalancePercent(overRange); got != 100 {
		t.Fatalf("CPUImbalancePercent over-range core = %d, want 100", got)
	}
}
