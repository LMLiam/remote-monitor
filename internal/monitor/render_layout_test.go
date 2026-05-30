package monitor_test

import (
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
	"testing"
)

func TestRenderResponsiveLayoutAdjustsColumnCountByWidth(t *testing.T) {
	t.Parallel()

	state := testTUIState()

	narrowLines := strings.Split(strings.TrimRight(ansi.StripANSI(render.Frame(state, 110, 92)), "\n"), "\n")
	mediumLines := strings.Split(strings.TrimRight(ansi.StripANSI(render.Frame(state, 176, 92)), "\n"), "\n")
	wideLines := strings.Split(strings.TrimRight(ansi.StripANSI(render.Frame(state, 240, 92)), "\n"), "\n")

	for _, line := range narrowLines {
		if lineHasAll(line, "│ CPU", "│ GPU") {
			t.Fatalf("narrow frame should reduce top-level columns, got %q", line)
		}
	}

	foundMediumTwoCol := false
	for _, line := range mediumLines {
		if lineHasAll(line, "│ CPU", "│ GPU") {
			foundMediumTwoCol = true

			break
		}
	}
	if !foundMediumTwoCol {
		t.Fatalf("medium frame should keep the CPU/GPU two-column layout")
	}

	if got := maxSectionTitlesOnLine(mediumLines); got != 2 {
		t.Fatalf("medium frame should use a two-column table layout, got max %d section titles on one line", got)
	}

	if got := maxSectionTitlesOnLine(wideLines); got <= 2 {
		t.Fatalf("wide frame should expand beyond the two-column layout when that reduces height, got max %d section titles on one line", got)
	}
}

func TestRenderWideLayoutKeepsCPUImbalanceReadable(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	lines := strings.SplitSeq(strings.TrimRight(ansi.StripANSI(render.Frame(state, 240, 92)), "\n"), "\n")

	for line := range lines {
		if strings.Contains(line, render.LabelCPUImbalance) && strings.Contains(line, "…") {
			t.Fatalf("wide layout should not truncate CPU Imbalance text, got %q", line)
		}
	}
}

func TestOptimizeSectionAssignmentsMinimizesMaxHeight(t *testing.T) {
	t.Parallel()

	sections := []render.RenderedLayoutSection{
		{ID: "a", Content: "", Height: 10},
		{ID: "b", Content: "", Height: 8},
		{ID: "c", Content: "", Height: 7},
		{ID: "d", Content: "", Height: 4},
	}

	assignments, heights, usedColumns := render.OptimizeSectionAssignments(sections, 2, render.SectionVerticalGap)

	if usedColumns != 2 {
		t.Fatalf("expected optimizer to use both columns, got %d", usedColumns)
	}

	maxHeight := 0
	for _, height := range heights {
		if height > maxHeight {
			maxHeight = height
		}
	}
	if maxHeight != 16 {
		t.Fatalf("expected exact minimum max height of 16, got %d with assignments %v and heights %v", maxHeight, assignments, heights)
	}
}

func TestBetterLayoutCandidatePrefersReadableWidthOverShorterHeight(t *testing.T) {
	t.Parallel()

	readable := render.LayoutCandidate{Columns: 3, MaxHeight: 20, Slack: 4, UsedColumn: 3, TruncationCount: 0, Rendered: nil}
	truncated := render.LayoutCandidate{Columns: 4, MaxHeight: 16, Slack: 0, UsedColumn: 4, TruncationCount: 2, Rendered: nil}

	if !render.BetterLayoutCandidate(readable, truncated) {
		t.Fatalf("expected readability to outrank vertical compactness: %+v vs %+v", readable, truncated)
	}
}

