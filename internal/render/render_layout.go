package render

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

// SectionGap and related constants define responsive table layout spacing.
const (
	SectionGap            = 2
	SectionVerticalGap    = 1
	MinResponsiveBoxWidth = 58
	CondensedBoxWidth     = 84

	minMultiColumnCandidate  = 2
	layoutProbeActivityWidth = 18
)

// LayoutSection describes a renderable dashboard section.
type LayoutSection struct {
	ID     string
	Render func(boxWidth int, condensed bool) string
}

// RenderedLayoutSection stores rendered section content and measured height.
type RenderedLayoutSection struct {
	ID      string
	Content string
	Height  int
}

// LayoutCandidate describes one possible responsive section arrangement.
type LayoutCandidate struct {
	Columns         int
	MaxHeight       int
	Slack           int
	UsedColumn      int
	TruncationCount int
	Rendered        [][]RenderedLayoutSection
}

type columnState struct {
	height   int
	nonEmpty bool
}

func renderTableLayout(state core.AppState, totalWidth int, showProcessPanels, condensed bool) string {
	sections := BuildLayoutSections(state, showProcessPanels)
	if len(sections) == 0 {
		return ""
	}

	return renderOptimizedSectionLayout(totalWidth, sections, condensed)
}

// BuildLayoutSections returns the ordered dashboard sections available for state.
func BuildLayoutSections(state core.AppState, showProcessPanels bool) []LayoutSection {
	sections := []LayoutSection{
		{ID: "cpu", Render: func(boxWidth int, condensed bool) string { return renderCPUSection(state, boxWidth, condensed) }},
		{ID: "gpu", Render: func(boxWidth int, condensed bool) string { return renderGPUSection(state, boxWidth, condensed) }},
		{ID: "system", Render: func(boxWidth int, _ bool) string { return renderSystemSection(state, boxWidth) }},
		{ID: "memory", Render: func(boxWidth int, condensed bool) string { return renderMemorySection(state, boxWidth, condensed) }},
	}

	if len(BuildStorageRows(state, layoutProbeActivityWidth, false)) > 0 {
		sections = append(sections, LayoutSection{ID: "storage", Render: func(boxWidth int, condensed bool) string { return renderStorageSection(state, boxWidth, condensed) }})
	}
	if len(BuildNetworkRows(state, layoutProbeActivityWidth, false)) > 0 {
		sections = append(sections, LayoutSection{ID: "network", Render: func(boxWidth int, condensed bool) string { return renderNetworkSection(state, boxWidth, condensed) }})
	}
	if len(BuildPowerRows(state, layoutProbeActivityWidth, false)) > 0 {
		sections = append(sections, LayoutSection{ID: "power", Render: func(boxWidth int, condensed bool) string { return renderPowerSection(state, boxWidth, condensed) }})
	}
	if showProcessPanels {
		sections = append(sections,
			LayoutSection{ID: "top-processes", Render: func(boxWidth int, _ bool) string { return renderTopProcessesSection(state, boxWidth) }},
			LayoutSection{ID: "gpu-processes", Render: func(boxWidth int, _ bool) string { return renderGPUProcessesSection(state, boxWidth) }},
		)
	}

	return sections
}

func renderOptimizedSectionLayout(totalWidth int, sections []LayoutSection, condensed bool) string {
	maxColumns := 1
	if !condensed {
		maxColumns = MaxFeasibleColumnCount(totalWidth, len(sections), MinResponsiveBoxWidth, SectionGap)
	}

	bestFound := false
	var best LayoutCandidate
	for columns := 1; columns <= maxColumns; columns++ {
		candidate := BuildLayoutCandidate(totalWidth, sections, columns, condensed)
		if !bestFound || BetterLayoutCandidate(candidate, best) {
			best = candidate
			bestFound = true
		}
	}

	return renderLayoutCandidate(best)
}

// MaxFeasibleColumnCount returns the widest column count that preserves minimum width.
func MaxFeasibleColumnCount(totalWidth, sectionCount, minBoxWidth, gap int) int {
	if sectionCount < 1 {
		return 1
	}

	maxColumns := 1
	for columns := minMultiColumnCandidate; columns <= sectionCount; columns++ {
		if ResponsiveColumnWidth(totalWidth, columns, gap) < minBoxWidth {
			break
		}
		maxColumns = columns
	}

	return maxColumns
}

