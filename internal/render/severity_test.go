//nolint:testpackage // Issue 58 requires direct in-package coverage of unexported render helpers.
package render

import (
	"testing"

	"github.com/lmliam/remote-monitor/internal/render/ansi"
)

type severityCase struct {
	name string
	got  string
	want string
}

func TestSeverityThresholds(t *testing.T) {
	t.Parallel()

	checkSeverityCases(t, []severityCase{
		{name: "util missing", got: UtilSeverity(-1), want: severityNeutral},
		{name: "util info", got: UtilSeverity(0), want: severityInfo},
		{name: "util ok", got: UtilSeverity(40), want: severityOK},
		{name: "util warn", got: UtilSeverity(80), want: severityWarn},
		{name: "util critical", got: UtilSeverity(95), want: severityCritical},
		{name: "memory missing", got: memorySeverity(-1), want: severityNeutral},
		{name: "memory info", got: memorySeverity(59), want: severityInfo},
		{name: "memory ok", got: memorySeverity(60), want: severityOK},
		{name: "memory warn", got: memorySeverity(85), want: severityWarn},
		{name: "memory critical", got: memorySeverity(90), want: severityCritical},
		{name: "disk missing", got: diskUtilSeverity(-1), want: severityNeutral},
		{name: "disk info", got: diskUtilSeverity(39), want: severityInfo},
		{name: "disk ok", got: diskUtilSeverity(40), want: severityOK},
		{name: "disk warn", got: diskUtilSeverity(60), want: severityWarn},
		{name: "disk critical", got: diskUtilSeverity(90), want: severityCritical},
		{name: "availability missing", got: availabilitySeverity(-1), want: severityNeutral},
		{name: "availability critical", got: availabilitySeverity(5), want: severityCritical},
		{name: "availability warn", got: availabilitySeverity(15), want: severityWarn},
		{name: "availability info", got: availabilitySeverity(35), want: severityInfo},
		{name: "availability ok", got: availabilitySeverity(36), want: severityOK},
		{name: "temperature missing", got: temperatureSeverity(-1), want: severityNeutral},
		{name: "temperature info", got: temperatureSeverity(59), want: severityInfo},
		{name: "temperature ok", got: temperatureSeverity(60), want: severityOK},
		{name: "temperature warn", got: temperatureSeverity(70), want: severityWarn},
		{name: "temperature critical", got: temperatureSeverity(80), want: severityCritical},
		{name: "power missing", got: powerSeverity(0, 100), want: severityNeutral},
		{name: "power info", got: powerSeverity(64, 100), want: severityInfo},
		{name: "power ok", got: powerSeverity(65, 100), want: severityOK},
		{name: "power warn", got: powerSeverity(90, 100), want: severityWarn},
		{name: "power critical", got: powerSeverity(98, 100), want: severityCritical},
		{name: "pressure missing", got: psiSeverity(-1), want: severityNeutral},
		{name: "pressure info", got: psiSeverity(0.5), want: severityInfo},
		{name: "pressure ok", got: psiSeverity(1), want: severityOK},
		{name: "pressure warn", got: psiSeverity(5), want: severityWarn},
		{name: "pressure critical", got: psiSeverity(20), want: severityCritical},
		{name: "disk latency missing", got: diskLatencyHistorySeverity(-1), want: severityNeutral},
		{name: "disk latency info", got: diskLatencyHistorySeverity(9), want: severityInfo},
		{name: "disk latency ok", got: diskLatencyHistorySeverity(10), want: severityOK},
		{name: "disk latency warn", got: diskLatencyHistorySeverity(30), want: severityWarn},
		{name: "disk latency critical", got: diskLatencyHistorySeverity(100), want: severityCritical},
		{name: "net issues missing", got: netIssueSeverity(-1), want: severityNeutral},
		{name: "net issues ok", got: netIssueSeverity(0), want: severityOK},
		{name: "net issues info", got: netIssueSeverity(1), want: severityInfo},
		{name: "net issues warn", got: netIssueSeverity(20), want: severityWarn},
		{name: "net issues critical", got: netIssueSeverity(50), want: severityCritical},
	})
}

func checkSeverityCases(t *testing.T, tests []severityCase) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.got != tc.want {
				t.Fatalf("severity = %q, want %q", tc.got, tc.want)
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
