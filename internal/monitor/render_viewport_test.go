package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
	"testing"
	"time"
)

func testScrollableDashboardState() core.AppState {
	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) {
			cfg.Host = testHost
			cfg.Interval = time.Second
			cfg.HistoryLimit = 240
			cfg.StaleAfter = 4 * time.Second
		})
		state.Current = testSample(func(smp *core.Sample) {
			smp.RemoteName = testRemoteName
			smp.UptimeSeconds = 3600
			smp.RemoteTimestamp = testRemoteTimestamp
			smp.Load1 = 0.04
			smp.Load5 = 0.04
			smp.Load15 = 1.07
			smp.CPUCores = 12
			smp.CPUName = testCPUName
			smp.CPUPercent = 1
			smp.RAMUsedMiB = 1039
			smp.RAMTotalMiB = 15967
			smp.RAMAvailableMiB = 14928
			smp.RAMCacheMiB = 3024
			smp.CPUFreqMHz = 3680
			smp.CPUMaxFreqMHz = 4700
			smp.CPUTempC = 66
			smp.SwapFreeKiB = 4194304
			smp.SwapTotalKiB = 4194304
			smp.RootSource = testDiskSource
			smp.RootUsedKiB = 15664464
			smp.RootTotalKiB = 1055762868
			smp.RootUsedPercent = 2
			smp.DiskDevice = testDiskDevice
			smp.DiskReadBps = 4096
			smp.DiskWriteBps = 8192
			smp.DiskUtil = 3
			smp.DiskAwaitMS = 1.37
			smp.DiskQueueDepth = 0.21
			smp.Net = []core.NetStat{
				testNetStat(func(net *core.NetStat) {
					net.Iface = testIfaceEth0
					net.RXBps = 4096
					net.TXBps = 8192
					net.SpeedMbps = 1000
					net.RXDrops = 0
					net.RXErrors = 0
					net.TXDrops = 0
					net.TXErrors = 0
				}),
				testNetStat(func(net *core.NetStat) {
					net.Iface = testIfaceTailscale
					net.RXBps = 260
					net.TXBps = 1536
					net.SpeedMbps = -1
					net.RXDrops = 2
					net.RXErrors = 0
					net.TXDrops = 0
					net.TXErrors = 1
				}),
			}
			smp.GPUs = []core.GPUStat{testGPUStat(func(gpu *core.GPUStat) {
				gpu.Index = 0
				gpu.Name = testGPUName
				gpu.Util = 0
				gpu.MemUtil = 17
				gpu.MemUsed = 2038
				gpu.MemTotal = 12288
				gpu.Temp = 56
				gpu.PowerDraw = 26.07
				gpu.PowerLimit = 170.00
				gpu.Fan = 0
				gpu.SMClock = 210
				gpu.MaxSMClock = 2100
				gpu.MemClock = 810
				gpu.MaxMemClock = 7501
				gpu.PState = "P5"
			})}
		})
		state.HasSample = true
		state.RuntimeState = core.StatusLive
		state.RuntimeDetail = core.DetailStreamHealthy
		state.StreamAlive = true
		state.SampleCount = 7
		state.ReconnectCount = 0
		state.LastRx = time.Now()
		state.CPUHistory = []int{0, 1, 2, 1}
		state.RAMHistory = []int{7, 7, 6, 7}
		state.DiskHistory = []int{0, 2, 4, 3}
		state.GPUHistory = []int{0, 0, 1, 0}
		state.VRAMHistory = []int{16, 16, 17, 17}
		state.TempHistory = []int{54, 55, 56, 56}
		state.PowerHistory = []int{14, 15, 15, 15}
		state.NetRXHistory = []int64{1024, 4096, 2048, 4096}
		state.NetTXHistory = []int64{2048, 8192, 4096, 8192}
	})
	state.Current.CPUCoresUsage = []core.CPUCore{{Index: 0, Percent: 2}, {Index: 1, Percent: 0}}

	return state
}

