package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"sort"
	"strings"
)

// BuildCPURows builds the CPU section rows for the dashboard table layout.
func BuildCPURows(state core.AppState, valueWidth, activityWidth int, condensed bool) []TableRowSpec {
	s := state.Current
	rows := make([]TableRowSpec, 0, cpuRowsCapacity)
	postMapRows := make([]TableRowSpec, 0, cpuPostMapRowsCap)

	cpuSeverity := UtilSeverity(s.CPUPercent)
	rows = append(rows, TableFullRow("CPU Overall", SeverityColor(cpuSeverity), fmt.Sprintf("%s • load %.2f", percentDisplay(s.CPUPercent), s.Load1), SeverityColor(cpuSeverity), "", "", "", gaugeCell(s.CPUPercent, activityWidth, cpuSeverity)))

	hot := append([]core.CPUCore(nil), s.CPUCoresUsage...)
	sort.Slice(hot, func(i, j int) bool {
		if hot[i].Percent == hot[j].Percent {
			return hot[i].Index < hot[j].Index
		}

		return hot[i].Percent > hot[j].Percent
	})

	if len(s.CPUCoresUsage) > 0 {
		const activeThreshold = 1
		activeCount := metrics.CPUActiveCoreCount(s.CPUCoresUsage, activeThreshold)
		activePct := metrics.PercentOf(int64(activeCount), int64(len(s.CPUCoresUsage)))
		peak := metrics.CPUPeakCore(s.CPUCoresUsage)
		avg := metrics.CPUAveragePercent(s.CPUCoresUsage)
		imbalance := metrics.CPUImbalancePercent(s.CPUCoresUsage)

		rows = append(rows,
			TableFullRow(LabelCPUActive, ansi.Cyan, fmt.Sprintf("%d / %d cores >%d%%", activeCount, len(s.CPUCoresUsage), activeThreshold), ansi.Cyan, "", "", "", gaugeBarCell(activePct, activityWidth, ansi.Cyan, percentDisplay(activePct))),
		)

		postMapRows = append(postMapRows, TableFullRow(LabelCPUImbalance, SeverityColor(UtilSeverity(imbalance)), fmt.Sprintf("hot %d %s • avg active %s", peak.Index, percentDisplay(peak.Percent), percentDisplay(avg)), SeverityColor(UtilSeverity(imbalance)), "", "", "", gaugeBarCell(imbalance, activityWidth, SeverityColor(UtilSeverity(imbalance)), percentDisplay(imbalance))))
	}

	cpuBreakdowns := []struct {
		label   string
		percent int
	}{{label: LabelCPUUser, percent: s.CPUUserPercent}, {label: "CPU System", percent: s.CPUSystemPercent}}
	if !condensed {
		cpuBreakdowns = append(cpuBreakdowns,
			struct {
				label   string
				percent int
			}{label: "CPU IOWait", percent: s.CPUIOWaitPercent},
			struct {
				label   string
				percent int
			}{label: "CPU Steal", percent: s.CPUStealPercent},
		)
	}
	for _, breakdown := range cpuBreakdowns {
		if breakdown.percent < 0 {
			continue
		}
		sev := UtilSeverity(breakdown.percent)
		rows = append(rows, TableFullRow(breakdown.label, SeverityColor(sev), percentDisplay(breakdown.percent), SeverityColor(sev), "", "", "", gaugeCell(breakdown.percent, activityWidth, sev)))
	}

	if s.CPUPressureSomeAvg10 >= 0 || s.CPUPressureFullAvg10 >= 0 {
		pressurePeak := s.CPUPressureSomeAvg10
		if s.CPUPressureFullAvg10 > pressurePeak {
			pressurePeak = s.CPUPressureFullAvg10
		}
		pressureSeverity := mergeSeverity(psiSeverity(s.CPUPressureSomeAvg10), psiSeverity(s.CPUPressureFullAvg10))
		rows = append(rows, TableFullRow(LabelCPUPSI, SeverityColor(pressureSeverity), "some "+formatPSIValue(s.CPUPressureSomeAvg10), SeverityColor(pressureSeverity), "", "", "", gaugeBarCell(psiPercent(pressurePeak), activityWidth, SeverityColor(pressureSeverity), "full "+formatPSIValue(s.CPUPressureFullAvg10))))
	}

	if s.CPUFreqMHz >= 0 || s.CPUMaxFreqMHz > 0 {
		freqPct := metrics.ClockPercent(s.CPUFreqMHz, s.CPUMaxFreqMHz)
		freqSeverity := clockSeverity(s.CPUFreqMHz, s.CPUMaxFreqMHz)
		freqRow := TableFullRow(LabelCPUFreq, ansi.Lav, formatClockValue(s.CPUFreqMHz), SeverityColor(freqSeverity), "", "max n/a", ansi.Muted, "")
		if freqPct >= 0 {
			freqRow.ActivityCell = gaugeBarCell(freqPct, activityWidth, SeverityColor(freqSeverity), percentDisplay(freqPct))
			freqRow.ActivityText = ""
			freqRow.ActivityColor = ""
		} else if s.CPUMaxFreqMHz > 0 {
			freqRow.ActivityText = fmt.Sprintf("max %d MHz", s.CPUMaxFreqMHz)
			freqRow.ActivityColor = SeverityColor(freqSeverity)
		}
		rows = append(rows, freqRow)
	}

	if s.CPUTempC >= 0 {
		tempSeverity := temperatureSeverity(s.CPUTempC)
		rows = append(rows, TableFullRow(LabelCPUTemp, ansi.Amber, tempDisplay(s.CPUTempC), SeverityColor(tempSeverity), "", "", "", gaugeBarCell(metrics.TemperaturePercent(s.CPUTempC), activityWidth, SeverityColor(tempSeverity), tempDisplay(s.CPUTempC))))
	}

	if !condensed {
		for _, core := range hot[:min(hotCorePreviewLimit, len(hot))] {
			sev := UtilSeverity(core.Percent)
			postMapRows = append(postMapRows, TableFullRow(fmt.Sprintf("CPU Hot %d", core.Index), SeverityColor(sev), percentDisplay(core.Percent), SeverityColor(sev), "", "", "", gaugeCell(core.Percent, activityWidth, sev)))
		}
	}

	if len(s.CPUCoresUsage) > 0 {
		peak := hot[0].Percent
		rows = append(rows,
			tableDividerRow(),
			TableFullRow(LabelCPUMap, ansi.Sand, fmt.Sprintf("%d cores • peak %s", len(s.CPUCoresUsage), percentDisplay(peak)), ansi.Sand, "", "", "", CoreHeatmapCell(s.CPUCoresUsage, activityWidth)),
		)
		if len(postMapRows) > 0 {
			rows = append(rows, tableDividerRow())
			rows = append(rows, postMapRows...)
		}
	}

	_ = valueWidth

	return rows
}

