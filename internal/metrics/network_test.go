package metrics_test

import (
	"reflect"
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
)

const (
	testIfaceEth0  = "eth0"
	testIfaceEth1  = "eth1"
	testIfaceWlan0 = "wlan0"
)

func TestSelectNetStatsDefaultPreservesInterfaces(t *testing.T) {
	t.Parallel()

	nets := testNetStats()
	got := metrics.SelectNetStats(nets, nil, nil, false)

	if !reflect.DeepEqual(got, nets) {
		t.Fatalf("default selection = %#v, want %#v", got, nets)
	}
}

func TestSelectNetStatsIncludesExactNamesAndGlobs(t *testing.T) {
	t.Parallel()

	got := metrics.SelectNetStats(testNetStats(), []string{testIfaceEth0, "wlan*"}, nil, false)

	assertIfaces(t, got, []string{testIfaceEth0, testIfaceWlan0})
}

func TestSelectNetStatsExcludesExactNamesAndGlobs(t *testing.T) {
	t.Parallel()

	got := metrics.SelectNetStats(testNetStats(), nil, []string{"lo", "docker*", "br-*", "eth1", "en*"}, false)

	assertIfaces(t, got, []string{testIfaceEth0, testIfaceWlan0})
}

func TestSelectNetStatsAppliesIncludeBeforeExclude(t *testing.T) {
	t.Parallel()

	got := metrics.SelectNetStats(testNetStats(), []string{"eth*", "en*"}, []string{testIfaceEth1}, false)

	assertIfaces(t, got, []string{testIfaceEth0, "enp5s0"})
}

func TestSelectNetStatsAllowsMissingMatches(t *testing.T) {
	t.Parallel()

	got := metrics.SelectNetStats(testNetStats(), []string{"wg*"}, nil, false)

	if len(got) != 0 {
		t.Fatalf("missing include matches = %#v, want empty selection", got)
	}
}

func TestSelectNetStatsAggregatesSelectedInterfaces(t *testing.T) {
	t.Parallel()

	got := metrics.SelectNetStats(testNetStats(), []string{testIfaceEth0, testIfaceWlan0}, nil, true)

	if len(got) != 1 {
		t.Fatalf("aggregate rows = %#v, want one row", got)
	}
	agg := got[0]
	if agg.Iface != metrics.NetAggregateInterface {
		t.Fatalf("aggregate iface = %q", agg.Iface)
	}
	if agg.RXBps != 300 || agg.TXBps != 70 || agg.RXPps != 30 || agg.TXPps != 7 {
		t.Fatalf("aggregate rates = %#v", agg)
	}
	if agg.SpeedMbps != 1100 {
		t.Fatalf("aggregate speed = %d", agg.SpeedMbps)
	}
	if agg.RXDrops != 3 || agg.RXErrors != 1 || agg.RXOverruns != 2 || agg.TXDrops != 1 || agg.TXErrors != 4 || agg.TXOverruns != 1 {
		t.Fatalf("aggregate issue counters = %#v", agg)
	}
}

func TestSelectNetStatsAggregatesMissingSelectionToEmpty(t *testing.T) {
	t.Parallel()

	got := metrics.SelectNetStats(testNetStats(), []string{"wg*"}, nil, true)

	if len(got) != 0 {
		t.Fatalf("aggregate missing selection = %#v, want empty selection", got)
	}
}

func testNetStats() []core.NetStat {
	return []core.NetStat{
		testNetStat("lo", 1, 1, 1, 1, -1, 0, 0, 0, 0, 0, 0),
		testNetStat(testIfaceEth0, 100, 50, 10, 5, 1000, 2, 1, 0, 1, 0, 0),
		testNetStat(testIfaceEth1, 400, 100, 40, 10, 1000, 0, 0, 0, 0, 0, 0),
		testNetStat(testIfaceWlan0, 200, 20, 20, 2, 100, 1, 0, 2, 0, 4, 1),
		testNetStat("docker0", 800, 70, 80, 7, -1, 3, 0, 0, 0, 0, 0),
		testNetStat("br-lan", 900, 90, 90, 9, -1, 0, 0, 0, 0, 0, 0),
		testNetStat("enp5s0", 50, 15, 5, 2, 2500, 0, 0, 0, 0, 0, 0),
	}
}

func testNetStat(iface string, rxBps, txBps, rxPps, txPps int64, speedMbps int, rxDrops, rxErrors, rxOverruns, txDrops, txErrors, txOverruns int64) core.NetStat {
	return core.NetStat{
		Iface:      iface,
		RXBps:      rxBps,
		TXBps:      txBps,
		RXPps:      rxPps,
		TXPps:      txPps,
		SpeedMbps:  speedMbps,
		RXDrops:    rxDrops,
		RXErrors:   rxErrors,
		RXOverruns: rxOverruns,
		TXDrops:    txDrops,
		TXErrors:   txErrors,
		TXOverruns: txOverruns,
	}
}

func assertIfaces(t *testing.T, got []core.NetStat, want []string) {
	t.Helper()

	gotIfaces := make([]string, 0, len(got))
	for _, net := range got {
		gotIfaces = append(gotIfaces, net.Iface)
	}
	if !reflect.DeepEqual(gotIfaces, want) {
		t.Fatalf("interfaces = %#v, want %#v", gotIfaces, want)
	}
}
