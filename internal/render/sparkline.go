package render

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"math"
	"strings"
)

func sparkline(values []int, width int) string {
	if width <= 0 {
		return ""
	}
	if len(values) == 0 {
		return strings.Repeat("·", width)
	}
	start := max(0, len(values)-width)
	window := values[start:]
	if len(window) < width {
		padded := make([]int, width)
		copy(padded[width-len(window):], window)
		window = padded
	}
	levels := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	var b strings.Builder
	b.Grow(width * sparklineRuneByteBudget)
	for _, v := range window {
		pct := clamp(v, percentMin, percentMax)
		idx := int(math.Round(float64(pct) / percentScale * float64(len(levels)-1)))
		b.WriteRune(levels[idx])
	}

	return b.String()
}

func sparklineColored(values []int, width int, metricKind string, thresholds core.Thresholds) string {
	thresholds = thresholdsOrDefaults(thresholds)
	if width <= 0 {
		return ""
	}
	if len(values) == 0 {
		return ansi.Colorize(ansi.Dim, strings.Repeat(" ", width))
	}

	start := max(0, len(values)-width)
	window := values[start:]
	var b strings.Builder
	if len(window) < width {
		b.WriteString(ansi.Colorize(ansi.Dim, strings.Repeat(" ", width-len(window))))
	}
	for _, v := range window {
		glyph := sparkline([]int{v}, 1)
		color := historyColor(metricKind, v, thresholds)
		b.WriteString(ansi.Colorize(color, glyph))
	}

	return b.String()
}

func historyColor(metricKind string, value int, thresholds core.Thresholds) string {
	thresholds = thresholdsOrDefaults(thresholds)
	if value < 0 {
		return ansi.Dim
	}
	switch metricKind {
	case "util":
		return SeverityColor(UtilSeverity(value, thresholds))
	case "clock":
		return ansi.Lav
	case "memory":
		return SeverityColor(memorySeverity(value, thresholds))
	case "availability":
		return SeverityColor(availabilitySeverity(value, thresholds))
	case "cpu-temperature":
		return SeverityColor(cpuTemperatureSeverity(value, thresholds))
	case "gpu-temperature":
		return SeverityColor(temperatureSeverity(value, thresholds))
	case "vram":
		return SeverityColor(vramSeverity(value, thresholds))
	case "disk":
		return SeverityColor(diskUtilSeverity(value, thresholds))
	case "latency":
		return SeverityColor(diskLatencyHistorySeverity(value))
	case "issues":
		return SeverityColor(netIssueSeverity(value))
	case "power":
		return SeverityColor(powerSeverity(float64(value), percentMax))
	case "net-rx":
		return ansi.Blue
	case "net-tx":
		return ansi.Cyan
	default:
		return ansi.Muted
	}
}

func sparklineScaled64(values []int64, width int, color string) string {
	if width <= 0 {
		return ""
	}
	if len(values) == 0 {
		return ansi.Colorize(ansi.Dim, strings.Repeat(" ", width))
	}

	start := max(0, len(values)-width)
	window := values[start:]
	var peak int64
	for _, v := range window {
		if v > peak {
			peak = v
		}
	}
	if peak <= 0 {
		return ansi.Colorize(ansi.Dim, strings.Repeat("▁", width))
	}

	scaled := make([]int, 0, len(window))
	for _, v := range window {
		scaled = append(scaled, metrics.PercentOf(v, peak))
	}

	var b strings.Builder
	if len(window) < width {
		b.WriteString(ansi.Colorize(ansi.Dim, strings.Repeat(" ", width-len(window))))
	}
	for _, glyph := range sparkline(scaled, len(scaled)) {
		b.WriteString(ansi.Colorize(color, string(glyph)))
	}

	return b.String()
}
