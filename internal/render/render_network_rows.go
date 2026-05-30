package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

// BuildNetworkRows builds network and TCP health rows for the dashboard.
func BuildNetworkRows(state core.AppState, activityWidth int, condensed bool) []TableRowSpec {
	s := state.Current
	rows := make([]TableRowSpec, 0, len(s.Net)*2+1)
	if s.TCPRetransSegsPerSec >= 0 || s.TCPResetsPerSec >= 0 {
		tcpSeverity := tcpHealthSeverity(s.TCPRetransSegsPerSec, s.TCPResetsPerSec)
		rows = append(rows, TableFullRow("TCP Health", SeverityColor(tcpSeverity), "retx "+formatOpsPerSec(s.TCPRetransSegsPerSec), SeverityColor(tcpSeverity), "", "reset "+formatOpsPerSec(s.TCPResetsPerSec), SeverityColor(tcpSeverity), ""))
	}
	for _, net := range s.Net {
		ceiling := metrics.NetGaugeCeiling(state, net)
		rxColor := ansi.Blue
		if net.RXErrors > 0 {
			rxColor = ansi.Red
		} else if net.RXDrops > 0 || net.RXOverruns > 0 {
			rxColor = ansi.Yellow
		}
		txColor := ansi.Cyan
		if net.TXErrors > 0 {
			txColor = ansi.Red
		} else if net.TXDrops > 0 || net.TXOverruns > 0 {
			txColor = ansi.Yellow
		}
		rows = append(rows,
			TableFullRow("Net "+net.Iface+" RX", rxColor, networkValueText(net.RXBps, ceiling, net, net.RXPps, condensed), rxColor, "", "", "", rateGaugeSummaryCell(net.RXBps, ceiling, activityWidth, rxColor, fmt.Sprintf("d%d/e%d/o%d", metrics.Max64(net.RXDrops, 0), metrics.Max64(net.RXErrors, 0), metrics.Max64(net.RXOverruns, 0)))),
			TableFullRow("Net "+net.Iface+" TX", txColor, networkValueText(net.TXBps, ceiling, net, net.TXPps, condensed), txColor, "", "", "", rateGaugeSummaryCell(net.TXBps, ceiling, activityWidth, txColor, fmt.Sprintf("d%d/e%d/o%d", metrics.Max64(net.TXDrops, 0), metrics.Max64(net.TXErrors, 0), metrics.Max64(net.TXOverruns, 0)))),
		)
	}

	return rows
}

func networkValueText(bps, ceiling int64, net core.NetStat, pps int64, condensed bool) string {
	bpsText := strings.ReplaceAll(formatBps(bps), " ", "")
	utilText := strings.ReplaceAll(NetUtilSummary(bps, ceiling, net), " / ", "/")
	if condensed {
		return fmt.Sprintf("%s %s", bpsText, utilText)
	}

	return fmt.Sprintf("%s %s %s", bpsText, utilText, formatPPSCompact(pps))
}

func formatPPSCompact(v int64) string {
	if v < 0 {
		return TextNA
	}
	value := float64(v)
	switch {
	case value >= ppsMillionScale:
		return fmt.Sprintf("%.1fMpps", value/ppsMillionScale)
	case value >= ppsThousandScale:
		return fmt.Sprintf("%.1fkpps", value/ppsThousandScale)
	default:
		return fmt.Sprintf("%dpps", v)
	}
}