// BuildLayoutCandidate renders and scores one column-count layout candidate.
func BuildLayoutCandidate(totalWidth int, sections []LayoutSection, columns int, condensed bool) LayoutCandidate {
	boxWidth := ResponsiveColumnWidth(totalWidth, columns, SectionGap)
	sectionCondensed := condensed || boxWidth < CondensedBoxWidth

	rendered := make([]RenderedLayoutSection, 0, len(sections))
	truncationCount := 0
	for _, section := range sections {
		content := section.Render(boxWidth, sectionCondensed)
		if content == "" {
			continue
		}
		truncationCount += SectionTruncationPenalty(section.ID, content)
		rendered = append(rendered, RenderedLayoutSection{
			ID:      section.ID,
			Content: content,
			Height:  len(SplitRenderedLines(content)),
		})
	}

	assignments, heights, usedColumns := OptimizeSectionAssignments(rendered, columns, SectionVerticalGap)
	columnSections := make([][]RenderedLayoutSection, columns)
	for idx, column := range assignments {
		columnSections[column] = append(columnSections[column], rendered[idx])
	}

	maxHeight := 0
	for _, height := range heights {
		if height > maxHeight {
			maxHeight = height
		}
	}
	slack := 0
	for _, height := range heights {
		slack += maxHeight - height
	}

	return LayoutCandidate{
		Columns:         columns,
		MaxHeight:       maxHeight,
		Slack:           slack,
		UsedColumn:      usedColumns,
		TruncationCount: truncationCount,
		Rendered:        columnSections,
	}
}

type sectionAssignmentOptimizer struct {
	sections        []RenderedLayoutSection
	columns         int
	gap             int
	assignments     []int
	columnHeights   []int
	columnCounts    []int
	bestAssignments []int
	bestHeights     []int
	bestMaxHeight   int
	bestSlack       int
	bestUsedColumns int
}

// OptimizeSectionAssignments assigns sections to columns with balanced heights.
func OptimizeSectionAssignments(sections []RenderedLayoutSection, columns, gap int) (bestAssignments, bestHeights []int, bestUsedColumns int) {
	if len(sections) == 0 || columns <= 0 {
		return nil, make([]int, max(columns, 0)), 0
	}

	optimizer := sectionAssignmentOptimizer{
		sections:        sections,
		columns:         columns,
		gap:             gap,
		assignments:     make([]int, len(sections)),
		columnHeights:   make([]int, columns),
		columnCounts:    make([]int, columns),
		bestAssignments: make([]int, len(sections)),
		bestHeights:     make([]int, columns),
		bestMaxHeight:   int(^uint(0) >> 1),
		bestSlack:       int(^uint(0) >> 1),
		bestUsedColumns: int(^uint(0) >> 1),
	}

	optimizer.search(0)

	return optimizer.bestAssignments, optimizer.bestHeights, optimizer.bestUsedColumns
}

func (optimizer *sectionAssignmentOptimizer) search(sectionIndex int) {
	if sectionIndex == len(optimizer.sections) {
		optimizer.recordCandidate()

		return
	}

	tried := map[columnState]bool{}
	for column := range optimizer.columns {
		state := columnState{
			height:   optimizer.columnHeights[column],
			nonEmpty: optimizer.columnCounts[column] > 0,
		}
		if tried[state] {
			continue
		}
		tried[state] = true
		optimizer.tryColumn(sectionIndex, column)
	}
}

func (optimizer *sectionAssignmentOptimizer) tryColumn(sectionIndex, column int) {
	addedHeight := optimizer.sections[sectionIndex].Height
	if optimizer.columnCounts[column] > 0 {
		addedHeight += optimizer.gap
	}

	optimizer.assignments[sectionIndex] = column
	optimizer.columnCounts[column]++
	optimizer.columnHeights[column] += addedHeight
	if optimizer.currentMaxHeight() <= optimizer.bestMaxHeight {
		optimizer.search(sectionIndex + 1)
	}
	optimizer.columnHeights[column] -= addedHeight
	optimizer.columnCounts[column]--
}

