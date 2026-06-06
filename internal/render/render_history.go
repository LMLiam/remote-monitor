package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

const (
	historyBoxPadding     = 4
	historyColumnGap      = 2
	historyColumnCount    = 2
	historySuffixMinWidth = 5
	historyGraphMinWidth  = 6
)

type historyMetricSpec struct {
	label      string
	suffix     string
	color      string
	metricKind string
	values     []int
	values64   []int64
	isRate     bool
}

// HistoryBox renders the rolling history panel beneath the live tables.
func HistoryBox(state core.AppState, totalWidth int) string {
	innerWidth := totalWidth - historyBoxPadding
	thresholds := thresholdsOrDefaults(state.Cfg.Thresholds)
	var b strings.Builder
	b.WriteString(boxRule("╭", "╮", innerWidth))
	b.WriteString(boxLine("History (newest on the right, rolling samples)", innerWidth, ansi.TitleColor))
	b.WriteString(boxRule("├", "┤", innerWidth))
	specs := historyMetricSpecs(state)

	leftWidth := (innerWidth - historyColumnGap) / historyColumnCount
	rightWidth := innerWidth - historyColumnGap - leftWidth
	for i := 0; i < len(specs); i += historyColumnCount {
		left := historyMetricCell(specs[i], leftWidth, thresholds)
		right := strings.Repeat(" ", rightWidth)
		if i+1 < len(specs) {
			right = ansi.Pad(historyMetricCell(specs[i+1], rightWidth, thresholds), rightWidth)
		}
		row := ansi.Pad(left, leftWidth) + "  " + right
		b.WriteString(boxLine(row, innerWidth, ""))
		if i+historyColumnCount < len(specs) {
			b.WriteString(boxLine("", innerWidth, ""))
		}
	}
	b.WriteString(boxRule("╰", "╯", innerWidth))

	return b.String()
}

func historyMetricSpecs(state core.AppState) []historyMetricSpec {
	current := state.Current
	thresholds := thresholdsOrDefaults(state.Cfg.Thresholds)
	cpuUtil := lastOrZero(state.CPUHistory)
	ramUtil := lastOrZero(state.RAMHistory)
	ramAvailable := metrics.RAMAvailablePercent(current)
	diskUtil := lastOrZero(state.DiskHistory)
	diskLatencySeverity := mergeSeverity(
		diskAwaitSeverity(current.DiskAwaitMS),
		diskQueueSeverity(current.DiskQueueDepth),
	)
	netIssues := lastOrZero(state.NetIssueHistory)
	gpuUtil := lastOrZero(state.GPUHistory)
	vramUtil := lastOrZero(state.VRAMHistory)
	gpuTemp := metrics.OverallTempValue(current)
	powerDraw := metrics.OverallPowerDraw(current)
	powerLimit := metrics.OverallPowerLimit(current)

	return []historyMetricSpec{
		historyMetric("CPU", state.CPUHistory, percentDisplay(cpuUtil), SeverityColor(UtilSeverity(cpuUtil, thresholds)), "util"),
		historyMetric("CPU FREQ", state.CPUFreqHistory, formatClockValue(current.CPUFreqMHz), ansi.Lav, "clock"),
		historyMetric("CPU TEMP", state.CPUTempHistory, tempDisplay(current.CPUTempC), SeverityColor(cpuTemperatureSeverity(current.CPUTempC, thresholds)), "cpu-temperature"),
		historyMetric("RAM", state.RAMHistory, percentDisplay(ramUtil), SeverityColor(memorySeverity(ramUtil, thresholds)), "memory"),
		historyMetric("RAM AVAIL", state.RAMAvailHistory, percentDisplay(ramAvailable), SeverityColor(availabilitySeverity(ramAvailable, thresholds)), "availability"),
		historyMetric("DISK", state.DiskHistory, percentDisplay(diskUtil), SeverityColor(diskUtilSeverity(diskUtil, thresholds)), "disk"),
		historyMetric("DISK LAT", state.DiskLatencyHistory, FormatMillisValue(current.DiskAwaitMS), SeverityColor(diskLatencySeverity), "latency"),
		historyRateMetric("NET RX", state.NetRXHistory, formatBps(lastOrZero64(state.NetRXHistory)), ansi.Blue, "net-rx"),
		historyRateMetric("NET TX", state.NetTXHistory, formatBps(lastOrZero64(state.NetTXHistory)), ansi.Cyan, "net-tx"),
		historyMetric("NET ISSUES", state.NetIssueHistory, netIssueSummary(current), SeverityColor(netIssueSeverity(netIssues)), "issues"),
		historyMetric(LabelGPU, state.GPUHistory, percentDisplay(gpuUtil), SeverityColor(UtilSeverity(gpuUtil, thresholds)), "util"),
		historyMetric("VRAM", state.VRAMHistory, percentDisplay(vramUtil), SeverityColor(vramSeverity(vramUtil, thresholds)), "vram"),
		historyMetric("GPU TEMP", state.TempHistory, tempDisplay(gpuTemp), SeverityColor(temperatureSeverity(gpuTemp, thresholds)), "gpu-temperature"),
		historyMetric("POWER", state.PowerHistory, formatPowerValue(powerDraw), SeverityColor(powerSeverity(powerDraw, powerLimit)), "power"),
	}
}

func historyMetric(label string, values []int, suffix, color, metricKind string) historyMetricSpec {
	return historyMetricSpec{
		label:      label,
		suffix:     suffix,
		color:      color,
		metricKind: metricKind,
		values:     values,
		values64:   nil,
		isRate:     false,
	}
}

func historyRateMetric(label string, values []int64, suffix, color, metricKind string) historyMetricSpec {
	return historyMetricSpec{
		label:      label,
		suffix:     suffix,
		color:      color,
		metricKind: metricKind,
		values:     nil,
		values64:   values,
		isRate:     true,
	}
}

func historyMetricCell(spec historyMetricSpec, width int, thresholds core.Thresholds) string {
	if width <= 0 {
		return ""
	}
	label := inlineChip(spec.label, ansi.PanelAltBg)
	suffixWidth := max(historySuffixMinWidth, ansi.VisibleLen(spec.suffix))
	graphWidth := max(historyGraphMinWidth, width-ansi.VisibleLen(label)-suffixWidth-historyColumnGap)

	var graph string
	if spec.isRate {
		graph = sparklineScaled64(spec.values64, graphWidth, spec.color)
	} else {
		graph = sparklineColored(spec.values, graphWidth, spec.metricKind, thresholds)
	}
	suffix := fillBlock(ansi.RightJustify(spec.suffix, suffixWidth), suffixWidth, spec.color, ansi.PanelAltBg, false)

	return fmt.Sprintf("%s %s %s", label, graph, suffix)
}
