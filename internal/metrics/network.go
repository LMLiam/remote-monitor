package metrics

import (
	"path"

	core "github.com/lmliam/remote-monitor/internal/core"
)

const (
	linkBytesPerMegabit = 125000
	netIssueDropWeight  = 10
	netIssueErrorWeight = 20

	// NetAggregateInterface is the synthetic interface name used for aggregate network output.
	NetAggregateInterface = "aggregate"
)

// SelectNetStats applies include/exclude interface patterns and optional aggregation.
func SelectNetStats(nets []core.NetStat, includePatterns, excludePatterns []string, aggregate bool) []core.NetStat {
	selected := make([]core.NetStat, 0, len(nets))
	for _, net := range nets {
		if len(includePatterns) > 0 && !ifaceMatchesAny(net.Iface, includePatterns) {
			continue
		}
		if ifaceMatchesAny(net.Iface, excludePatterns) {
			continue
		}
		selected = append(selected, net)
	}

	if !aggregate || len(selected) == 0 {
		return selected
	}

	return []core.NetStat{aggregateNetStats(selected)}
}

func ifaceMatchesAny(iface string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := path.Match(pattern, iface)
		if err == nil && matched {
			return true
		}
	}

	return false
}

func aggregateNetStats(nets []core.NetStat) core.NetStat {
	return core.NetStat{
		Iface:      NetAggregateInterface,
		RXBps:      sumKnownInt64(nets, func(net core.NetStat) int64 { return net.RXBps }),
		TXBps:      sumKnownInt64(nets, func(net core.NetStat) int64 { return net.TXBps }),
		RXPps:      sumKnownInt64(nets, func(net core.NetStat) int64 { return net.RXPps }),
		TXPps:      sumKnownInt64(nets, func(net core.NetStat) int64 { return net.TXPps }),
		SpeedMbps:  sumPositiveInt(nets, func(net core.NetStat) int { return net.SpeedMbps }),
		RXDrops:    sumKnownInt64(nets, func(net core.NetStat) int64 { return net.RXDrops }),
		RXErrors:   sumKnownInt64(nets, func(net core.NetStat) int64 { return net.RXErrors }),
		RXOverruns: sumKnownInt64(nets, func(net core.NetStat) int64 { return net.RXOverruns }),
		TXDrops:    sumKnownInt64(nets, func(net core.NetStat) int64 { return net.TXDrops }),
		TXErrors:   sumKnownInt64(nets, func(net core.NetStat) int64 { return net.TXErrors }),
		TXOverruns: sumKnownInt64(nets, func(net core.NetStat) int64 { return net.TXOverruns }),
	}
}

func sumKnownInt64(nets []core.NetStat, value func(core.NetStat) int64) int64 {
	var total int64
	seen := false
	for _, net := range nets {
		v := value(net)
		if v >= 0 {
			total += v
			seen = true
		}
	}
	if !seen {
		return -1
	}

	return total
}

func sumPositiveInt(nets []core.NetStat, value func(core.NetStat) int) int {
	total := 0
	for _, net := range nets {
		v := value(net)
		if v > 0 {
			total += v
		}
	}
	if total == 0 {
		return -1
	}

	return total
}

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
