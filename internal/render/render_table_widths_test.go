//nolint:testpackage // Issue 58 requires direct in-package coverage of unexported render helpers.
package render

import (
	"strings"
	"testing"
)

func TestComputeTableWidthsForRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		boxWidth     int
		rows         []TableRowSpec
		wantLabel    int
		wantValue    int
		wantActivity int
	}{
		{
			name:         "fallback when box has no content",
			boxWidth:     0,
			rows:         nil,
			wantLabel:    tableFallbackLabelWidth,
			wantValue:    tableFallbackValueWidth,
			wantActivity: tableFallbackActivity,
		},
		{
			name:         "spare width grows value without activity cells",
			boxWidth:     60,
			rows:         []TableRowSpec{TableFullRow("Load", "", "1.23", "", "", "", "", "")},
			wantLabel:    tableMinLabelWidth,
			wantValue:    26,
			wantActivity: tableActivityBaseWidth,
		},
		{
			name:         "activity cells receive spare width",
			boxWidth:     80,
			rows:         []TableRowSpec{TableFullRow(LabelCPUMap, "", "99%", "", "", "", "", "busy")},
			wantLabel:    tableMinLabelWidth,
			wantValue:    tableMinValueWidth,
			wantActivity: 40,
		},
		{
			name:         "long pair values keep useful value width",
			boxWidth:     100,
			rows:         []TableRowSpec{TableFullRow("Throughput", "", strings.Repeat("1", 20)+" / "+strings.Repeat("2", 20), "", "", "", "", "busy")},
			wantLabel:    tableMinLabelWidth,
			wantValue:    tablePairValueCap,
			wantActivity: 44,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			labelWidth, valueWidth, activityWidth := ComputeTableWidthsForRows(
				tc.boxWidth,
				func(_, _ int) []TableRowSpec {
					return tc.rows
				},
			)
			if labelWidth != tc.wantLabel || valueWidth != tc.wantValue || activityWidth != tc.wantActivity {
				t.Fatalf("widths = %d/%d/%d, want %d/%d/%d", labelWidth, valueWidth, activityWidth, tc.wantLabel, tc.wantValue, tc.wantActivity)
			}
		})
	}
}

func TestDesiredTableWidths(t *testing.T) {
	t.Parallel()

	rows := []TableRowSpec{
		tableDividerRow(),
		TableFullRow("Processor Package", "", "", "", "  12345678901234567890  ", "", "", ""),
		TableFullRow(LabelCPUMap, "", "99%", "", "", "", "", "busy\nidle"),
	}

	labelWidth, valueWidth, activityWidth := desiredTableWidths(rows)
	if labelWidth != 17 || valueWidth != 20 || activityWidth != tableActivityMultilineMin {
		t.Fatalf("desired widths = %d/%d/%d, want 17/20/%d", labelWidth, valueWidth, activityWidth, tableActivityMultilineMin)
	}
}

func TestTableWidthHelpers(t *testing.T) {
	t.Parallel()

	activityRows := []TableRowSpec{TableFullRow(LabelCPUMap, "", "99%", "", "", "", "", "busy")}
	if got := preferredValueWidthCap(45, nil); got != tablePreferredValueMin {
		t.Fatalf("preferredValueWidthCap narrow = %d, want %d", got, tablePreferredValueMin)
	}
	if got := preferredValueWidthCap(90, nil); got != tablePreferredValueMax {
		t.Fatalf("preferredValueWidthCap wide = %d, want %d", got, tablePreferredValueMax)
	}
	if got := minimumActivityWidth(nil); got != tableActivityBaseWidth {
		t.Fatalf("minimumActivityWidth empty = %d, want %d", got, tableActivityBaseWidth)
	}
	if got := minimumActivityWidth(activityRows); got != tableActivityMultilineMin {
		t.Fatalf("minimumActivityWidth activity rows = %d, want %d", got, tableActivityMultilineMin)
	}
	if tableHasActivityCells(nil) {
		t.Fatal("tableHasActivityCells empty = true, want false")
	}
	if !tableHasActivityCells(activityRows) {
		t.Fatal("tableHasActivityCells activity rows = false, want true")
	}
}

func TestSummaryAndProcessTableWidths(t *testing.T) {
	t.Parallel()

	labelWidth, summaryWidth := computeSummaryTableWidths(47)
	if labelWidth != 10 || summaryWidth != 30 {
		t.Fatalf("summary widths = %d/%d, want 10/30", labelWidth, summaryWidth)
	}

	nameWidth, pidWidth, usageWidth, memoryWidth := computeProcessTableWidths(80)
	if nameWidth != 39 || pidWidth != processPIDWidth || usageWidth != processMinUsageWidth || memoryWidth != 13 {
		t.Fatalf("process widths = %d/%d/%d/%d, want 39/%d/%d/13", nameWidth, pidWidth, usageWidth, memoryWidth, processPIDWidth, processMinUsageWidth)
	}
}
