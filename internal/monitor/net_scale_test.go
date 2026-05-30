package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"

	"testing"
)

func TestNetDisplayCeilingAddsHeadroomAboveCurrentTraffic(t *testing.T) {
	t.Parallel()

	current := int64(4 * 1024 * 1024)
	got := metrics.NetDisplayCeiling(current)

	if got <= current {
		t.Fatalf("expected display ceiling to leave headroom above current traffic: got %d current %d", got, current)
	}
}

func TestRollingNetCeilingDecaysInsteadOfSnappingToCurrentSample(t *testing.T) {
	t.Parallel()

	previous := metrics.NetDisplayCeiling(10 * 1024 * 1024)
	current := metrics.NetDisplayCeiling(128 * 1024)
	next := metrics.RollingNetCeiling(previous, current)

	if next >= previous {
		t.Fatalf("expected ceiling to decay below previous peak: %d", next)
	}
	if next <= current {
		t.Fatalf("expected ceiling to remain above current Sample: %d", next)
	}
}

func TestNetGaugeCeilingUsesRollingPerInterfaceValueWhenSpeedUnknown(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.NetCeilings = map[string]int64{
			testIfaceTailscale: 5 * 1024 * 1024,
		}
	})
	net := testNetStat(func(net *core.NetStat) {
		net.Iface = testIfaceTailscale
		net.RXBps = 1024
		net.TXBps = 2048
		net.SpeedMbps = -1
	})
	if got := metrics.NetGaugeCeiling(state, net); got != 5*1024*1024 {
		t.Fatalf("NetGaugeCeiling = %d", got)
	}
}

func TestNetGaugeCeilingUsesRollingPerInterfaceValueEvenWhenLinkSpeedKnown(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.NetCeilings = map[string]int64{
			testIfaceEth0: 8 * 1024 * 1024,
		}
	})
	net := testNetStat(func(net *core.NetStat) {
		net.Iface = testIfaceEth0
		net.RXBps = 1024
		net.TXBps = 2048
		net.SpeedMbps = 10000
	})
	if got := metrics.NetGaugeCeiling(state, net); got != 8*1024*1024 {
		t.Fatalf("NetGaugeCeiling should prefer rolling display ceiling over raw link speed, got %d", got)
	}
}
