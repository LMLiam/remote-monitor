package render

import (
	"fmt"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

// TableBox renders a titled three-column metrics table.
func TableBox(title string, rows []TableRowSpec, labelWidth, valueWidth, activityWidth int) []string {
	outerWidth := labelWidth + valueWidth + activityWidth + tableContentPadding
	lines := make([]string, 0, len(rows)+renderedTableRowCapacity)
	lines = append(lines,
		trimLine(boxRule("╭", "╮", outerWidth-renderedBoxInnerTrim)),
		trimLine(boxLine(title, outerWidth-renderedBoxInnerTrim, ansi.TitleColor)),
		trimLine(tableRule("├", "┬", "┬", "┤", labelWidth, valueWidth, activityWidth)),
		trimLine(renderRow(
			tableHeaderCell("Metric", labelWidth),
			tableHeaderCell("Value", valueWidth),
			tableHeaderCell("Activity", activityWidth),
		)),
		trimLine(tableRule("├", "┼", "┼", "┤", labelWidth, valueWidth, activityWidth)),
	)

	for _, row := range rows {
		if row.Divider {
			lines = append(lines, trimLine(tableRule("├", "┼", "┼", "┤", labelWidth, valueWidth, activityWidth)))

			continue
		}
		labelLines := splitRenderedCellLines(cell(row.LabelText, labelWidth, row.LabelColor), labelWidth)
		value := row.ValueCell
		if value == "" {
			value = cell(row.ValueText, valueWidth, row.ValueColor)
		}
		valueLines := splitRenderedCellLines(value, valueWidth)
		activity := row.ActivityCell
		if activity == "" {
			activity = cell(row.ActivityText, activityWidth, row.ActivityColor)
		}
		activityLines := splitRenderedCellLines(activity, activityWidth)

		rowHeight := max(len(labelLines), max(len(valueLines), len(activityLines)))
		for i := range rowHeight {
			lines = append(lines, trimLine(renderRow(
				renderedCellLine(labelLines, i, labelWidth),
				renderedCellLine(valueLines, i, valueWidth),
				renderedCellLine(activityLines, i, activityWidth),
			)))
		}
	}

	lines = append(lines, trimLine(tableRule("╰", "┴", "┴", "╯", labelWidth, valueWidth, activityWidth)))

	return lines
}

func renderSummaryTableBox(title string, rows []StatusSummaryRowSpec, labelWidth, summaryWidth int) []string {
	outerWidth := labelWidth + summaryWidth + summaryTablePadding
	lines := make([]string, 0, len(rows)+renderedTableRowCapacity)
	lines = append(lines,
		trimLine(boxRule("╭", "╮", outerWidth-renderedBoxInnerTrim)),
		trimLine(boxLine(title, outerWidth-renderedBoxInnerTrim, ansi.TitleColor)),
		trimLine(summaryRule("├", "┬", "┤", labelWidth, summaryWidth)),
		trimLine(renderSummaryRow(
			tableHeaderCell("Signal", labelWidth),
			tableHeaderCell("Summary", summaryWidth),
		)),
		trimLine(summaryRule("├", "┼", "┤", labelWidth, summaryWidth)),
	)

	for _, row := range rows {
		lines = append(lines, trimLine(renderSummaryRow(
			cell(row.LabelText, labelWidth, row.LabelColor),
			ansi.Pad(row.SummaryCell, summaryWidth),
		)))
	}

	lines = append(lines, trimLine(summaryRule("╰", "┴", "╯", labelWidth, summaryWidth)))

	return lines
}

func renderProcessTableBox(title string, headers [4]string, rows []ProcessListRowSpec, nameWidth, pidWidth, usageWidth, memoryWidth int) []string {
	outerWidth := nameWidth + pidWidth + usageWidth + memoryWidth + processTablePadding
	lines := make([]string, 0, len(rows)+renderedTableRowCapacity)
	lines = append(lines,
		trimLine(boxRule("╭", "╮", outerWidth-renderedBoxInnerTrim)),
		trimLine(boxLine(title, outerWidth-renderedBoxInnerTrim, ansi.TitleColor)),
		trimLine(processRule("├", "┬", "┬", "┬", "┤", nameWidth, pidWidth, usageWidth, memoryWidth)),
		trimLine(renderProcessRow(
			tableHeaderCell(headers[0], nameWidth),
			tableHeaderCell(headers[1], pidWidth),
			tableHeaderCell(headers[2], usageWidth),
			tableHeaderCell(headers[3], memoryWidth),
		)),
		trimLine(processRule("├", "┼", "┼", "┼", "┤", nameWidth, pidWidth, usageWidth, memoryWidth)),
	)

	for _, row := range rows {
		lines = append(lines, trimLine(renderProcessRow(
			cell(row.ProcessText, nameWidth, row.ProcessColor),
			cell(row.PIDText, pidWidth, row.PIDColor),
			cell(row.UsageText, usageWidth, row.UsageColor),
			cell(row.MemoryText, memoryWidth, row.MemoryColor),
		)))
	}

	lines = append(lines, trimLine(processRule("╰", "┴", "┴", "┴", "╯", nameWidth, pidWidth, usageWidth, memoryWidth)))

	return lines
}

func trimLine(s string) string {
	return strings.TrimSuffix(s, "\n")
}

func splitRenderedCellLines(content string, width int) []string {
	parts := strings.Split(content, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		lines = append(lines, ansi.Pad(part, width))
	}
	if len(lines) == 0 {
		return []string{cell("", width, "")}
	}

	return lines
}

func renderedCellLine(lines []string, idx, width int) string {
	if idx < len(lines) {
		return ansi.Pad(lines[idx], width)
	}

	return cell("", width, "")
}

func tableRule(left, mid1, mid2, right string, labelWidth, valueWidth, activityWidth int) string {
	rule := left + strings.Repeat("─", labelWidth+ruleCellPadding) + mid1 + strings.Repeat("─", valueWidth+ruleCellPadding) + mid2 + strings.Repeat("─", activityWidth+ruleCellPadding) + right

	return ansi.Colorize(ansi.BorderColor, rule) + "\n"
}

func summaryRule(left, mid, right string, labelWidth, summaryWidth int) string {
	rule := left + strings.Repeat("─", labelWidth+ruleCellPadding) + mid + strings.Repeat("─", summaryWidth+ruleCellPadding) + right

	return ansi.Colorize(ansi.BorderColor, rule) + "\n"
}

func processRule(left, mid1, mid2, mid3, right string, nameWidth, pidWidth, usageWidth, memoryWidth int) string {
	rule := left + strings.Repeat("─", nameWidth+ruleCellPadding) + mid1 + strings.Repeat("─", pidWidth+ruleCellPadding) + mid2 + strings.Repeat("─", usageWidth+ruleCellPadding) + mid3 + strings.Repeat("─", memoryWidth+ruleCellPadding) + right

	return ansi.Colorize(ansi.BorderColor, rule) + "\n"
}

func renderRow(label, value, activity string) string {
	border := ansi.Colorize(ansi.BorderColor, "│")

	return fmt.Sprintf("%s %s %s %s %s %s %s\n", border, label, border, value, border, activity, border)
}

func renderSummaryRow(label, summary string) string {
	border := ansi.Colorize(ansi.BorderColor, "│")

	return fmt.Sprintf("%s %s %s %s %s\n", border, label, border, summary, border)
}

func renderProcessRow(name, pid, usage, memory string) string {
	border := ansi.Colorize(ansi.BorderColor, "│")

	return fmt.Sprintf("%s %s %s %s %s %s %s %s %s\n", border, name, border, pid, border, usage, border, memory, border)
}

func boxRule(left, right string, innerWidth int) string {
	return ansi.Colorize(ansi.BorderColor, left+strings.Repeat("─", innerWidth+ruleCellPadding)+right) + "\n"
}

func boxLine(text string, innerWidth int, color string) string {
	if color == "" && ansi.HasANSI(text) {
		border := ansi.Colorize(ansi.BorderColor, "│")

		return border + " " + ansi.Pad(text, innerWidth) + " " + border + "\n"
	}
	fg := ansi.Sand
	bg := ansi.PanelBg
	attrs := []string{}
	if color != "" {
		fg = color
		bg = ansi.HeaderBg
		attrs = append(attrs, ansi.Bold)
	}
	content := fillBlock(text, innerWidth, fg, bg, len(attrs) > 0)
	border := ansi.Colorize(ansi.BorderColor, "│")

	return border + " " + content + " " + border + "\n"
}

func cell(text string, width int, color string) string {
	fg := color
	if fg == "" {
		fg = ansi.Sand
	}

	return fillBlock(text, width, fg, ansi.PanelBg, false)
}

func tableHeaderCell(text string, width int) string {
	return fillBlock(text, width, ansi.Sand, ansi.PanelAltBg, true)
}
