package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	monitor "github.com/lmliam/remote-monitor/internal/monitor"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"math"
	"strings"
	"testing"
	"time"
)

func TestTUIModelStartsAtTopOfDashboard(t *testing.T) {
	t.Parallel()

	model := monitor.NewTUIModel(testTUIState())

	if got := model.Viewport.YOffset(); got != 0 {
		t.Fatalf("viewport started at y offset %d, want 0", got)
	}

	lines := strings.Split(ansi.StripANSI(model.Viewport.View()), "\n")
	top := strings.Join(lines[:min(20, len(lines))], "\n")
	if !strings.Contains(top, "TARGET") {
		t.Fatalf("top viewport content missing dashboard header: %q", top)
	}
	if strings.Contains(top, "Top Processes") {
		t.Fatalf("viewport should start at the top of the dashboard, got process panel in top slice: %q", top)
	}
}

func TestTUIModelPreservesExplicitScrollOffsetAcrossRefresh(t *testing.T) {
	t.Parallel()

	model := monitor.NewTUIModel(testTUIState())

	model.State.ScrollOffset = 12
	model.RefreshViewport()

	if got := model.Viewport.YOffset(); got != 12 {
		t.Fatalf("viewport y offset = %d, want 12", got)
	}
}

func TestTUIViewCentersDashboardWithinTallTerminal(t *testing.T) {
	t.Parallel()

	model := monitor.NewTUIModel(testTUIState())
	model.Width = 176
	model.Height = 120
	model.RefreshViewport()

	rawLines := strings.Split(ansi.StripANSI(model.Viewport.View()), "\n")
	viewLines := strings.Split(ansi.StripANSI(model.View().Content), "\n")
	top := placedLineIndex(viewLines, rawLines)
	bottom := len(viewLines) - (top + len(rawLines))
	if top <= 0 {
		t.Fatalf("expected centered view to include top padding, got top content at line %d", top)
	}
	if diff := math.Abs(float64(top - bottom)); diff > 1 {
		t.Fatalf("expected balanced vertical centering, top=%d bottom=%d", top, bottom)
	}

	left, right := horizontalPlacementMargins(viewLines[top], rawLines[0], model.Width)
	if left <= 0 || right <= 0 {
		t.Fatalf("expected centered view to include side padding, got left=%d right=%d line=%q", left, right, viewLines[top])
	}
	if diff := math.Abs(float64(left - right)); diff > 1 {
		t.Fatalf("expected balanced horizontal centering, left=%d right=%d", left, right)
	}
}

func TestTUIViewKeepsVisibleSliceHorizontallyCenteredWhileScrolled(t *testing.T) {
	t.Parallel()

	model := monitor.NewTUIModel(testTUIState())
	model.Width = 176
	model.Height = 28
	model.State.ScrollOffset = 8
	model.RefreshViewport()

	if got := model.Viewport.YOffset(); got != 8 {
		t.Fatalf("viewport y offset = %d, want 8", got)
	}

	rawLines := strings.Split(ansi.StripANSI(model.Viewport.View()), "\n")
	viewLines := strings.Split(ansi.StripANSI(model.View().Content), "\n")
	top := placedLineIndex(viewLines, rawLines)
	bottom := len(viewLines) - (top + len(rawLines))
	if top <= 0 || bottom <= 0 {
		t.Fatalf("expected scrolled slice to remain vertically centered, got top=%d bottom=%d", top, bottom)
	}
	if diff := math.Abs(float64(top - bottom)); diff > 1 {
		t.Fatalf("expected balanced vertical centering while scrolled, top=%d bottom=%d", top, bottom)
	}

	left, right := horizontalPlacementMargins(viewLines[top], rawLines[0], model.Width)
	if left <= 0 || right <= 0 {
		t.Fatalf("expected centered scrolled slice to include side padding, got left=%d right=%d line=%q", left, right, viewLines[top])
	}
	if diff := math.Abs(float64(left - right)); diff > 1 {
		t.Fatalf("expected balanced horizontal centering while scrolled, left=%d right=%d", left, right)
	}
}

func placedLineIndex(viewLines, rawLines []string) int {
	for i, raw := range rawLines {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		for j, line := range viewLines {
			if strings.Contains(line, raw) {
				return j - i
			}
		}
	}

	return -1
}

func horizontalPlacementMargins(placedLine, rawLine string, totalWidth int) (leftMargin, rightMargin int) {
	left := strings.Index(placedLine, rawLine)
	if left < 0 {
		return 0, 0
	}
	right := totalWidth - left - ansi.VisibleLen(rawLine)

	return left, right
}