func (optimizer *sectionAssignmentOptimizer) recordCandidate() {
	currentMax := optimizer.currentMaxHeight()
	usedColumns := optimizer.usedColumns()
	currentSlack := optimizer.currentSlack(currentMax)
	if !optimizer.isBetterCandidate(currentMax, currentSlack, usedColumns) {
		return
	}

	optimizer.bestMaxHeight = currentMax
	optimizer.bestSlack = currentSlack
	optimizer.bestUsedColumns = usedColumns
	copy(optimizer.bestAssignments, optimizer.assignments)
	copy(optimizer.bestHeights, optimizer.columnHeights)
}

func (optimizer *sectionAssignmentOptimizer) currentMaxHeight() int {
	currentMax := 0
	for _, height := range optimizer.columnHeights {
		if height > currentMax {
			currentMax = height
		}
	}

	return currentMax
}

func (optimizer *sectionAssignmentOptimizer) usedColumns() int {
	usedColumns := 0
	for _, count := range optimizer.columnCounts {
		if count > 0 {
			usedColumns++
		}
	}

	return usedColumns
}

func (optimizer *sectionAssignmentOptimizer) currentSlack(currentMax int) int {
	slack := 0
	for _, height := range optimizer.columnHeights {
		slack += currentMax - height
	}

	return slack
}

func (optimizer *sectionAssignmentOptimizer) isBetterCandidate(currentMax, currentSlack, usedColumns int) bool {
	if currentMax != optimizer.bestMaxHeight {
		return currentMax < optimizer.bestMaxHeight
	}
	if currentSlack != optimizer.bestSlack {
		return currentSlack < optimizer.bestSlack
	}

	return usedColumns < optimizer.bestUsedColumns
}

// BetterLayoutCandidate reports whether candidate is preferable to best.
func BetterLayoutCandidate(candidate, best LayoutCandidate) bool {
	if candidate.TruncationCount != best.TruncationCount {
		return candidate.TruncationCount < best.TruncationCount
	}
	if candidate.MaxHeight != best.MaxHeight {
		return candidate.MaxHeight < best.MaxHeight
	}
	if candidate.Slack != best.Slack {
		return candidate.Slack < best.Slack
	}
	if candidate.Columns != best.Columns {
		return candidate.Columns < best.Columns
	}

	return candidate.UsedColumn < best.UsedColumn
}

func countTruncationMarkers(content string) int {
	return strings.Count(ansi.StripANSI(content), "…")
}

// SectionTruncationPenalty counts readability loss for one rendered section.
func SectionTruncationPenalty(sectionID, content string) int {
	switch sectionID {
	case "top-processes", "gpu-processes":
		return 0
	default:
		return countTruncationMarkers(content)
	}
}

func renderLayoutCandidate(candidate LayoutCandidate) string {
	renderedColumns := make([][]string, 0, len(candidate.Rendered))
	for _, column := range candidate.Rendered {
		if len(column) == 0 {
			continue
		}

		parts := make([]string, 0, len(column))
		for _, section := range column {
			parts = append(parts, section.Content)
		}
		renderedColumns = append(renderedColumns, SplitRenderedLines(strings.Join(parts, "\n\n")))
	}

	if len(renderedColumns) == 0 {
		return ""
	}
	if len(renderedColumns) == 1 {
		return strings.Join(renderedColumns[0], "\n")
	}

	return joinLineColumnSet(renderedColumns, SectionGap)
}

func renderCPUSection(state core.AppState, boxWidth int, condensed bool) string {
	labelWidth, valueWidth, activityWidth := ComputeTableWidthsForRows(boxWidth, func(valueWidth, activityWidth int) []TableRowSpec {
		return BuildCPURows(state, valueWidth, activityWidth, condensed)
	})
	rows := BuildCPURows(state, valueWidth, activityWidth, condensed)

	return strings.Join(TableBox(cpuTableTitle(state.Current), rows, labelWidth, valueWidth, activityWidth), "\n")
}

func renderGPUSection(state core.AppState, boxWidth int, condensed bool) string {
	labelWidth, valueWidth, activityWidth := ComputeTableWidthsForRows(boxWidth, func(valueWidth, activityWidth int) []TableRowSpec {
		return buildGPURows(state, valueWidth, activityWidth, condensed)
	})
	rows := buildGPURows(state, valueWidth, activityWidth, condensed)

	return strings.Join(TableBox(gpuTableTitle(state.Current), rows, labelWidth, valueWidth, activityWidth), "\n")
}

func renderSystemSection(state core.AppState, boxWidth int) string {
	labelWidth, summaryWidth := computeSummaryTableWidths(boxWidth)

	return strings.Join(renderSummaryTableBox("System", buildSystemSummaryRows(state, summaryWidth), labelWidth, summaryWidth), "\n")
}

