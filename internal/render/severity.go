package render

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
)

// UtilSeverity maps a utilization percent to a severity name.
func UtilSeverity(v int, thresholds core.Thresholds) string {
	thresholds = thresholdsOrDefaults(thresholds)
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= thresholds.CPUCriticalPercent:
		return severityCritical
	case v >= utilWarnPercent:
		return severityWarn
	case v >= utilOKPercent:
		return severityOK
	default:
		return severityInfo
	}
}

func memorySeverity(v int, _ core.Thresholds) string {
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= memoryCriticalPercent:
		return severityCritical
	case v >= memoryWarnPercent:
		return severityWarn
	case v >= memoryOKPercent:
		return severityOK
	default:
		return severityInfo
	}
}

func availabilitySeverity(v int, thresholds core.Thresholds) string {
	thresholds = thresholdsOrDefaults(thresholds)
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v <= thresholds.RAMCriticalAvailablePercent:
		return severityCritical
	case v <= thresholds.RAMWarnAvailablePercent:
		return severityWarn
	case v <= availabilityInfoPercent:
		return severityInfo
	default:
		return severityOK
	}
}

func psiSeverity(v float64) string {
	switch {
	case v < 0:
		return severityNeutral
	case v >= psiCriticalPercent:
		return severityCritical
	case v >= psiWarnPercent:
		return severityWarn
	case v >= 1:
		return severityOK
	default:
		return severityInfo
	}
}

func diskUtilSeverity(v int, _ core.Thresholds) string {
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= diskUtilCriticalPercent:
		return severityCritical
	case v >= diskUtilWarnPercent:
		return severityWarn
	case v >= diskUtilOKPercent:
		return severityOK
	default:
		return severityInfo
	}
}

func diskUsageSeverity(v int, thresholds core.Thresholds) string {
	thresholds = thresholdsOrDefaults(thresholds)
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= thresholds.DiskCriticalPercent:
		return severityCritical
	case v >= thresholds.DiskWarnPercent:
		return severityWarn
	case v >= diskUtilOKPercent:
		return severityOK
	default:
		return severityInfo
	}
}

func temperatureSeverity(v int, thresholds core.Thresholds) string {
	thresholds = thresholdsOrDefaults(thresholds)
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= thresholds.GPUCriticalTemp:
		return severityCritical
	case v >= thresholds.GPUWarnTemp:
		return severityWarn
	case v >= temperatureOKPercent:
		return severityOK
	default:
		return severityInfo
	}
}

func cpuTemperatureSeverity(v int, thresholds core.Thresholds) string {
	thresholds = thresholdsOrDefaults(thresholds)
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= thresholds.CPUCriticalTemp:
		return severityCritical
	case v >= thresholds.CPUWarnTemp:
		return severityWarn
	case v >= temperatureOKPercent:
		return severityOK
	default:
		return severityInfo
	}
}

func vramSeverity(v int, thresholds core.Thresholds) string {
	thresholds = thresholdsOrDefaults(thresholds)
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= thresholds.VRAMCriticalPercent:
		return severityCritical
	case v >= thresholds.VRAMWarnPercent:
		return severityWarn
	case v >= memoryOKPercent:
		return severityOK
	default:
		return severityInfo
	}
}

func powerSeverity(draw, limit float64) string {
	if draw <= 0 || limit <= 0 {
		return severityNeutral
	}
	pct := (draw / limit) * percentScale
	switch {
	case pct >= powerCriticalPercent:
		return severityCritical
	case pct >= powerWarnPercent:
		return severityWarn
	case pct >= powerOKPercent:
		return severityOK
	default:
		return severityInfo
	}
}

func fanSeverity(v int, thresholds core.Thresholds) string {
	return UtilSeverity(v, thresholds)
}

func clockSeverity(current, maxValue int, thresholds core.Thresholds) string {
	return UtilSeverity(metrics.ClockPercent(current, maxValue), thresholds)
}

func thresholdsOrDefaults(thresholds core.Thresholds) core.Thresholds {
	var zeroThresholds core.Thresholds
	if thresholds == zeroThresholds {
		return core.DefaultThresholds()
	}

	return thresholds
}

func mergeSeverity(a, b string) string {
	if severityRank(b) > severityRank(a) {
		return b
	}

	return a
}

// SeverityColor returns the foreground ANSI color for a severity name.
func SeverityColor(severity string) string {
	switch severity {
	case severityCritical, severityHot:
		return ansi.Red
	case severityWarn:
		return ansi.Yellow
	case severityInfo:
		return ansi.Blue
	case severityOK:
		return ansi.Green
	default:
		return ansi.Muted
	}
}

func severityBackground(severity string) string {
	switch severity {
	case severityCritical, severityHot:
		return ansi.RedBg
	case severityWarn:
		return ansi.YellowBg
	case severityInfo:
		return ansi.BlueBg
	case severityOK:
		return ansi.GreenBg
	default:
		return ansi.MutedBg
	}
}

func severityRank(severity string) int {
	switch severity {
	case severityCritical, severityHot:
		return severityRankCritical
	case severityWarn:
		return severityRankWarn
	case severityInfo:
		return 1
	default:
		return 0
	}
}

func statusColor(status string) string {
	switch status {
	case core.StatusLive:
		return ansi.Green
	case core.StatusStale:
		return ansi.Yellow
	case core.StatusDisconnected:
		return ansi.Red
	default:
		return ansi.Cyan
	}
}

func statusBackground(status string) string {
	switch status {
	case core.StatusLive:
		return ansi.GreenBg
	case core.StatusStale:
		return ansi.YellowBg
	case core.StatusDisconnected:
		return ansi.RedBg
	default:
		return ansi.CyanBg
	}
}
