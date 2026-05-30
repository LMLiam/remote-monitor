package render

import (
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

// ComputeTableWidthsForRows computes balanced table column widths for generated rows.
func ComputeTableWidthsForRows(boxWidth int, buildRows func(valueWidth, activityWidth int) []TableRowSpec) (labelWidth, valueWidth, activityWidth int) {
	contentWidth := boxWidth - tableContentPadding
	if contentWidth <= 0 {
		return tableFallbackLabelWidth, tableFallbackValueWidth, tableFallbackActivity
	}

	rows := buildRows(contentWidth, contentWidth)
	labelWidth, valueWidth, activityWidth = desiredTableWidths(rows)
	minActivityWidth := minimumActivityWidth(rows)
	hasActivityCells := tableHasActivityCells(rows)
	if hasActivityCells {
		softValueCap := preferredValueWidthCap(contentWidth, rows)
		if valueWidth > softValueCap {
			activityWidth += valueWidth - softValueCap
			valueWidth = softValueCap
		}
	}

	short := (labelWidth + valueWidth + activityWidth) - contentWidth
	if short > 0 {
		reduceLabel := min(short, max(0, labelWidth-tableMinLabelWidth))
		labelWidth -= reduceLabel
		short -= reduceLabel
	}
	if short > 0 {
		reduceValue := min(short, max(0, valueWidth-tableMinValueWidth))
		valueWidth -= reduceValue
		short -= reduceValue
	}
	if short > 0 {
		reduceActivity := min(short, max(0, activityWidth-minActivityWidth))
		activityWidth -= reduceActivity
	}

	extra := contentWidth - labelWidth - valueWidth - activityWidth
	if extra > 0 {
		if hasActivityCells {
			growToDominance := min(extra, max(0, valueWidth+renderedBoxInnerTrim-activityWidth))
			activityWidth += growToDominance
			extra -= growToDominance

			growToParity := min(extra, max(0, valueWidth-activityWidth))
			activityWidth += growToParity
			extra -= growToParity

			activityWidth += extra
			extra = 0
		}
		valueWidth += extra
	}

	return labelWidth, valueWidth, activityWidth
}

func desiredTableWidths(rows []TableRowSpec) (labelWidth, valueWidth, activityWidth int) {
	labelWidth = ansi.VisibleLen("Metric")
	valueWidth = ansi.VisibleLen("Value")
	activityWidth = ansi.VisibleLen("Activity")
	minActivityWidth := minimumActivityWidth(rows)

	for _, row := range rows {
		if row.Divider {
			continue
		}

		labelWidth = max(labelWidth, ansi.VisibleLen(strings.TrimSpace(row.LabelText)))

		switch {
		case strings.TrimSpace(row.ValueText) != "":
			valueWidth = max(valueWidth, ansi.VisibleLen(strings.TrimSpace(row.ValueText)))
		case strings.TrimSpace(row.ValueCell) != "":
			valueWidth = max(valueWidth, ansi.VisibleLen(strings.TrimSpace(ansi.StripANSI(row.ValueCell))))
		}

		switch {
		case strings.TrimSpace(row.ActivityText) != "":
			activityWidth = max(activityWidth, ansi.VisibleLen(strings.TrimSpace(row.ActivityText)))
		case row.ActivityCell != "":
			activityWidth = max(activityWidth, minActivityWidth)
		}
	}

	labelWidth = clamp(labelWidth, tableMinLabelWidth, tableMaxLabelWidth)
	valueWidth = max(tableMinValueWidth, valueWidth)
	activityWidth = max(minActivityWidth, activityWidth)

	return labelWidth, valueWidth, activityWidth
}

func preferredValueWidthCap(contentWidth int, rows []TableRowSpec) int {
	capWidth := clamp(contentWidth/tablePreferredValueDiv+tablePreferredValueExtra, tablePreferredValueMin, tablePreferredValueMax)
	for _, row := range rows {
		if row.Divider {
			continue
		}
		valueTextWidth := ansi.VisibleLen(strings.TrimSpace(row.ValueText))
		if strings.Contains(row.ValueText, "/") || strings.Contains(row.ValueText, "•") {
			capWidth = max(capWidth, min(tablePairValueCap, valueTextWidth+tablePreferredValueExtra))
		}
	}

	return capWidth
}

func minimumActivityWidth(rows []TableRowSpec) int {
	minWidth := tableActivityBaseWidth
	for _, row := range rows {
		if row.ActivityCell != "" {
			minWidth = max(minWidth, tableActivityCellMin)
		}
		if row.LabelText == LabelCPUMap || strings.Contains(row.ActivityCell, "\n") {
			minWidth = max(minWidth, tableActivityMultilineMin)
		}
	}

	return minWidth
}

func tableHasActivityCells(rows []TableRowSpec) bool {
	for _, row := range rows {
		if row.ActivityCell != "" {
			return true
		}
	}

	return false
}

func computeSummaryTableWidths(boxWidth int) (labelWidth, summaryWidth int) {
	contentWidth := boxWidth - summaryTablePadding
	labelWidth = clamp(contentWidth/summaryLabelDivisor, summaryMinLabelWidth, summaryMaxLabelWidth)
	summaryWidth = contentWidth - labelWidth

	return labelWidth, summaryWidth
}

func computeProcessTableWidths(boxWidth int) (nameWidth, pidWidth, usageWidth, memoryWidth int) {
	contentWidth := boxWidth - processTablePadding
	pidWidth = processPIDWidth
	usageWidth = clamp(contentWidth/processUsageDivisor, processMinUsageWidth, processMaxUsageWidth)
	memoryWidth = clamp(contentWidth/processMemoryDivisor, processMinMemoryWidth, processMaxMemoryWidth)
	nameWidth = contentWidth - pidWidth - usageWidth - memoryWidth

	if nameWidth < processMinNameWidth {
		short := processMinNameWidth - nameWidth
		reduceMemory := min(short, max(0, memoryWidth-processMinMemoryWidth))
		memoryWidth -= reduceMemory
		short -= reduceMemory
		reduceUsage := min(short, max(0, usageWidth-processMinUsageWidth))
		usageWidth -= reduceUsage
		nameWidth = contentWidth - pidWidth - usageWidth - memoryWidth
	}

	return nameWidth, pidWidth, usageWidth, memoryWidth
}