func TestRenderOptimizedSectionLayoutMatchesBruteForceMinimum(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	sections := render.BuildLayoutSections(state, true)

	for _, totalWidth := range []int{174, 238} {
		maxColumns := render.MaxFeasibleColumnCount(totalWidth, len(sections), render.MinResponsiveBoxWidth, render.SectionGap)
		best := bestRenderedCandidateForTest(totalWidth, sections, maxColumns)
		want := bruteForceCandidateForTest(totalWidth, sections, maxColumns)

		if best.TruncationCount != want.TruncationCount || best.MaxHeight != want.MaxHeight || best.Slack != want.Slack || best.Columns != want.Columns {
			t.Fatalf("width %d: optimizer chose columns=%d truncation=%d maxHeight=%d slack=%d, want columns=%d truncation=%d maxHeight=%d slack=%d", totalWidth, best.Columns, best.TruncationCount, best.MaxHeight, best.Slack, want.Columns, want.TruncationCount, want.MaxHeight, want.Slack)
		}
	}
}

func bestRenderedCandidateForTest(totalWidth int, sections []render.LayoutSection, maxColumns int) render.LayoutCandidate {
	best := emptyLayoutCandidateForTest()
	bestFound := false
	for columns := 1; columns <= maxColumns; columns++ {
		candidate := render.BuildLayoutCandidate(totalWidth, sections, columns, false)
		if !bestFound || render.BetterLayoutCandidate(candidate, best) {
			best = candidate
			bestFound = true
		}
	}

	return best
}

func bruteForceCandidateForTest(totalWidth int, sections []render.LayoutSection, maxColumns int) render.LayoutCandidate {
	best := emptyLayoutCandidateForTest()
	bestFound := false
	for columns := 1; columns <= maxColumns; columns++ {
		boxWidth := render.ResponsiveColumnWidth(totalWidth, columns, render.SectionGap)
		sectionCondensed := boxWidth < render.CondensedBoxWidth
		heights, truncationCount := renderSectionStatsForTest(sections, boxWidth, sectionCondensed)
		maxHeight, slack := bruteForceBestLayoutForTest(heights, columns, render.SectionVerticalGap)
		candidate := render.LayoutCandidate{
			Columns:         columns,
			MaxHeight:       maxHeight,
			Slack:           slack,
			UsedColumn:      columns,
			TruncationCount: truncationCount,
			Rendered:        nil,
		}
		if !bestFound || render.BetterLayoutCandidate(candidate, best) {
			best = candidate
			bestFound = true
		}
	}

	return best
}

func emptyLayoutCandidateForTest() render.LayoutCandidate {
	return render.LayoutCandidate{
		Columns:         0,
		MaxHeight:       0,
		Slack:           0,
		UsedColumn:      0,
		TruncationCount: 0,
		Rendered:        nil,
	}
}

func renderSectionStatsForTest(sections []render.LayoutSection, boxWidth int, condensed bool) (heights []int, totalHeight int) {
	heights = make([]int, 0, len(sections))
	truncationCount := 0
	for _, section := range sections {
		content := section.Render(boxWidth, condensed)
		if content == "" {
			continue
		}
		heights = append(heights, len(render.SplitRenderedLines(content)))
		truncationCount += render.SectionTruncationPenalty(section.ID, content)
	}

	return heights, truncationCount
}

func bruteForceBestLayoutForTest(heights []int, columns, gap int) (maxHeight, slack int) {
	bestMaxHeight := int(^uint(0) >> 1)
	bestSlack := int(^uint(0) >> 1)

	columnHeights := make([]int, columns)
	columnCounts := make([]int, columns)

	var search func(idx int)
	search = func(idx int) {
		if idx == len(heights) {
			currentMax := 0
			for _, height := range columnHeights {
				if height > currentMax {
					currentMax = height
				}
			}

			currentSlack := 0
			for _, height := range columnHeights {
				currentSlack += currentMax - height
			}

			if currentMax < bestMaxHeight || (currentMax == bestMaxHeight && currentSlack < bestSlack) {
				bestMaxHeight = currentMax
				bestSlack = currentSlack
			}

			return
		}

		for col := range columns {
			addedHeight := heights[idx]
			if columnCounts[col] > 0 {
				addedHeight += gap
			}

			columnCounts[col]++
			columnHeights[col] += addedHeight
			search(idx + 1)
			columnHeights[col] -= addedHeight
			columnCounts[col]--
		}
	}

	search(0)

	return bestMaxHeight, bestSlack
}