// CoreHeatmapCell renders per-core utilization as a fixed-width heatmap cell.
func CoreHeatmapCell(cores []core.CPUCore, width int) string {
	if width <= 0 {
		return ""
	}
	if len(cores) == 0 {
		return fillBlock(TextNA, width, ansi.Muted, ansi.PanelAltBg, false)
	}

	ordered := append([]core.CPUCore(nil), cores...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Index < ordered[j].Index })

	if width >= coreMultiRowMinWidth && len(ordered) > coreGroupSize {
		return multiRowCoreHeatmapCell(ordered, width)
	}

	return singleLineCoreHeatmapCell(ordered, width)
}

func multiRowCoreHeatmapCell(ordered []core.CPUCore, width int) string {
	const (
		coreCellWidth = 3
		coreGapWidth  = 1
		maxGridRows   = 8
	)

	maxCols := max(coreMinGridColumns, min(len(ordered), (width+coreGapWidth)/(coreCellWidth+coreGapWidth)))
	cols := squareGridColumns(len(ordered), maxCols)
	if cols < coreMinGridColumns {
		return singleLineCoreHeatmapCell(ordered, width)
	}

	rows := (len(ordered) + cols - 1) / cols
	for rows > maxGridRows && cols < maxCols {
		cols++
		rows = (len(ordered) + cols - 1) / cols
	}
	if rows > maxGridRows {
		rows = maxGridRows
	}

	slotCount := rows * cols
	peaks := aggregateCorePeaks(ordered, slotCount)
	gap := strings.Repeat(" ", coreGapWidth)
	lines := make([]string, 0, rows)
	for row := range rows {
		var b strings.Builder
		for col := range cols {
			slot := row*cols + col
			if col > 0 {
				b.WriteString(gap)
			}
			if slot >= len(peaks) || peaks[slot] < 0 {
				b.WriteString(ansi.StyledText("", ansi.TrackBg, strings.Repeat(" ", coreCellWidth)))

				continue
			}
			peak := peaks[slot]
			b.WriteString(ansi.StyledText(SeverityColor(UtilSeverity(peak)), ansi.TrackBg, strings.Repeat(coreLevelGlyph(peak), coreCellWidth)))
		}
		lines = append(lines, centerRenderedLine(b.String(), width))
	}

	return strings.Join(lines, "\n")
}

