//nolint:testpackage // Issue 58 requires direct in-package coverage of unexported render helpers.
package render

import (
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
)

type severityCase struct {
	name     string
	severity func() string
	want     string
}

func TestSeverityThresholds(t *testing.T) {
	t.Parallel()

	thresholds := core.DefaultThresholds()
	var zeroThresholds core.Thresholds
	checkSeverityCases(t, []severityCase{
		{name: "util zero thresholds uses defaults", severity: func() string { return UtilSeverity(95, zeroThresholds) }, want: severityCritical},
		{name: "util missing", severity: func() string { return UtilSeverity(-1, thresholds) }, want: severityNeutral},
		{name: "util info", severity: func() string { return UtilSeverity(0, thresholds) }, want: severityInfo},
		{name: "util ok", severity: func() string { return UtilSeverity(40, thresholds) }, want: severityOK},
		{name: "util warn", severity: func() string { return UtilSeverity(80, thresholds) }, want: severityWarn},
		{name: "util critical", severity: func() string { return UtilSeverity(95, thresholds) }, want: severityCritical},
		{name: "memory missing", severity: func() string { return memorySeverity(-1, thresholds) }, want: severityNeutral},
		{name: "memory info", severity: func() string { return memorySeverity(59, thresholds) }, want: severityInfo},
		{name: "memory ok", severity: func() string { return memorySeverity(60, thresholds) }, want: severityOK},
		{name: "memory warn", severity: func() string { return memorySeverity(85, thresholds) }, want: severityWarn},
		{name: "memory critical", severity: func() string { return memorySeverity(90, thresholds) }, want: severityCritical},
		{name: "disk missing", severity: func() string { return diskUtilSeverity(-1, thresholds) }, want: severityNeutral},
		{name: "disk info", severity: func() string { return diskUtilSeverity(39, thresholds) }, want: severityInfo},
		{name: "disk ok", severity: func() string { return diskUtilSeverity(40, thresholds) }, want: severityOK},
		{name: "disk warn", severity: func() string { return diskUtilSeverity(60, thresholds) }, want: severityWarn},
		{name: "disk critical", severity: func() string { return diskUtilSeverity(90, thresholds) }, want: severityCritical},
		{name: "disk usage missing", severity: func() string { return diskUsageSeverity(-1, thresholds) }, want: severityNeutral},
		{name: "disk usage info", severity: func() string { return diskUsageSeverity(39, thresholds) }, want: severityInfo},
		{name: "disk usage ok", severity: func() string { return diskUsageSeverity(40, thresholds) }, want: severityOK},
		{name: "disk usage warn", severity: func() string { return diskUsageSeverity(90, thresholds) }, want: severityWarn},
		{name: "disk usage critical", severity: func() string { return diskUsageSeverity(95, thresholds) }, want: severityCritical},
		{name: "disk usage zero thresholds uses defaults", severity: func() string { return diskUsageSeverity(95, zeroThresholds) }, want: severityCritical},
		{name: "availability missing", severity: func() string { return availabilitySeverity(-1, thresholds) }, want: severityNeutral},
		{name: "availability critical", severity: func() string { return availabilitySeverity(5, thresholds) }, want: severityCritical},
		{name: "availability warn", severity: func() string { return availabilitySeverity(15, thresholds) }, want: severityWarn},
		{name: "availability info", severity: func() string { return availabilitySeverity(35, thresholds) }, want: severityInfo},
		{name: "availability ok", severity: func() string { return availabilitySeverity(36, thresholds) }, want: severityOK},
		{name: "availability zero thresholds uses defaults", severity: func() string { return availabilitySeverity(5, zeroThresholds) }, want: severityCritical},
		{name: "temperature missing", severity: func() string { return temperatureSeverity(-1, thresholds) }, want: severityNeutral},
		{name: "temperature info", severity: func() string { return temperatureSeverity(59, thresholds) }, want: severityInfo},
		{name: "temperature ok", severity: func() string { return temperatureSeverity(60, thresholds) }, want: severityOK},
		{name: "temperature warn", severity: func() string { return temperatureSeverity(70, thresholds) }, want: severityWarn},
		{name: "temperature critical", severity: func() string { return temperatureSeverity(80, thresholds) }, want: severityCritical},
		{name: "temperature zero thresholds uses defaults", severity: func() string { return temperatureSeverity(80, zeroThresholds) }, want: severityCritical},
		{name: "cpu temperature missing", severity: func() string { return cpuTemperatureSeverity(-1, thresholds) }, want: severityNeutral},
		{name: "cpu temperature info", severity: func() string { return cpuTemperatureSeverity(59, thresholds) }, want: severityInfo},
		{name: "cpu temperature ok", severity: func() string { return cpuTemperatureSeverity(60, thresholds) }, want: severityOK},
		{name: "cpu temperature warm", severity: func() string { return cpuTemperatureSeverity(75, thresholds) }, want: severityWarn},
		{name: "cpu temperature critical", severity: func() string { return cpuTemperatureSeverity(85, thresholds) }, want: severityCritical},
		{name: "cpu temperature zero thresholds uses defaults", severity: func() string { return cpuTemperatureSeverity(85, zeroThresholds) }, want: severityCritical},
		{name: "vram missing", severity: func() string { return vramSeverity(-1, thresholds) }, want: severityNeutral},
		{name: "vram info", severity: func() string { return vramSeverity(59, thresholds) }, want: severityInfo},
		{name: "vram ok", severity: func() string { return vramSeverity(60, thresholds) }, want: severityOK},
		{name: "vram warn", severity: func() string { return vramSeverity(85, thresholds) }, want: severityWarn},
		{name: "vram critical", severity: func() string { return vramSeverity(95, thresholds) }, want: severityCritical},
		{name: "vram zero thresholds uses defaults", severity: func() string { return vramSeverity(95, zeroThresholds) }, want: severityCritical},
		{name: "power missing", severity: func() string { return powerSeverity(0, 100) }, want: severityNeutral},
		{name: "power info", severity: func() string { return powerSeverity(64, 100) }, want: severityInfo},
		{name: "power ok", severity: func() string { return powerSeverity(65, 100) }, want: severityOK},
		{name: "power warn", severity: func() string { return powerSeverity(90, 100) }, want: severityWarn},
		{name: "power critical", severity: func() string { return powerSeverity(98, 100) }, want: severityCritical},
		{name: "pressure missing", severity: func() string { return psiSeverity(-1) }, want: severityNeutral},
		{name: "pressure info", severity: func() string { return psiSeverity(0.5) }, want: severityInfo},
		{name: "pressure ok", severity: func() string { return psiSeverity(1) }, want: severityOK},
		{name: "pressure warn", severity: func() string { return psiSeverity(5) }, want: severityWarn},
		{name: "pressure critical", severity: func() string { return psiSeverity(20) }, want: severityCritical},
		{name: "disk latency missing", severity: func() string { return diskLatencyHistorySeverity(-1) }, want: severityNeutral},
		{name: "disk latency info", severity: func() string { return diskLatencyHistorySeverity(9) }, want: severityInfo},
		{name: "disk latency ok", severity: func() string { return diskLatencyHistorySeverity(10) }, want: severityOK},
		{name: "disk latency warn", severity: func() string { return diskLatencyHistorySeverity(30) }, want: severityWarn},
		{name: "disk latency critical", severity: func() string { return diskLatencyHistorySeverity(100) }, want: severityCritical},
		{name: "net issues missing", severity: func() string { return netIssueSeverity(-1) }, want: severityNeutral},
		{name: "net issues ok", severity: func() string { return netIssueSeverity(0) }, want: severityOK},
		{name: "net issues info", severity: func() string { return netIssueSeverity(1) }, want: severityInfo},
		{name: "net issues warn", severity: func() string { return netIssueSeverity(20) }, want: severityWarn},
		{name: "net issues critical", severity: func() string { return netIssueSeverity(50) }, want: severityCritical},
	})
}

