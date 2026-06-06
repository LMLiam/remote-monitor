//nolint:testpackage // Issue 58 requires direct in-package coverage of unexported render helpers.
package render

import (
	"testing"
	"time"
)

func TestFormatMemoryValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "negative mib value", got: formatMiBValue(-1), want: TextNA},
		{name: "mib value", got: formatMiBValue(512), want: "512 MiB"},
		{name: "gib value", got: formatMiBValue(1024), want: "1.0 GiB"},
		{name: "negative mib pair", got: formatMiBPair(-1, 1024), want: TextNA},
		{name: "zero mib pair total", got: formatMiBPair(512, 0), want: TextNA},
		{name: "mib pair", got: formatMiBPair(512, 1024), want: "512 / 1024 MiB"},
		{name: "negative kib value", got: FormatKiBValue(-1), want: TextNA},
		{name: "kib value", got: FormatKiBValue(1), want: "1 KiB"},
		{name: "kib to mib value", got: FormatKiBValue(1024), want: "1.0 MiB"},
		{name: "kib to gib value", got: FormatKiBValue(1048576), want: "1.0 GiB"},
		{name: "negative kib pair", got: formatKiBPair(-1, 1024), want: TextNA},
		{name: "zero kib pair total", got: formatKiBPair(512, 0), want: TextNA},
		{name: "kib pair", got: formatKiBPair(1024, 1048576), want: "1.0 MiB / 1.0 GiB"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.got != tc.want {
				t.Fatalf("format result = %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestFormatMetricValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "negative bytes per second", got: formatBps(-1), want: TextNA},
		{name: "bytes per second", got: formatBps(999), want: "999 B/s"},
		{name: "kib per second", got: formatBps(1024), want: "1.0 KiB/s"},
		{name: "negative percent", got: percentDisplay(-1), want: TextNA},
		{name: "percent", got: percentDisplay(42), want: "42%"},
		{name: "negative temperature", got: tempDisplay(-1), want: TextNA},
		{name: "temperature", got: tempDisplay(70), want: "70C"},
		{name: "nonpositive float", got: formatFloat(0), want: "0.00"},
		{name: "positive float", got: formatFloat(1.235), want: "1.24"},
		{name: "negative millis", got: FormatMillisValue(-1), want: TextNA},
		{name: "millis", got: FormatMillisValue(1.234), want: "1.23 ms"},
		{name: "negative queue depth", got: FormatQueueDepth(-1), want: TextNA},
		{name: "queue depth", got: FormatQueueDepth(2.345), want: "2.35x"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.got != tc.want {
				t.Fatalf("format result = %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestFormatPowerAndClockValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "nonpositive power pair draw", got: formatPowerPair(0, 100), want: TextNA},
		{name: "nonpositive power pair limit", got: formatPowerPair(50, 0), want: TextNA},
		{name: "power pair", got: formatPowerPair(50, 100), want: "50.00W / 100.00W"},
		{name: "nonpositive power value", got: formatPowerValue(0), want: TextNA},
		{name: "power value", got: formatPowerValue(12.345), want: "12.35W"},
		{name: "clock pair", got: formatClockPair(1200, 3600), want: "1200 / 3600 MHz"},
		{name: "current clock only", got: formatClockPair(1200, 0), want: "1200 MHz"},
		{name: "max clock only", got: formatClockPair(-1, 3600), want: "max 3600 MHz"},
		{name: "missing clock pair", got: formatClockPair(-1, 0), want: TextNA},
		{name: "missing clock value", got: formatClockValue(-1), want: TextNA},
		{name: "clock value", got: formatClockValue(1200), want: "1200 MHz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.got != tc.want {
				t.Fatalf("format result = %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestFormatDurations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "missing uptime", got: formatUptime(0), want: TextNA},
		{name: "subminute uptime", got: formatUptime(59), want: "0m"},
		{name: "minute uptime", got: formatUptime(60), want: "1m"},
		{name: "hour uptime", got: formatUptime(3660), want: "1h 1m"},
		{name: "nonpositive age", got: formatAge(0), want: "0s"},
		{name: "rounded age seconds", got: formatAge(1500 * time.Millisecond), want: "2s"},
		{name: "age minutes", got: formatAge(75 * time.Second), want: "1m"},
		{name: "age hours", got: formatAge(2 * time.Hour), want: "2h"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.got != tc.want {
				t.Fatalf("format result = %q, want %q", tc.got, tc.want)
			}
		})
	}
}
