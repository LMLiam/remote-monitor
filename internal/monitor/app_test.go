package monitor_test

import (
	"bytes"
	"context"
	"strings"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	monitor "github.com/lmliam/remote-monitor/internal/monitor"
	"github.com/lmliam/remote-monitor/internal/version"
	"testing"
	"time"
)

func TestRunPrintsVersionWithoutHost(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := monitor.Run(context.Background(), []string{"-version"}, &out); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != version.Current().String() {
		t.Fatalf("version output = %q", got)
	}
}

func TestRenderIntervalUsesConfiguredFPS(t *testing.T) {
	t.Parallel()

	if got := monitor.RenderInterval(testConfig(func(cfg *core.Config) { cfg.RenderFPS = 20 }), true); got != 50*time.Millisecond {
		t.Fatalf("RenderInterval tty = %s", got)
	}
	if got := monitor.RenderInterval(testConfig(func(cfg *core.Config) { cfg.RenderFPS = 20 }), false); got != time.Second {
		t.Fatalf("RenderInterval non-tty = %s", got)
	}
}

func TestApplyEventIgnoresOlderBufferedEventAfterSample(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) { cfg.HistoryLimit = 30 })
	})
	sampleTime := time.Unix(10, 0)

	monitor.ApplySample(&state, testSample(func(smp *core.Sample) { smp.CPUPercent = 7; smp.ReceivedAt = sampleTime }))
	monitor.ApplyEvent(&state, testStreamEvent(func(ev *core.StreamEvent) {
		ev.State = core.StatusConnecting
		ev.Detail = core.DetailOpeningSSHSession
		ev.At = sampleTime.Add(-time.Millisecond)
	}))

	if state.RuntimeState != core.StatusLive {
		t.Fatalf("stale event overwrote live Sample state: %q", state.RuntimeState)
	}
	if state.RuntimeDetail != core.DetailStreamHealthy {
		t.Fatalf("stale event overwrote runtime detail: %q", state.RuntimeDetail)
	}
}

func TestDrainPendingAppliesLatestBufferedSample(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) { cfg.HistoryLimit = 30 })
	})
	sampleCh := make(chan core.Sample, 4)
	eventCh := make(chan core.StreamEvent, 4)
	base := time.Unix(20, 0)

	eventCh <- testStreamEvent(func(ev *core.StreamEvent) {
		ev.State = core.StatusConnecting
		ev.Detail = core.DetailOpeningSSHSession
		ev.At = base
	})
	sampleCh <- testSample(func(smp *core.Sample) { smp.CPUPercent = 11; smp.ReceivedAt = base.Add(time.Millisecond) })
	sampleCh <- testSample(func(smp *core.Sample) { smp.CPUPercent = 22; smp.ReceivedAt = base.Add(2 * time.Millisecond) })

	if !monitor.DrainPending(&state, sampleCh, eventCh) {
		t.Fatal("expected DrainPending to report a Sample update")
	}
	if state.Current.CPUPercent != 22 {
		t.Fatalf("latest Sample not applied: %#v", state.Current)
	}
	if state.RuntimeState != core.StatusLive {
		t.Fatalf("expected latest Sample to leave runtime live, got %q", state.RuntimeState)
	}
	if state.SampleCount != 2 {
		t.Fatalf("expected both samples to be applied to history/state, got %d", state.SampleCount)
	}
}

func TestDrainPendingKeepsNewerDisconnectEvent(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) { cfg.HistoryLimit = 30 })
	})
	sampleCh := make(chan core.Sample, 2)
	eventCh := make(chan core.StreamEvent, 2)
	base := time.Unix(30, 0)

	sampleCh <- testSample(func(smp *core.Sample) { smp.CPUPercent = 22; smp.ReceivedAt = base.Add(time.Millisecond) })
	eventCh <- testStreamEvent(func(ev *core.StreamEvent) {
		ev.State = core.StatusDisconnected
		ev.Detail = core.DetailSSHStreamEnded
		ev.StreamAlive = false
		ev.At = base.Add(2 * time.Millisecond)
	})

	monitor.DrainPending(&state, sampleCh, eventCh)

	if state.RuntimeState != core.StatusDisconnected {
		t.Fatalf("newer disconnect event should win over older Sample, got %q", state.RuntimeState)
	}
	if state.RuntimeDetail != core.DetailSSHStreamEnded {
		t.Fatalf("disconnect detail not preserved: %q", state.RuntimeDetail)
	}
}

