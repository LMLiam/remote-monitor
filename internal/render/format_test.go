//nolint:testpackage // Issue 58 requires direct in-package coverage of unexported render helpers.
package render

import (
	"testing"
	"time"
)

type formatCase struct {
	name   string
	format func() string
	want   string
}

func TestFormatMemoryValues(t *testing.T) {
	t.Parallel()

	checkFormatCases(t, []formatCase{
		{name: "negative mib value", format: func() string { return formatMiBValue(-1) }, want: TextNA},
		{name: "mib value", format: func() string { return formatMiBValue(512) }, want: "512 MiB"},
		{name: "gib value", format: func() string { return formatMiBValue(1024) }, want: "1.0 GiB"},
		{name: "negative mib pair", format: func() string { return formatMiBPair(-1, 1024) }, want: TextNA},
		{name: "zero mib pair total", format: func() string { return formatMiBPair(512, 0) }, want: TextNA},
		{name: "mib pair", format: func() string { return formatMiBPair(512, 1024) }, want: "512 / 1024 MiB"},
		{name: "negative kib value", format: func() string { return FormatKiBValue(-1) }, want: TextNA},
		{name: "kib value", format: func() string { return FormatKiBValue(1) }, want: "1 KiB"},
		{name: "kib to mib value", format: func() string { return FormatKiBValue(1024) }, want: "1.0 MiB"},
		{name: "kib to gib value", format: func() string { return FormatKiBValue(1048576) }, want: "1.0 GiB"},
		{name: "negative kib pair", format: func() string { return formatKiBPair(-1, 1024) }, want: TextNA},
		{name: "zero kib pair total", format: func() string { return formatKiBPair(512, 0) }, want: TextNA},
		{name: "kib pair", format: func() string { return formatKiBPair(1024, 1048576) }, want: "1.0 MiB / 1.0 GiB"},
	})
}

func TestFormatMetricValues(t *testing.T) {
	t.Parallel()

	checkFormatCases(t, []formatCase{
		{name: "negative bytes per second", format: func() string { return formatBps(-1) }, want: TextNA},
		{name: "bytes per second", format: func() string { return formatBps(999) }, want: "999 B/s"},
		{name: "kib per second", format: func() string { return formatBps(1024) }, want: "1.0 KiB/s"},
		{name: "negative percent", format: func() string { return percentDisplay(-1) }, want: TextNA},
		{name: "percent", format: func() string { return percentDisplay(42) }, want: "42%"},
		{name: "negative temperature", format: func() string { return tempDisplay(-1) }, want: TextNA},
		{name: "temperature", format: func() string { return tempDisplay(70) }, want: "70C"},
		{name: "nonpositive float", format: func() string { return formatFloat(0) }, want: "0.00"},
		{name: "positive float", format: func() string { return formatFloat(1.235) }, want: "1.24"},
		{name: "negative millis", format: func() string { return FormatMillisValue(-1) }, want: TextNA},
		{name: "millis", format: func() string { return FormatMillisValue(1.234) }, want: "1.23 ms"},
		{name: "negative queue depth", format: func() string { return FormatQueueDepth(-1) }, want: TextNA},
		{name: "queue depth", format: func() string { return FormatQueueDepth(2.345) }, want: "2.35x"},
	})
}

func TestFormatPowerAndClockValues(t *testing.T) {
	t.Parallel()

	checkFormatCases(t, []formatCase{
		{name: "nonpositive power pair draw", format: func() string { return formatPowerPair(0, 100) }, want: TextNA},
		{name: "nonpositive power pair limit", format: func() string { return formatPowerPair(50, 0) }, want: TextNA},
		{name: "power pair", format: func() string { return formatPowerPair(50, 100) }, want: "50.00W / 100.00W"},
		{name: "nonpositive power value", format: func() string { return formatPowerValue(0) }, want: TextNA},
		{name: "power value", format: func() string { return formatPowerValue(12.345) }, want: "12.35W"},
		{name: "clock pair", format: func() string { return formatClockPair(1200, 3600) }, want: "1200 / 3600 MHz"},
		{name: "current clock only", format: func() string { return formatClockPair(1200, 0) }, want: "1200 MHz"},
		{name: "max clock only", format: func() string { return formatClockPair(-1, 3600) }, want: "max 3600 MHz"},
		{name: "missing clock pair", format: func() string { return formatClockPair(-1, 0) }, want: TextNA},
		{name: "missing clock value", format: func() string { return formatClockValue(-1) }, want: TextNA},
		{name: "clock value", format: func() string { return formatClockValue(1200) }, want: "1200 MHz"},
	})
}

func TestFormatDurations(t *testing.T) {
	t.Parallel()

	checkFormatCases(t, []formatCase{
		{name: "missing uptime", format: func() string { return formatUptime(0) }, want: TextNA},
		{name: "subminute uptime", format: func() string { return formatUptime(59) }, want: "0m"},
		{name: "minute uptime", format: func() string { return formatUptime(60) }, want: "1m"},
		{name: "hour uptime", format: func() string { return formatUptime(3660) }, want: "1h 1m"},
		{name: "nonpositive age", format: func() string { return formatAge(0) }, want: "0s"},
		{name: "rounded age seconds", format: func() string { return formatAge(1500 * time.Millisecond) }, want: "2s"},
		{name: "age minutes", format: func() string { return formatAge(75 * time.Second) }, want: "1m"},
		{name: "age hours", format: func() string { return formatAge(2 * time.Hour) }, want: "2h"},
	})
}

func checkFormatCases(t *testing.T, tests []formatCase) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.format()
			if got != tc.want {
				t.Fatalf("format result = %q, want %q", got, tc.want)
			}
		})
	}
}
