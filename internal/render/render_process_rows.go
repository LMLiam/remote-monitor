package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strconv"
)

func buildTopProcessTableRows(state core.AppState) []ProcessListRowSpec {
	s := state.Current
	thresholds := thresholdsOrDefaults(state.Cfg.Thresholds)
	if len(s.TopProcesses) == 0 {
		return []ProcessListRowSpec{processRow(TextNone, ansi.Muted, "--", ansi.Muted, "idle", ansi.Muted, TextNA, ansi.Muted)}
	}

	rows := make([]ProcessListRowSpec, 0, len(s.TopProcesses))
	for _, proc := range s.TopProcesses {
		sev := UtilSeverity(clamp(proc.CPUPercent, percentMin, percentMax), thresholds)
		rows = append(rows, processRow(fallbackString(proc.Command, fmt.Sprintf("pid %d", proc.PID)), SeverityColor(sev), strconv.Itoa(proc.PID), ansi.Sand, processGaugeSuffix(proc.CPUPercent), SeverityColor(sev), formatMiBValue(proc.RSSMiB), ansi.Sand))
	}

	return rows
}

func buildGPUProcessTableRows(state core.AppState) []ProcessListRowSpec {
	s := state.Current
	thresholds := thresholdsOrDefaults(state.Cfg.Thresholds)
	if len(s.GPUs) == 0 {
		return []ProcessListRowSpec{processRow("unavailable", ansi.Muted, "--", ansi.Muted, "no gpu", ansi.Muted, TextNA, ansi.Muted)}
	}
	if len(s.GPUProcesses) == 0 {
		return []ProcessListRowSpec{processRow(TextNone, ansi.Muted, "--", ansi.Muted, "idle", ansi.Muted, TextNA, ansi.Muted)}
	}

	rows := make([]ProcessListRowSpec, 0, len(s.GPUProcesses))
	for _, proc := range s.GPUProcesses {
		vramPct := gpuProcessVRAMPercent(s, proc)
		sev := vramSeverity(vramPct, thresholds)
		if vramPct < 0 {
			sev = severityNeutral
		}
		rows = append(rows, processRow(fallbackString(proc.Command, fmt.Sprintf("pid %d", proc.PID)), SeverityColor(sev), strconv.Itoa(proc.PID), ansi.Sand, gpuProcessLocationText(s, proc), SeverityColor(sev), formatMiBValue(proc.UsedMemMiB), SeverityColor(sev)))
	}

	return rows
}
