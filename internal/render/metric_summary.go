package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"strings"
)

func cpuDisplayName(s core.Sample) string {
	return strings.TrimSpace(s.CPUName)
}

func gpuDisplayName(s core.Sample) string {
	if len(s.GPUs) == 0 {
		return ""
	}
	name := strings.TrimSpace(s.GPUs[0].Name)
	if name == "" {
		name = fmt.Sprintf("GPU %d", s.GPUs[0].Index)
	}
	if len(s.GPUs) == 1 {
		return name
	}

	return fmt.Sprintf("%s +%d", name, len(s.GPUs)-1)
}

func cpuTableTitle(s core.Sample) string {
	name := cpuDisplayName(s)
	if name == "" {
		return "CPU"
	}

	return "CPU • " + name
}

func gpuTableTitle(s core.Sample) string {
	name := gpuDisplayName(s)
	if name == "" {
		return LabelGPU
	}

	return "GPU • " + name
}
