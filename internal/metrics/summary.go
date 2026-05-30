package metrics

import core "github.com/lmliam/remote-monitor/internal/core"

// OverallGPUUtil returns peak utilization across sampled GPUs.
func OverallGPUUtil(s core.Sample) int {
	best := 0
	for _, gpu := range s.GPUs {
		if gpu.Util > best {
			best = gpu.Util
		}
	}

	return best
}

// OverallVRAMPct returns aggregate VRAM utilization across sampled GPUs.
func OverallVRAMPct(s core.Sample) int {
	best := 0
	for _, gpu := range s.GPUs {
		pct := PercentOf(gpu.MemUsed, gpu.MemTotal)
		if pct > best {
			best = pct
		}
	}

	return best
}

// OverallTempValue returns the hottest sampled GPU temperature.
func OverallTempValue(s core.Sample) int {
	best := -1
	for _, gpu := range s.GPUs {
		if gpu.Temp > best {
			best = gpu.Temp
		}
	}

	return best
}

// OverallTempPct returns the hottest sampled GPU temperature as a percent-like value.
func OverallTempPct(s core.Sample) int {
	return TemperaturePercent(OverallTempValue(s))
}

// OverallPowerDraw returns total positive GPU power draw.
func OverallPowerDraw(s core.Sample) float64 {
	total := 0.0
	for _, gpu := range s.GPUs {
		if gpu.PowerDraw > 0 {
			total += gpu.PowerDraw
		}
	}

	return total
}

// OverallPowerLimit returns total positive GPU power limit.
func OverallPowerLimit(s core.Sample) float64 {
	total := 0.0
	for _, gpu := range s.GPUs {
		if gpu.PowerLimit > 0 {
			total += gpu.PowerLimit
		}
	}

	return total
}

// OverallPowerPct returns aggregate GPU power draw as a percent of aggregate limits.
func OverallPowerPct(s core.Sample) int {
	return PowerPercent(OverallPowerDraw(s), OverallPowerLimit(s))
}

// RAMAvailablePercent returns available RAM as a percent of total RAM.
func RAMAvailablePercent(s core.Sample) int {
	return PercentOf(s.RAMAvailableMiB, s.RAMTotalMiB)
}
