package metrics

import core "github.com/lmliam/remote-monitor/internal/core"

const (
	netAutoFloorBps    int64 = 64 * 1024
	netAutoDecayStep   int64 = 32 * 1024
	netDisplayHeadroom int64 = 128 * 1024
	netHeadroomDivisor       = 2
	netDecayDivisor          = 12
)

// UpdateNetCeilings updates rolling network display ceilings from a new sample.
func UpdateNetCeilings(state *core.AppState, smp core.Sample) {
	if state.NetCeilings == nil {
		state.NetCeilings = map[string]int64{}
	}

	active := make(map[string]struct{}, len(smp.Net))
	for _, net := range smp.Net {
		active[net.Iface] = struct{}{}
		peak := NetDisplayCeiling(Max64(net.RXBps, net.TXBps))
		state.NetCeilings[net.Iface] = RollingNetCeiling(state.NetCeilings[net.Iface], peak)
	}

	for iface := range state.NetCeilings {
		if _, ok := active[iface]; !ok {
			delete(state.NetCeilings, iface)
		}
	}
}

// NetGaugeCeiling returns the rolling display ceiling for one network interface.
func NetGaugeCeiling(state core.AppState, net core.NetStat) int64 {
	if ceiling := state.NetCeilings[net.Iface]; ceiling > 0 {
		return ceiling
	}

	return RollingNetCeiling(0, NetDisplayCeiling(Max64(net.RXBps, net.TXBps)))
}

// NetDisplayCeiling adds stable headroom above current network throughput.
func NetDisplayCeiling(current int64) int64 {
	if current < netAutoFloorBps {
		return netAutoFloorBps
	}

	headroom := Max64(current/netHeadroomDivisor, netDisplayHeadroom)
	ceiling := current + headroom
	if ceiling < netAutoFloorBps {
		return netAutoFloorBps
	}

	return ceiling
}

// RollingNetCeiling raises quickly and decays slowly toward current throughput.
func RollingNetCeiling(previous, current int64) int64 {
	if current < netAutoFloorBps {
		current = netAutoFloorBps
	}
	if previous < netAutoFloorBps {
		return current
	}
	if current >= previous {
		return current
	}

	next := max(max(previous-Max64(previous/netDecayDivisor, netAutoDecayStep), current), netAutoFloorBps)

	return next
}