func testTUIState() core.AppState {
	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) {
			cfg.Host = testHost
			cfg.Interval = time.Second
			cfg.HistoryLimit = 240
			cfg.StaleAfter = 4 * time.Second
			cfg.Theme = core.ThemeAurora
		})
		state.Current = testTUISample()
		state.HasSample = true
		state.RuntimeState = core.StatusLive
		state.RuntimeDetail = core.DetailStreamHealthy
		state.StreamAlive = true
		state.SampleCount = 343
		state.LastRx = time.Now()
		state.CPUHistory = []int{1, 1, 2, 1}
		state.CPUFreqHistory = []int{78, 79, 78, 78}
		state.CPUTempHistory = []int{45, 46, 46, 43}
		state.RAMHistory = []int{7, 7, 7, 7}
		state.RAMAvailHistory = []int{93, 93, 93, 93}
		state.DiskHistory = []int{60, 61, 62, 63}
		state.DiskLatencyHistory = []int{
			5, 7, 8, 6,
		}
		state.GPUHistory = []int{0, 0, 1, 0}
		state.VRAMHistory = []int{17, 17, 17, 17}
		state.TempHistory = []int{43, 43, 44, 43}
		state.PowerHistory = []int{13, 13, 13, 13}
		state.NetRXHistory = []int64{100 * 1024, 200 * 1024, 300 * 1024, 645 * 1024}
		state.NetTXHistory = []int64{1 * 1024 * 1024, 2 * 1024 * 1024, 3 * 1024 * 1024, 3_800 * 1024}
		state.NetIssueHistory = []int{0, 0, 10, 0}
	})

	return state
}
func testTUISample() core.Sample {
	return testSample(func(smp *core.Sample) {
		smp.RemoteTimestamp = "2026-05-29 00:10:35"
		smp.RemoteName = testRemoteName
		smp.UptimeSeconds = 9*3600 + 48*60
		smp.Load1 = 0.19
		smp.Load5 = 0.47
		smp.Load15 = 0.63
		smp.CPUCores = 12
		smp.CPUName = testCPUName
		smp.CPUPercent = 13
		smp.CPUUserPercent = 9
		smp.CPUSystemPercent = 3
		smp.CPUIOWaitPercent = 1
		smp.CPUStealPercent = 0
		smp.RAMUsedMiB = 4160
		smp.RAMTotalMiB = 15967
		smp.RAMAvailableMiB = 11807
		smp.RAMFreeMiB = 11342
		smp.RAMCacheMiB = 832
		smp.RAMBuffersMiB = 96
		smp.RAMReclaimableMiB = 165
		smp.RAMSharedMiB = 448
		smp.CPUFreqMHz = 3680
		smp.CPUMaxFreqMHz = 4700
		smp.CPUTempC = 46
		smp.CPUPressureSomeAvg10 = 0.12
		smp.CPUPressureFullAvg10 = 0.01
		smp.MemPressureSomeAvg10 = 0.06
		smp.MemPressureFullAvg10 = 0.00
		smp.SwapFreeKiB = 4 * 1024 * 1024
		smp.SwapTotalKiB = 4 * 1024 * 1024
		smp.SwapInBps = 0
		smp.SwapOutBps = 0
		smp.RootSource = testDiskSource
		smp.RootUsedKiB = 14_900 * 1024
		smp.RootTotalKiB = 1_006_900 * 1024
		smp.RootUsedPercent = 2
		smp.DiskDevice = testDiskDevice
		smp.DiskReadBps = 165 * 1024 * 1024
		smp.DiskWriteBps = 10 * 1024 * 1024
		smp.DiskReadMergedPerSec = 12
		smp.DiskWriteMergedPerSec = 7
		smp.DiskUtil = 63
		smp.DiskAwaitMS = 2.4
		smp.DiskQueueDepth = 0.4
		smp.DiskInflight = 1
		smp.TCPRetransSegsPerSec = 1
		smp.TCPResetsPerSec = 0
		smp.Net = []core.NetStat{
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceEth0
				net.RXBps = 645 * 1024
				net.TXBps = 3_800 * 1024
				net.RXPps = 44
				net.TXPps = 48
				net.SpeedMbps = 10_000
			}),
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceTailscale
				net.RXBps = 197 * 1024
				net.TXBps = 3_200 * 1024
				net.RXPps = 21
				net.TXPps = 17
			}),
		}
		smp.CPUCoresUsage = []core.CPUCore{
			{Index: 0, Percent: 4},
			{Index: 1, Percent: 0},
			{Index: 2, Percent: 2},
			{Index: 3, Percent: 1},
			{Index: 4, Percent: 1},
			{Index: 5, Percent: 1},
			{Index: 6, Percent: 6},
			{Index: 7, Percent: 0},
			{Index: 8, Percent: 2},
			{Index: 9, Percent: 0},
			{Index: 10, Percent: 1},
			{Index: 11, Percent: 1},
		}
		smp.TopProcesses = []core.ProcessStat{
			{PID: 4242, Command: testPythonCommand, CPUPercent: 99, RSSMiB: 2048},
			{PID: 2211, Command: "bash", CPUPercent: 1, RSSMiB: 4},
		}
		smp.GPUProcesses = []core.GPUProcessStat{
			{GPUUUID: testGPUUUIDZero, PID: 4242, Command: testPythonCommand, UsedMemMiB: 2048},
			{GPUUUID: testGPUUUIDZero, PID: 6009, Command: "nvidia-smi", UsedMemMiB: 23},
		}
		smp.GPUs = []core.GPUStat{testGPUStat(func(gpu *core.GPUStat) {
			gpu.Index = 0
			gpu.UUID = testGPUUUIDZero
			gpu.Name = testGPUName
			gpu.Util = 3
			gpu.MemUtil = 41
			gpu.EncoderUtil = 0
			gpu.DecoderUtil = 0
			gpu.MemUsed = 2115
			gpu.MemTotal = 12288
			gpu.Temp = 43
			gpu.PowerDraw = 21.52
			gpu.PowerLimit = 170
			gpu.Fan = 31
			gpu.SMClock = 210
			gpu.MaxSMClock = 2100
			gpu.MemClock = 810
			gpu.MaxMemClock = 7501
			gpu.GraphicsClock = 210
			gpu.VideoClock = 555
			gpu.PCIeGenCurrent = 3
			gpu.PCIeGenMax = 4
			gpu.PCIeWidthCurrent = 8
			gpu.PCIeWidthMax = 16
			gpu.PState = "P5"
		})}
	})
}
