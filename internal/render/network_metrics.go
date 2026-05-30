package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
)

func netIssueSummary(s core.Sample) string {
	if len(s.Net) == 0 {
		return TextNA
	}
	drops, errors := metrics.NetIssueTotals(s)

	return NetDirectionHealthSummary(drops, errors)
}

func netIssueSeverity(v int) string {
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= netIssueCritical:
		return severityCritical
	case v >= netIssueWarn:
		return severityWarn
	case v > 0:
		return severityInfo
	default:
		return severityOK
	}
}

// FormatLinkSpeed renders an interface link speed for display.
func FormatLinkSpeed(mbps int) string {
	if mbps <= 0 {
		return "link n/a"
	}
	if mbps >= decimalKiloScale {
		return fmt.Sprintf("%.1f Gbps", float64(mbps)/decimalKiloScale)
	}

	return fmt.Sprintf("%d Mbps", mbps)
}

func tcpHealthSeverity(retransPerSec, resetsPerSec int64) string {
	switch {
	case retransPerSec < 0 || resetsPerSec < 0:
		return severityNeutral
	case resetsPerSec >= tcpResetCriticalPPS || retransPerSec >= tcpRetransCriticalPPS:
		return severityCritical
	case resetsPerSec >= 1 || retransPerSec >= tcpRetransWarnPPS:
		return severityWarn
	case retransPerSec > 0:
		return severityInfo
	default:
		return "ok"
	}
}

// NetUtilSummary summarizes utilization against link speed or an automatic ceiling.
func NetUtilSummary(value, ceiling int64, net core.NetStat) string {
	if value < 0 {
		if net.SpeedMbps > 0 {
			return "n/a / " + formatLinkSpeedCompact(net.SpeedMbps)
		}

		return TextNA
	}
	if linkBps := metrics.NetLinkBps(net); linkBps > 0 {
		return fmt.Sprintf("%s / %s", percentDisplay(metrics.RatePercent(value, linkBps)), formatLinkSpeedCompact(net.SpeedMbps))
	}

	return percentDisplay(metrics.RatePercent(value, ceiling)) + " auto"
}

func formatLinkSpeedCompact(mbps int) string {
	if mbps <= 0 {
		return "link?"
	}
	if mbps >= decimalKiloScale {
		return fmt.Sprintf("%.1fG", float64(mbps)/decimalKiloScale)
	}

	return fmt.Sprintf("%dM", mbps)
}

// NetDirectionHealthSummary formats network drops and errors as a compact health signal.
func NetDirectionHealthSummary(drops, errors int64) string {
	switch {
	case drops < 0 || errors < 0:
		return "d?/e?"
	default:
		return fmt.Sprintf("d%d/e%d", drops, errors)
	}
}
