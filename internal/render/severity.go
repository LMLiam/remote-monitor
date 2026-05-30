package render

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
)

// UtilSeverity maps a utilization percent to a severity name.
func UtilSeverity(v int) string {
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= utilCriticalPercent:
		return severityCritical
	case v >= utilWarnPercent:
		return severityWarn
	case v >= utilOKPercent:
		return severityOK
	default:
		return severityInfo
	}
}

func memorySeverity(v int) string {
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

func availabilitySeverity(v int) string {
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v <= availabilityCriticalPercent:
		return severityCritical
	case v <= availabilityWarnPercent:
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

func diskUtilSeverity(v int) string {
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

func temperatureSeverity(v int) string {
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= temperatureCriticalPercent:
		return severityCritical
	case v >= temperatureWarnPercent:
		return severityWarn
	case v >= temperatureOKPercent:
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

func fanSeverity(v int) string {
	return UtilSeverity(v)
}

func clockSeverity(current, maxValue int) string {
	return UtilSeverity(metrics.ClockPercent(current, maxValue))
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
