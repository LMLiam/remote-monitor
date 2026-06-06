//nolint:testpackage // Issue 58 requires direct in-package coverage of unexported render helpers.
package render

import (
	"testing"

	"github.com/lmliam/remote-monitor/internal/render/ansi"
)

const testMetricKindUtil = "util"

func TestSparkline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []int
		width  int
		want   string
	}{
		{name: "plain zero width", values: []int{100}, width: 0, want: ""},
		{name: "plain empty values", values: nil, width: 3, want: "···"},
		{name: "pads on the left", values: []int{100}, width: 3, want: "▁▁█"},
		{name: "clamps out of range", values: []int{-10, 110}, width: 2, want: "▁█"},
		{name: "uses newest values within width", values: []int{0, 25, 50, 75, 100}, width: 3, want: "▅▆█"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := sparkline(tc.values, tc.width); got != tc.want {
				t.Fatalf("sparkline(%v, %d) = %q, want %q", tc.values, tc.width, got, tc.want)
			}
		})
	}
}

func TestSparklineColoredVisibleOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		values     []int
		width      int
		metricKind string
		want       string
	}{
		{name: "colored zero width", values: []int{100}, width: 0, metricKind: testMetricKindUtil, want: ""},
		{name: "colored empty values", values: nil, width: 3, metricKind: testMetricKindUtil, want: "   "},
		{name: "pads colored values on the left", values: []int{100}, width: 3, metricKind: testMetricKindUtil, want: "  █"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ansi.StripANSI(sparklineColored(tc.values, tc.width, tc.metricKind))
			if got != tc.want {
				t.Fatalf("visible sparklineColored = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSparklineScaled64VisibleOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []int64
		width  int
		want   string
	}{
		{name: "scaled zero width", values: []int64{1}, width: 0, want: ""},
		{name: "scaled empty values", values: nil, width: 3, want: "   "},
		{name: "nonpositive peak fills baseline", values: []int64{0, 0}, width: 3, want: "▁▁▁"},
		{name: "scales to peak", values: []int64{1, 2, 4}, width: 3, want: "▃▅█"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ansi.StripANSI(sparklineScaled64(tc.values, tc.width, ansi.Cyan))
			if got != tc.want {
				t.Fatalf("visible sparklineScaled64 = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHistoryColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		metricKind string
		value      int
		want       string
	}{
		{name: "negative is dim", metricKind: testMetricKindUtil, value: -1, want: ansi.Dim},
		{name: "util uses severity", metricKind: testMetricKindUtil, value: 95, want: ansi.Red},
		{name: "clock is lavender", metricKind: "clock", value: 50, want: ansi.Lav},
		{name: "rx is blue", metricKind: "net-rx", value: 50, want: ansi.Blue},
		{name: "tx is cyan", metricKind: "net-tx", value: 50, want: ansi.Cyan},
		{name: "unknown is muted", metricKind: "custom", value: 50, want: ansi.Muted},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := historyColor(tc.metricKind, tc.value); got != tc.want {
				t.Fatalf("historyColor(%q, %d) = %q, want %q", tc.metricKind, tc.value, got, tc.want)
			}
		})
	}
}
