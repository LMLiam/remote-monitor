package metrics_test

import (
	"testing"

	"github.com/lmliam/remote-monitor/internal/metrics"
)

func TestRatePercentHandlesSentinelsMinimumAndClamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   int64
		ceiling int64
		want    int
	}{
		{name: "unknown value", value: -1, ceiling: 100, want: 0},
		{name: "zero value", value: 0, ceiling: 100, want: 0},
		{name: "unknown ceiling", value: 10, ceiling: -1, want: 0},
		{name: "tiny positive value", value: 1, ceiling: 1000, want: 1},
		{name: "half", value: 50, ceiling: 100, want: 50},
		{name: "clamped", value: 150, ceiling: 100, want: 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := metrics.RatePercent(tc.value, tc.ceiling); got != tc.want {
				t.Fatalf("RatePercent(%d, %d) = %d, want %d", tc.value, tc.ceiling, got, tc.want)
			}
		})
	}
}

func TestPercentOfHandlesSentinelsRoundingAndClamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		used  int64
		total int64
		want  int
	}{
		{name: "unknown used", used: -1, total: 100, want: 0},
		{name: "zero total", used: 10, total: 0, want: 0},
		{name: "rounds", used: 1, total: 3, want: 33},
		{name: "half", used: 50, total: 100, want: 50},
		{name: "clamped", used: 150, total: 100, want: 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := metrics.PercentOf(tc.used, tc.total); got != tc.want {
				t.Fatalf("PercentOf(%d, %d) = %d, want %d", tc.used, tc.total, got, tc.want)
			}
		})
	}
}

func TestClockPowerAndTemperaturePercentHelpers(t *testing.T) {
	t.Parallel()

	if got := metrics.ClockPercent(-1, 4700); got != -1 {
		t.Fatalf("ClockPercent unknown current = %d, want -1", got)
	}
	if got := metrics.ClockPercent(2300, 0); got != -1 {
		t.Fatalf("ClockPercent missing max = %d, want -1", got)
	}
	if got := metrics.ClockPercent(2350, 4700); got != 50 {
		t.Fatalf("ClockPercent half = %d, want 50", got)
	}
	if got := metrics.ClockPercent(6000, 4700); got != 100 {
		t.Fatalf("ClockPercent clamped = %d, want 100", got)
	}

	if got := metrics.PowerPercent(0, 100); got != 0 {
		t.Fatalf("PowerPercent zero draw = %d, want 0", got)
	}
	if got := metrics.PowerPercent(50, 100); got != 50 {
		t.Fatalf("PowerPercent half = %d, want 50", got)
	}
	if got := metrics.PowerPercent(150, 100); got != 100 {
		t.Fatalf("PowerPercent clamped = %d, want 100", got)
	}

	if got := metrics.TemperaturePercent(-1); got != 0 {
		t.Fatalf("TemperaturePercent unknown = %d, want 0", got)
	}
	if got := metrics.TemperaturePercent(63); got != 63 {
		t.Fatalf("TemperaturePercent known = %d, want 63", got)
	}
	if got := metrics.TemperaturePercent(150); got != 100 {
		t.Fatalf("TemperaturePercent clamped = %d, want 100", got)
	}
}

func TestClampAndMax64Helpers(t *testing.T) {
	t.Parallel()

	if got := metrics.Clamp(-5, 0, 100); got != 0 {
		t.Fatalf("Clamp below range = %d, want 0", got)
	}
	if got := metrics.Clamp(50, 0, 100); got != 50 {
		t.Fatalf("Clamp inside range = %d, want 50", got)
	}
	if got := metrics.Clamp(150, 0, 100); got != 100 {
		t.Fatalf("Clamp above range = %d, want 100", got)
	}
	if got := metrics.Max64(10, 20); got != 20 {
		t.Fatalf("Max64 right = %d, want 20", got)
	}
	if got := metrics.Max64(30, 20); got != 30 {
		t.Fatalf("Max64 left = %d, want 30", got)
	}
}