func TestApplySampleAppendsExpandedHistorySeries(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) { cfg.HistoryLimit = 30 })
	})
	smp := testSample(func(smp *core.Sample) {
		smp.CPUPercent = 88
		smp.CPUCores = 12
		smp.CPUFreqMHz = 3680
		smp.CPUMaxFreqMHz = 4700
		smp.CPUTempC = 66
		smp.RAMUsedMiB = 2455
		smp.RAMTotalMiB = 15967
		smp.RAMAvailableMiB = 13512
		smp.DiskUtil = 12
		smp.DiskAwaitMS = 1.37
		smp.DiskQueueDepth = 0.21
		smp.Disks = []core.DiskStat{
			testDiskStat(func(disk *core.DiskStat) {
				disk.Device = testNVMeDiskDevice
				disk.Util = 63
				disk.AwaitMS = 2.4
				disk.QueueDepth = 0.4
			}),
		}
		smp.Net = []core.NetStat{
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceEth0
				net.RXBps = 125000
				net.TXBps = 24000
				net.SpeedMbps = 1000
				net.RXDrops = 2
				net.RXErrors = 1
			}),
		}
		smp.GPUs = []core.GPUStat{testGPUStat(func(gpu *core.GPUStat) {
			gpu.Util = 7
			gpu.MemUsed = 4204
			gpu.MemTotal = 12288
			gpu.Temp = 64
			gpu.PowerDraw = 103.12
			gpu.PowerLimit = 170
		})}
		smp.ReceivedAt = time.Unix(40, 0)
	})

	monitor.ApplySample(&state, smp)

	if got := state.CPUFreqHistory; len(got) != 1 || got[0] != metrics.ClockPercent(smp.CPUFreqMHz, smp.CPUMaxFreqMHz) {
		t.Fatalf("cpuFreqHistory = %#v", got)
	}
	if got := state.CPUTempHistory; len(got) != 1 || got[0] != smp.CPUTempC {
		t.Fatalf("cpuTempHistory = %#v", got)
	}
	if got := state.RAMAvailHistory; len(got) != 1 || got[0] != metrics.RAMAvailablePercent(smp) {
		t.Fatalf("ramAvailHistory = %#v", got)
	}
	if got := state.DiskHistory; len(got) != 1 || got[0] != metrics.DiskUtilPercent(smp) {
		t.Fatalf("diskHistory = %#v", got)
	}
	if got := state.DiskLatencyHistory; len(got) != 1 || got[0] != metrics.DiskLatencyHistoryPercent(smp) {
		t.Fatalf("diskLatencyHistory = %#v", got)
	}
	if got := state.NetIssueHistory; len(got) != 1 || got[0] != metrics.NetIssueHistoryPercent(smp) {
		t.Fatalf("netIssueHistory = %#v", got)
	}
}

func TestApplySampleSelectsNetworkInterfacesBeforeHistory(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) {
			cfg.HistoryLimit = 30
			cfg.NetIncludePatterns = []string{"eth*", "wlan*"}
			cfg.NetExcludePatterns = []string{"eth1"}
		})
	})
	smp := testSample(func(smp *core.Sample) {
		smp.Net = []core.NetStat{
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceEth0
				net.RXBps = 100
				net.TXBps = 20
			}),
			testNetStat(func(net *core.NetStat) {
				net.Iface = "eth1"
				net.RXBps = 1000
				net.TXBps = 200
			}),
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceTailscale
				net.RXBps = 500
				net.TXBps = 50
			}),
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceWlan0
				net.RXBps = 300
				net.TXBps = 30
			}),
		}
		smp.ReceivedAt = time.Unix(50, 0)
	})

	monitor.ApplySample(&state, smp)

	if got := state.Current.Net; len(got) != 2 || got[0].Iface != testIfaceEth0 || got[1].Iface != testIfaceWlan0 {
		t.Fatalf("selected current net = %#v", got)
	}
	if got := state.NetRXHistory; len(got) != 1 || got[0] != 400 {
		t.Fatalf("selected net RX history = %#v", got)
	}
	if got := state.NetTXHistory; len(got) != 1 || got[0] != 50 {
		t.Fatalf("selected net TX history = %#v", got)
	}
}

func TestApplySampleAggregatesSelectedNetworkInterfaces(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) {
			cfg.HistoryLimit = 30
			cfg.NetIncludePatterns = []string{testIfaceEth0, testIfaceWlan0}
			cfg.NetAggregate = true
		})
	})
	smp := testSample(func(smp *core.Sample) {
		smp.Net = []core.NetStat{
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceEth0
				net.RXBps = 100
				net.TXBps = 20
				net.SpeedMbps = 1000
			}),
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceWlan0
				net.RXBps = 300
				net.TXBps = 30
				net.SpeedMbps = 100
			}),
			testNetStat(func(net *core.NetStat) {
				net.Iface = "docker0"
				net.RXBps = 1000
				net.TXBps = 200
			}),
		}
		smp.ReceivedAt = time.Unix(60, 0)
	})

	monitor.ApplySample(&state, smp)

	if got := state.Current.Net; len(got) != 1 || got[0].Iface != metrics.NetAggregateInterface || got[0].RXBps != 400 || got[0].TXBps != 50 || got[0].SpeedMbps != 1100 {
		t.Fatalf("aggregate current net = %#v", got)
	}
	if got := state.NetRXHistory; len(got) != 1 || got[0] != 400 {
		t.Fatalf("aggregate net RX history = %#v", got)
	}
}
