package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"strings"
)

func pStateMeaning(pstate string) string {
	switch strings.ToUpper(strings.TrimSpace(pstate)) {
	case "P0":
		return "max perf"
	case "P1":
		return "boost"
	case "P2":
		return "active 3D"
	case "P3":
		return "balanced"
	case "P4":
		return "light load"
	case "P5":
		return "idle / cool"
	case "P8", "P10", "P12":
		return "deep idle"
	case "":
		return "power state"
	default:
		return "power state"
	}
}

func processGaugeSuffix(cpuPercent int) string {
	if cpuPercent < 0 {
		return TextNA
	}

	return fmt.Sprintf("%d%%", cpuPercent)
}

func gpuProcessIndex(s core.Sample, proc core.GPUProcessStat) int {
	for _, gpu := range s.GPUs {
		if strings.TrimSpace(gpu.UUID) != "" && gpu.UUID == proc.GPUUUID {
			return gpu.Index
		}
	}
	if len(s.GPUs) == 1 {
		return s.GPUs[0].Index
	}

	return -1
}

func gpuProcessVRAMPercent(s core.Sample, proc core.GPUProcessStat) int {
	for _, gpu := range s.GPUs {
		if strings.TrimSpace(gpu.UUID) != "" && gpu.UUID == proc.GPUUUID {
			return metrics.PercentOf(proc.UsedMemMiB, gpu.MemTotal)
		}
	}
	if len(s.GPUs) == 1 {
		return metrics.PercentOf(proc.UsedMemMiB, s.GPUs[0].MemTotal)
	}

	return -1
}

func gpuProcessLocationText(s core.Sample, proc core.GPUProcessStat) string {
	if idx := gpuProcessIndex(s, proc); idx >= 0 {
		return fmt.Sprintf("GPU%d", idx)
	}

	return "GPU?"
}
