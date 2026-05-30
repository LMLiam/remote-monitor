package metrics

import core "github.com/lmliam/remote-monitor/internal/core"

const (
	linkBytesPerMegabit = 125000
	netIssueDropWeight  = 10
	netIssueErrorWeight = 20
)

// TotalNetRXBps returns summed receive throughput across interfaces.
func TotalNetRXBps(s core.Sample) int64 {
	var total int64
	for _, net := range s.Net {
		if net.RXBps > 0 {
			total += net.RXBps
		}
	}

	return total
}

// TotalNetTXBps returns summed transmit throughput across interfaces.
func TotalNetTXBps(s core.Sample) int64 {
	var total int64
	for _, net := range s.Net {
		if net.TXBps > 0 {
			total += net.TXBps
		}
	}

	return total
}

// NetIssueTotals returns accumulated network drops and errors across interfaces.
func NetIssueTotals(s core.Sample) (drops, errors int64) {
	for _, net := range s.Net {
		if net.RXDrops > 0 {
			drops += net.RXDrops
		}
		if net.TXDrops > 0 {
			drops += net.TXDrops
		}
		if net.RXErrors > 0 {
			errors += net.RXErrors
		}
		if net.TXErrors > 0 {
			errors += net.TXErrors
		}
	}

	return drops, errors
}

// NetIssueHistoryPercent converts network drops and errors into a history value.
func NetIssueHistoryPercent(s core.Sample) int {
	if len(s.Net) == 0 {
		return -1
	}
	drops, errors := NetIssueTotals(s)

	return Clamp(int(drops*netIssueDropWeight+errors*netIssueErrorWeight), percentMin, percentMax)
}

// NetLinkBps converts an interface link speed from megabits to bytes per second.
func NetLinkBps(net core.NetStat) int64 {
	if net.SpeedMbps <= 0 {
		return -1
	}

	return int64(net.SpeedMbps) * linkBytesPerMegabit
}
