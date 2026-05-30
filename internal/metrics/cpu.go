package metrics

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"math"
)

// CPUActiveCoreCount counts cores whose utilization is above threshold.
func CPUActiveCoreCount(cores []core.CPUCore, threshold int) int {
	if threshold < 0 {
		threshold = 0
	}
	active := 0
	for _, core := range cores {
		if core.Percent > threshold {
			active++
		}
	}

	return active
}

// CPUAveragePercent returns the average utilization across active cores.
func CPUAveragePercent(cores []core.CPUCore) int {
	if len(cores) == 0 {
		return -1
	}
	total := 0
	active := 0
	for _, core := range cores {
		if core.Percent > 0 {
			total += core.Percent
			active++
		}
	}
	if active == 0 {
		return 0
	}

	return Clamp(int(math.Round(float64(total)/float64(active))), percentMin, percentMax)
}

// CPUPeakCore returns the busiest core, preferring the lowest index on ties.
func CPUPeakCore(cores []core.CPUCore) core.CPUCore {
	best := core.CPUCore{Index: -1, Percent: -1}
	for _, core := range cores {
		if core.Percent > best.Percent || (core.Percent == best.Percent && (best.Index < 0 || core.Index < best.Index)) {
			best = core
		}
	}

	return best
}

// CPUImbalancePercent returns the gap between the peak core and active average.
func CPUImbalancePercent(cores []core.CPUCore) int {
	peak := CPUPeakCore(cores)
	avg := CPUAveragePercent(cores)
	if peak.Percent < 0 || avg < 0 {
		return -1
	}

	return Clamp(peak.Percent-avg, percentMin, percentMax)
}