func TestRenderViewportScrollReachesHistoryOnShortTerminal(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	state := testScrollableDashboardState()

	frame, maxScroll := render.ViewportFrame(state, 176, 50, 0)
	cleaned := ansi.StripANSI(frame)

	if strings.Contains(cleaned, "History (newest on the right, rolling samples)") {
		t.Fatalf("expected top viewport of short frame to start above history, got %q", cleaned)
	}

	if got := len(strings.Split(strings.TrimRight(frame, "\n"), "\n")); got > 50 {
		t.Fatalf("frame rendered %d lines for height 50", got)
	}

	if maxScroll <= 0 {
		t.Fatalf("expected tall content to require scrolling, got maxScroll=%d", maxScroll)
	}

	scrolledFrame, _ := render.ViewportFrame(state, 176, 50, maxScroll)
	scrolledCleaned := ansi.StripANSI(scrolledFrame)
	if !strings.Contains(scrolledCleaned, "History (newest on the right, rolling samples)") {
		t.Fatalf("expected scrolled viewport to reach history, got %q", scrolledCleaned)
	}
	if !strings.Contains(scrolledCleaned, "POWER") {
		t.Fatalf("expected scrolled viewport to include lower history metrics, got %q", scrolledCleaned)
	}
}

func TestRenderTTYFrameClearsDirtySpacerLinesBeforeDrawing(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	state := testScrollableDashboardState()

	sequence := render.TTYFrame(state, 176, 92)
	if !strings.HasPrefix(sequence, cursorHome) {
		t.Fatalf("tty frame should home cursor before drawing, got prefix %q", sequence[:min(len(sequence), 8)])
	}
	if strings.HasPrefix(sequence, cursorHome+clearBelow) {
		t.Fatalf("tty frame should avoid a full-screen wipe before drawing, got prefix %q", sequence[:min(len(sequence), 8)])
	}
	if !strings.Contains(sequence, clearLine) {
		t.Fatalf("tty frame should clear individual lines before repainting")
	}

	frameLines := strings.Split(strings.TrimRight(ansi.StripANSI(render.Frame(state, 176, 92)), "\n"), "\n")
	historyIdx := -1
	for idx, line := range frameLines {
		if strings.Contains(line, "History (newest on the right, rolling samples)") {
			historyIdx = idx

			break
		}
	}
	if historyIdx < 2 {
		t.Fatalf("did not find history title with spacer in frame")
	}
	if !isBlankOrAuroraSpacer(frameLines[historyIdx-2]) {
		t.Fatalf("expected blank or aurora spacer before history box, got %q", frameLines[historyIdx-2])
	}

	frameWidth := 0
	for _, line := range frameLines {
		if width := ansi.VisibleLen(line); width > frameWidth {
			frameWidth = width
		}
	}
	dirty := make([]string, len(frameLines))
	for i := range dirty {
		dirty[i] = strings.Repeat("X", frameWidth)
	}
	painted := applyTTYSequence(dirty, sequence)
	if strings.Contains(painted[historyIdx-2], "X") {
		t.Fatalf("expected dirty spacer line to be cleared before repaint, got %q", painted[historyIdx-2])
	}
	if !isBlankOrAuroraSpacer(painted[historyIdx-2]) {
		t.Fatalf("expected spacer line to repaint blank or aurora backdrop, got %q", painted[historyIdx-2])
	}
}

func TestRenderTTYSequenceReturnsToColumnZeroBetweenLines(t *testing.T) {
	t.Parallel()

	screen := []string{"", ""}
	painted := applyTTYSequence(screen, render.TTYSequence("alpha\nbeta"))

	if got := painted[0]; got != "alpha" {
		t.Fatalf("first row = %q, want %q", got, "alpha")
	}
	if got := painted[1]; got != "beta" {
		t.Fatalf("second row = %q, want %q", got, "beta")
	}
}
