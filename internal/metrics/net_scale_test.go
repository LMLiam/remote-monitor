package metrics_test

import (
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
)

func TestNetDisplayCeilingAddsFloorAndHeadroom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current int64
		want    int64
	}{
		{name: "floor", current: 0, want: 64 * 1024},
		{name: "minimum headroom", current: 64 * 1024, want: 192 * 1024},
		{name: "proportional headroom", current: 512000, want: 768000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := metrics.NetDisplayCeiling(tc.current); got != tc.want {
				t.Fatalf("NetDisplayCeiling(%d) = %d, want %d", tc.current, got, tc.want)
			}
		})
	}
}

func TestRollingNetCeilingRaisesAndDecays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		previous int64
		current  int64
		want     int64
	}{
		{name: "initial floor", previous: 0, current: 0, want: 64 * 1024},
		{name: "raises immediately", previous: 64 * 1024, current: 200000, want: 200000},
		{name: "decays by divisor", previous: 1200000, current: 64 * 1024, want: 1100000},
		{name: "does not decay below floor", previous: 70000, current: 0, want: 64 * 1024},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := metrics.RollingNetCeiling(tc.previous, tc.current); got != tc.want {
				t.Fatalf("RollingNetCeiling(%d, %d) = %d, want %d", tc.previous, tc.current, got, tc.want)
			}
		})
	}
}

func TestUpdateNetCeilingsInitializesGrowsDecaysAndDeletes(t *testing.T) {
	t.Parallel()

	var state core.AppState
	first := core.EmptySample()
	first.Net = []core.NetStat{
		testNetStat(testIfaceEth0, 200000, 100000, 0, 0, -1, 0, 0, 0, 0, 0, 0),
		testNetStat(testIfaceWlan0, 0, 70000, 0, 0, -1, 0, 0, 0, 0, 0, 0),
	}

	metrics.UpdateNetCeilings(&state, first)
	if state.NetCeilings == nil {
		t.Fatal("UpdateNetCeilings left NetCeilings nil")
	}
	if got, want := state.NetCeilings[testIfaceEth0], metrics.NetDisplayCeiling(200000); got != want {
		t.Fatalf("eth0 ceiling after first sample = %d, want %d", got, want)
	}
	if got, want := state.NetCeilings[testIfaceWlan0], metrics.NetDisplayCeiling(70000); got != want {
		t.Fatalf("wlan0 ceiling after first sample = %d, want %d", got, want)
	}

	second := core.EmptySample()
	second.Net = []core.NetStat{
		testNetStat(testIfaceEth0, 0, 0, 0, 0, -1, 0, 0, 0, 0, 0, 0),
	}
	previousEth0 := state.NetCeilings[testIfaceEth0]
	metrics.UpdateNetCeilings(&state, second)

	wantEth0 := metrics.RollingNetCeiling(previousEth0, metrics.NetDisplayCeiling(0))
	if got := state.NetCeilings[testIfaceEth0]; got != wantEth0 {
		t.Fatalf("eth0 ceiling after decay = %d, want %d", got, wantEth0)
	}
	if _, ok := state.NetCeilings[testIfaceWlan0]; ok {
		t.Fatalf("wlan0 ceiling still present after inactive sample: %#v", state.NetCeilings)
	}
}

func TestNetGaugeCeilingUsesStoredOrComputed(t *testing.T) {
	t.Parallel()

	var state core.AppState
	state.NetCeilings = map[string]int64{
		testIfaceEth0: 999,
	}
	stored := testNetStat(testIfaceEth0, 200000, 100000, 0, 0, -1, 0, 0, 0, 0, 0, 0)
	computed := testNetStat(testIfaceWlan0, 200000, 100000, 0, 0, -1, 0, 0, 0, 0, 0, 0)

	if got := metrics.NetGaugeCeiling(state, stored); got != 999 {
		t.Fatalf("NetGaugeCeiling stored = %d, want 999", got)
	}
	if got, want := metrics.NetGaugeCeiling(state, computed), metrics.NetDisplayCeiling(200000); got != want {
		t.Fatalf("NetGaugeCeiling computed = %d, want %d", got, want)
	}
}