func renderMemorySection(state core.AppState, boxWidth int, condensed bool) string {
	labelWidth, valueWidth, activityWidth := ComputeTableWidthsForRows(boxWidth, func(valueWidth, activityWidth int) []TableRowSpec {
		_ = valueWidth

		return buildMemoryRows(state, activityWidth, condensed)
	})
	rows := buildMemoryRows(state, activityWidth, condensed)

	return strings.Join(TableBox("Memory", rows, labelWidth, valueWidth, activityWidth), "\n")
}

func renderStorageSection(state core.AppState, boxWidth int, condensed bool) string {
	labelWidth, valueWidth, activityWidth := ComputeTableWidthsForRows(boxWidth, func(valueWidth, activityWidth int) []TableRowSpec {
		_ = valueWidth

		return BuildStorageRows(state, activityWidth, condensed)
	})
	rows := BuildStorageRows(state, activityWidth, condensed)
	if len(rows) == 0 {
		return ""
	}

	return strings.Join(TableBox("Storage", rows, labelWidth, valueWidth, activityWidth), "\n")
}

func renderNetworkSection(state core.AppState, boxWidth int, condensed bool) string {
	labelWidth, valueWidth, activityWidth := ComputeTableWidthsForRows(boxWidth, func(valueWidth, activityWidth int) []TableRowSpec {
		_ = valueWidth

		return BuildNetworkRows(state, activityWidth, condensed)
	})
	rows := BuildNetworkRows(state, activityWidth, condensed)
	if len(rows) == 0 {
		return ""
	}

	return strings.Join(TableBox("Network", rows, labelWidth, valueWidth, activityWidth), "\n")
}

func renderPowerSection(state core.AppState, boxWidth int, condensed bool) string {
	labelWidth, valueWidth, activityWidth := ComputeTableWidthsForRows(boxWidth, func(valueWidth, activityWidth int) []TableRowSpec {
		_ = valueWidth

		return BuildPowerRows(state, activityWidth, condensed)
	})
	rows := BuildPowerRows(state, activityWidth, condensed)
	if len(rows) == 0 {
		return ""
	}

	return strings.Join(TableBox("Power", rows, labelWidth, valueWidth, activityWidth), "\n")
}

func renderTopProcessesSection(state core.AppState, boxWidth int) string {
	nameWidth, pidWidth, usageWidth, memoryWidth := computeProcessTableWidths(boxWidth)

	return strings.Join(
		renderProcessTableBox("Top Processes", [4]string{LabelProcess, LabelPID, "CPU", "RSS"}, buildTopProcessTableRows(state), nameWidth, pidWidth, usageWidth, memoryWidth),
		"\n",
	)
}

func renderGPUProcessesSection(state core.AppState, boxWidth int) string {
	nameWidth, pidWidth, usageWidth, memoryWidth := computeProcessTableWidths(boxWidth)

	return strings.Join(
		renderProcessTableBox("GPU Processes", [4]string{LabelProcess, LabelPID, LabelGPU, "VRAM"}, buildGPUProcessTableRows(state), nameWidth, pidWidth, usageWidth, memoryWidth),
		"\n",
	)
}

// ResponsiveColumnWidth returns the content width for one responsive column.
func ResponsiveColumnWidth(totalWidth, columns, gap int) int {
	if columns <= 1 {
		return totalWidth
	}

	return (totalWidth - ((columns - 1) * gap)) / columns
}

func joinLineColumnSet(columns [][]string, gap int) string {
	if len(columns) == 0 {
		return ""
	}

	widths := make([]int, len(columns))
	maxLines := 0
	for i, column := range columns {
		if len(column) > maxLines {
			maxLines = len(column)
		}
		for _, line := range column {
			if width := ansi.VisibleLen(line); width > widths[i] {
				widths[i] = width
			}
		}
	}

	spacer := strings.Repeat(" ", gap)
	var b strings.Builder
	for row := range maxLines {
		for col, lines := range columns {
			line := strings.Repeat(" ", widths[col])
			if row < len(lines) {
				line = ansi.Pad(lines[row], widths[col])
			}
			b.WriteString(line)
			if col < len(columns)-1 {
				b.WriteString(spacer)
			}
		}
		if row < maxLines-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