func singleLineCoreHeatmapCell(ordered []core.CPUCore, width int) string {
	groupGaps := 0
	if len(ordered) > coreGroupSize {
		groupGaps = (len(ordered) - 1) / coreGroupSize
	}
	if width < len(ordered)+groupGaps {
		if width < len(ordered) {
			return compressedCoreHeatmapCell(ordered, width)
		}
		groupGaps = 0
	}
	usableWidth := width - groupGaps
	baseWidth := usableWidth / len(ordered)
	extraWidth := usableWidth % len(ordered)
	if baseWidth < 1 {
		baseWidth = 1
	}

	var b strings.Builder
	for i, core := range ordered {
		if i > 0 && i%coreGroupSize == 0 {
			b.WriteString(ansi.Colorize(ansi.BorderColor, "╎"))
		}

		sev := UtilSeverity(core.Percent)
		glyph := coreLevelGlyph(core.Percent)
		segmentWidth := baseWidth
		if i < extraWidth {
			segmentWidth++
		}
		b.WriteString(ansi.StyledText(SeverityColor(sev), ansi.TrackBg, strings.Repeat(glyph, segmentWidth)))
	}

	return ansi.Pad(b.String(), width)
}

func aggregateCorePeaks(ordered []core.CPUCore, slotCount int) []int {
	if slotCount <= 0 {
		return nil
	}
	peaks := make([]int, slotCount)
	for i := range peaks {
		peaks[i] = -1
	}
	for slot := range slotCount {
		start := slot * len(ordered) / slotCount
		end := (slot + 1) * len(ordered) / slotCount
		if end <= start {
			if start >= len(ordered) {
				continue
			}
			end = start + 1
		}
		if start >= len(ordered) {
			continue
		}
		if end > len(ordered) {
			end = len(ordered)
		}
		peak := 0
		for _, core := range ordered[start:end] {
			if core.Percent > peak {
				peak = core.Percent
			}
		}
		peaks[slot] = peak
	}

	return peaks
}

func squareGridColumns(count, maxCols int) int {
	if count <= 1 {
		return 1
	}
	if maxCols < 1 {
		maxCols = 1
	}
	target := max(coreMinGridColumns, min(maxCols, intCeilSqrt(count)))
	bestCols := 1
	bestScore := gridSearchScoreCeil
	for cols := coreMinGridColumns; cols <= maxCols; cols++ {
		rows := (count + cols - 1) / cols
		score := absInt(rows-cols)*gridShapeScoreWeight + absInt(cols-target)
		if score < bestScore {
			bestScore = score
			bestCols = cols
		}
	}

	return bestCols
}

func intCeilSqrt(v int) int {
	if v <= 1 {
		return max(v, 1)
	}
	n := 1
	for n*n < v {
		n++
	}

	return n
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

func centerRenderedLine(content string, width int) string {
	contentWidth := ansi.VisibleLen(content)
	if contentWidth >= width {
		return ansi.Pad(content, width)
	}
	left := (width - contentWidth) / centerLineDivisor
	right := width - contentWidth - left

	return strings.Repeat(" ", left) + content + strings.Repeat(" ", right)
}

func compressedCoreHeatmapCell(ordered []core.CPUCore, width int) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	for slot := range width {
		start := slot * len(ordered) / width
		end := (slot + 1) * len(ordered) / width
		if end <= start {
			end = min(start+1, len(ordered))
		}

		peak := 0
		for _, core := range ordered[start:end] {
			if core.Percent > peak {
				peak = core.Percent
			}
		}
		b.WriteString(ansi.StyledText(SeverityColor(UtilSeverity(peak)), ansi.TrackBg, coreLevelGlyph(peak)))
	}

	return b.String()
}

func coreLevelGlyph(percent int) string {
	switch {
	case percent <= 0:
		return "▁"
	case percent <= coreLevelLow:
		return "▂"
	case percent <= coreLevelMild:
		return "▃"
	case percent <= coreLevelMedium:
		return "▄"
	case percent <= coreLevelRaised:
		return "▅"
	case percent <= coreLevelHigh:
		return "▆"
	case percent <= coreLevelVeryHigh:
		return "▇"
	default:
		return "█"
	}
}
