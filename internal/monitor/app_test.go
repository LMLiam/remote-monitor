package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	monitor "github.com/lmliam/remote-monitor/internal/monitor"
	"testing"
	"time"
)

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
	if got := state.DiskLatencyHistory; len(got) != 1 || got[0] != metrics.DiskLatencyHistoryPercent(smp) {
		t.Fatalf("diskLatencyHistory = %#v", got)
	}
	if got := state.NetIssueHistory; len(got) != 1 || got[0] != metrics.NetIssueHistoryPercent(smp) {
		t.Fatalf("netIssueHistory = %#v", got)
	}
}
