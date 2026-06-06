package metrics_test

import (
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
)

func TestSelectNetStatsAggregatesUnknownSentinels(t *testing.T) {
	t.Parallel()

	nets := []core.NetStat{
		testNetStat(testIfaceEth0, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1),
		testNetStat(testIfaceWlan0, -1, -1, -1, -1, 0, -1, -1, -1, -1, -1, -1),
	}

	got := metrics.SelectNetStats(nets, nil, nil, true)
	if len(got) != 1 {
		t.Fatalf("aggregate rows = %#v, want one row", got)
	}

	agg := got[0]
	if agg.Iface != metrics.NetAggregateInterface {
		t.Fatalf("aggregate iface = %q, want %q", agg.Iface, metrics.NetAggregateInterface)
	}
	if agg.RXBps != -1 || agg.TXBps != -1 || agg.RXPps != -1 || agg.TXPps != -1 {
		t.Fatalf("aggregate rates = %#v, want unknown sentinels", agg)
	}
	if agg.SpeedMbps != -1 {
		t.Fatalf("aggregate speed = %d, want -1", agg.SpeedMbps)
	}
	if agg.RXDrops != -1 || agg.RXErrors != -1 || agg.RXOverruns != -1 ||
		agg.TXDrops != -1 || agg.TXErrors != -1 || agg.TXOverruns != -1 {
		t.Fatalf("aggregate issue counters = %#v, want unknown sentinels", agg)
	}
}

func TestTotalNetBpsIgnoresSentinelsAndZero(t *testing.T) {
	t.Parallel()

	smp := core.EmptySample()
	smp.Net = []core.NetStat{
		testNetStat(testIfaceEth0, -1, -1, 0, 0, -1, 0, 0, 0, 0, 0, 0),
		testNetStat(testIfaceEth1, 0, 10, 0, 0, -1, 0, 0, 0, 0, 0, 0),
		testNetStat(testIfaceWlan0, 40, 0, 0, 0, -1, 0, 0, 0, 0, 0, 0),
		testNetStat("enp5s0", 60, 20, 0, 0, -1, 0, 0, 0, 0, 0, 0),
	}

	if got := metrics.TotalNetRXBps(smp); got != 100 {
		t.Fatalf("TotalNetRXBps = %d, want 100", got)
	}
	if got := metrics.TotalNetTXBps(smp); got != 30 {
		t.Fatalf("TotalNetTXBps = %d, want 30", got)
	}
}

func TestNetIssueTotalsAndHistoryPercent(t *testing.T) {
	t.Parallel()

	empty := core.EmptySample()
	if got := metrics.NetIssueHistoryPercent(empty); got != -1 {
		t.Fatalf("NetIssueHistoryPercent empty = %d, want -1", got)
	}

	quiet := core.EmptySample()
	quiet.Net = []core.NetStat{
		testNetStat(testIfaceEth0, 0, 0, 0, 0, -1, -1, 0, 12, 0, -1, 8),
	}
	drops, errors := metrics.NetIssueTotals(quiet)
	if drops != 0 || errors != 0 {
		t.Fatalf("NetIssueTotals quiet = (%d, %d), want (0, 0)", drops, errors)
	}
	if got := metrics.NetIssueHistoryPercent(quiet); got != 0 {
		t.Fatalf("NetIssueHistoryPercent quiet = %d, want 0", got)
	}

	noisy := core.EmptySample()
	noisy.Net = []core.NetStat{
		testNetStat(testIfaceEth0, 0, 0, 0, 0, -1, 2, 1, 50, 1, 0, 50),
		testNetStat(testIfaceWlan0, 0, 0, 0, 0, -1, -1, 0, 50, 10, 4, 50),
	}
	drops, errors = metrics.NetIssueTotals(noisy)
	if drops != 13 || errors != 5 {
		t.Fatalf("NetIssueTotals noisy = (%d, %d), want (13, 5)", drops, errors)
	}
	if got := metrics.NetIssueHistoryPercent(noisy); got != 100 {
		t.Fatalf("NetIssueHistoryPercent noisy = %d, want 100", got)
	}
}

func TestNetLinkBpsConvertsMegabitsAndSentinels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		net  core.NetStat
		want int64
	}{
		{name: "unknown", net: testNetStat(testIfaceEth0, 0, 0, 0, 0, -1, 0, 0, 0, 0, 0, 0), want: -1},
		{name: "zero", net: testNetStat(testIfaceEth0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0), want: -1},
		{name: "one megabit", net: testNetStat(testIfaceEth0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0), want: 125000},
		{name: "gigabit", net: testNetStat(testIfaceEth0, 0, 0, 0, 0, 1000, 0, 0, 0, 0, 0, 0), want: 125000000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := metrics.NetLinkBps(tc.net); got != tc.want {
				t.Fatalf("NetLinkBps(%d Mbps) = %d, want %d", tc.net.SpeedMbps, got, tc.want)
			}
		})
	}
}