func TestSeverityUsesCustomThresholds(t *testing.T) {
	t.Parallel()

	thresholds := core.DefaultThresholds()
	thresholds.CPUCriticalPercent = 70
	thresholds.CPUWarnTemp = 45
	thresholds.CPUCriticalTemp = 55
	thresholds.RAMWarnAvailablePercent = 10
	thresholds.RAMCriticalAvailablePercent = 4
	thresholds.GPUWarnTemp = 50
	thresholds.GPUCriticalTemp = 60
	thresholds.VRAMWarnPercent = 70
	thresholds.VRAMCriticalPercent = 80
	thresholds.DiskWarnPercent = 75
	thresholds.DiskCriticalPercent = 85

	checkSeverityCases(t, []severityCase{
		{name: "custom cpu util critical", severity: func() string { return UtilSeverity(75, thresholds) }, want: severityCritical},
		{name: "custom cpu temp warn", severity: func() string { return cpuTemperatureSeverity(50, thresholds) }, want: severityWarn},
		{name: "custom cpu temp critical", severity: func() string { return cpuTemperatureSeverity(60, thresholds) }, want: severityCritical},
		{name: "custom ram availability no longer warn", severity: func() string { return availabilitySeverity(12, thresholds) }, want: severityInfo},
		{name: "custom gpu temperature warn", severity: func() string { return temperatureSeverity(55, thresholds) }, want: severityWarn},
		{name: "custom gpu temperature critical", severity: func() string { return temperatureSeverity(65, thresholds) }, want: severityCritical},
		{name: "custom vram warn", severity: func() string { return vramSeverity(75, thresholds) }, want: severityWarn},
		{name: "custom vram critical", severity: func() string { return vramSeverity(85, thresholds) }, want: severityCritical},
		{name: "custom disk warn", severity: func() string { return diskUsageSeverity(80, thresholds) }, want: severityWarn},
		{name: "custom disk critical", severity: func() string { return diskUsageSeverity(90, thresholds) }, want: severityCritical},
	})
}

func checkSeverityCases(t *testing.T, tests []severityCase) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.severity()
			if got != tc.want {
				t.Fatalf("severity = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSeverityColorsRanksAndMerging(t *testing.T) {
	t.Parallel()

	if got := SeverityColor(severityCritical); got != ansi.Red {
		t.Fatalf("SeverityColor critical = %q, want red", got)
	}
	if got := SeverityColor(severityWarn); got != ansi.Yellow {
		t.Fatalf("SeverityColor warn = %q, want yellow", got)
	}
	if got := SeverityColor(severityOK); got != ansi.Green {
		t.Fatalf("SeverityColor ok = %q, want green", got)
	}
	if got := SeverityColor(severityNeutral); got != ansi.Muted {
		t.Fatalf("SeverityColor neutral = %q, want muted", got)
	}
	if got := severityBackground(severityInfo); got != ansi.BlueBg {
		t.Fatalf("severityBackground info = %q, want blue background", got)
	}
	if got := severityRank(severityCritical); got != severityRankCritical {
		t.Fatalf("severityRank critical = %d, want %d", got, severityRankCritical)
	}
	if got := mergeSeverity(severityInfo, severityWarn); got != severityWarn {
		t.Fatalf("mergeSeverity info/warn = %q, want warn", got)
	}
	if got := mergeSeverity(severityCritical, severityWarn); got != severityCritical {
		t.Fatalf("mergeSeverity critical/warn = %q, want critical", got)
	}
}
