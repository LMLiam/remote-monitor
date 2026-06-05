package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
	"testing"
	"time"
)

const testBatteryStatusDischarging = "Discharging"

func testWideFrameState() core.AppState {
	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) {
			cfg.Host = testHost
			cfg.Interval = time.Second
			cfg.HistoryLimit = 240
			cfg.StaleAfter = 4 * time.Second
		})
		state.Current = testWideFrameSample()
		state.HasSample = true
		state.RuntimeState = core.StatusLive
		state.RuntimeDetail = core.DetailStreamHealthy
		state.StreamAlive = true
		state.SampleCount = 7
		state.ReconnectCount = 0
		state.LastRx = time.Now()
		state.CPUHistory = []int{0, 1, 2, 1}
		state.CPUFreqHistory = []int{72, 75, 77, 78}
		state.CPUTempHistory = []int{62, 64, 65, 66}
		state.RAMHistory = []int{7, 7, 6, 7}
		state.RAMAvailHistory = []int{93, 93, 94, 93}
		state.DiskHistory = []int{0, 2, 4, 3}
		state.DiskLatencyHistory = []int{2, 3, 4, 3}
		state.GPUHistory = []int{0, 0, 1, 0}
		state.VRAMHistory = []int{16, 16, 17, 17}
		state.TempHistory = []int{54, 55, 56, 56}
		state.PowerHistory = []int{14, 15, 15, 15}
		state.NetRXHistory = []int64{1024, 4096, 2048, 4096}
		state.NetTXHistory = []int64{2048, 8192, 4096, 8192}
		state.NetIssueHistory = []int{0, 20, 20, 40}
	})
	state.Current.CPUCoresUsage = []core.CPUCore{{Index: 0, Percent: 2}, {Index: 1, Percent: 0}}

	return state
}

func testWideFrameSample() core.Sample {
	return testSample(func(smp *core.Sample) {
		smp.RemoteName = testRemoteName
		smp.UptimeSeconds = 3600
		smp.RemoteTimestamp = testRemoteTimestamp
		smp.Load1 = 0.04
		smp.Load5 = 0.04
		smp.Load15 = 1.07
		smp.CPUCores = 12
		smp.CPUName = testCPUName
		smp.CPUPercent = 1
		smp.CPUUserPercent = 4
		smp.CPUSystemPercent = 2
		smp.CPUIOWaitPercent = 1
		smp.CPUStealPercent = 0
		smp.RAMUsedMiB = 1039
		smp.RAMTotalMiB = 15967
		smp.RAMAvailableMiB = 14928
		smp.RAMFreeMiB = 13888
		smp.RAMCacheMiB = 3024
		smp.RAMBuffersMiB = 288
		smp.RAMReclaimableMiB = 640
		smp.RAMSharedMiB = 96
		smp.CPUFreqMHz = 3680
		smp.CPUMaxFreqMHz = 4700
		smp.CPUTempC = 66
		smp.CPUPressureSomeAvg10 = 2.43
		smp.CPUPressureFullAvg10 = 0.14
		smp.MemPressureSomeAvg10 = 1.20
		smp.MemPressureFullAvg10 = 0.04
		smp.SwapFreeKiB = 4194304
		smp.SwapTotalKiB = 4194304
		smp.SwapInBps = 8192
		smp.SwapOutBps = 4096
		smp.RootSource = testDiskSource
		smp.RootUsedKiB = 15664464
		smp.RootTotalKiB = 1055762868
		smp.RootUsedPercent = 2
		smp.DiskDevice = testDiskDevice
		smp.DiskReadBps = 4096
		smp.DiskWriteBps = 8192
		smp.DiskReadMergedPerSec = 12
		smp.DiskWriteMergedPerSec = 7
		smp.DiskUtil = 3
		smp.DiskAwaitMS = 1.37
		smp.DiskQueueDepth = 0.21
		smp.DiskInflight = 3
		smp.TCPRetransSegsPerSec = 9
		smp.TCPResetsPerSec = 1
		smp.Net = []core.NetStat{
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceEth0
				net.RXBps = 4096
				net.TXBps = 8192
				net.RXPps = 1024
				net.TXPps = 512
				net.SpeedMbps = 1000
				net.RXDrops = 0
				net.RXErrors = 0
				net.RXOverruns = 0
				net.TXDrops = 0
				net.TXErrors = 0
				net.TXOverruns = 0
			}),
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceTailscale
				net.RXBps = 260
				net.TXBps = 1536
				net.RXPps = 64
				net.TXPps = 32
				net.SpeedMbps = -1
				net.RXDrops = 2
				net.RXErrors = 0
				net.RXOverruns = 1
				net.TXDrops = 0
				net.TXErrors = 1
				net.TXOverruns = 0
			}),
		}
		smp.Filesystems = []core.FilesystemStat{
			{Source: testDiskSource, Mount: "/", UsedKiB: 15664464, TotalKiB: 1055762868, UsedPercent: 2, InodesUsedPercent: 17},
			{Source: "/dev/sdc", Mount: testDataMount, UsedKiB: 2048000, TotalKiB: 10485760, UsedPercent: 20, InodesUsedPercent: 11},
			{Source: "/dev/sdb", Mount: "/mnt/archive", UsedKiB: 52428800, TotalKiB: 104857600, UsedPercent: 50, InodesUsedPercent: 44},
		}
		smp.TopProcesses = []core.ProcessStat{
			{PID: 4242, Command: testPythonCommand, CPUPercent: 188, RSSMiB: 2048},
			{PID: 8181, Command: testFFmpegCommand, CPUPercent: 42, RSSMiB: 512},
		}
		smp.GPUProcesses = []core.GPUProcessStat{
			{GPUUUID: testGPUUUID, PID: 4242, Command: testPythonCommand, UsedMemMiB: 3072},
		}
		smp.GPUs = []core.GPUStat{testGPUStat(func(gpu *core.GPUStat) {
			gpu.Index = 0
			gpu.UUID = testGPUUUID
			gpu.Name = testGPUName
			gpu.Util = 0
			gpu.MemUtil = 17
			gpu.EncoderUtil = 12
			gpu.DecoderUtil = 8
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
			gpu.GraphicsClock = 1740
			gpu.VideoClock = 1620
			gpu.PCIeGenCurrent = 3
			gpu.PCIeGenMax = 4
			gpu.PCIeWidthCurrent = 8
			gpu.PCIeWidthMax = 16
			gpu.ThrottleReasons = "power cap"
			gpu.PState = "P5"
		})}
	})
}

func TestRenderWideFrameUsesSeparateTablesAndAnimatedBanner(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	state := testWideFrameState()

	frame := render.Frame(state, 176, 92)
	cleaned := ansi.StripANSI(frame)

	assertTextContainsAll(t, "frame", cleaned, wideFrameRequiredText())
	assertTextOmitsAll(t, "frame", cleaned, wideFrameUnwantedText())
	if !strings.Contains(frame, "\x1b[38;2;") {
		t.Fatalf("frame missing animated truecolor banner escape")
	}
	assertWideFrameTrackStyling(t, frame)
	assertWideFrameAvoidsRepeatedGPUName(t, cleaned)
	assertWideFrameRowOrder(t, cleaned)
	assertTallWideFrameContent(t, state)
	assertWideFrameStoragePlacement(t, cleaned)
	if !strings.Contains(cleaned, "History (newest on the right, rolling samples)") {
		t.Fatalf("expected 176x92 frame to keep history visible when the denser packing makes it fit")
	}
}

func TestRenderWideFrameHidesUnavailableGPUVendorDetails(t *testing.T) {
	t.Parallel()

	state := testWideFrameState()
	state.Current.GPUs = []core.GPUStat{testGPUStat(func(gpu *core.GPUStat) {
		gpu.Index = 0
		gpu.UUID = "intel-0000:00:02.0"
		gpu.Name = "Intel GPU"
		gpu.Util = 42
		gpu.MemUtil = -1
		gpu.EncoderUtil = -1
		gpu.DecoderUtil = -1
		gpu.MemUsed = -1
		gpu.MemTotal = -1
		gpu.Temp = 44
		gpu.PowerDraw = -1
		gpu.PowerLimit = -1
		gpu.Fan = -1
		gpu.SMClock = 1016
		gpu.MaxSMClock = -1
		gpu.MemClock = -1
		gpu.MaxMemClock = -1
		gpu.GraphicsClock = 1016
		gpu.VideoClock = -1
		gpu.PCIeGenCurrent = -1
		gpu.PCIeGenMax = -1
		gpu.PCIeWidthCurrent = -1
		gpu.PCIeWidthMax = -1
		gpu.ThrottleReasons = ""
		gpu.PState = ""
	})}

	cleaned := ansi.StripANSI(render.FullFrame(state, 176, 48))
	assertTextContainsAll(t, "Intel GPU frame", cleaned, []string{testGPU0Load, testGPU0Temp, testGPU0SM, testGPU0Graphics})
	assertTextOmitsAll(t, "Intel GPU frame", cleaned, []string{testGPU0Encoder, testGPU0Decoder, testGPU0Fan, testGPU0PState, testGPU0Video, testGPU0PCIe, testGPU0Throttle})
}

func wideFrameRequiredText() []string {
	return []string{
		"██▀███  ▓█████  ███▄ ▄███▓",
		"CPU • AMD Ryzen 5 5600X 6-Core Processor",
		"GPU • NVIDIA GeForce RTX 3060",
		"System",
		"Memory",
		"Storage",
		"Network",
		"Top Processes",
		"GPU Processes",
		"Signal",
		"Summary",
		render.LabelProcess,
		render.LabelPID,
		"RSS",
		"VRAM",
		render.LabelCPUActive,
		render.LabelCPUImbalance,
		"avg active",
		render.LabelCPUUser,
		"CPU System",
		"CPU IOWait",
		"CPU Steal",
		render.LabelCPUPSI,
		"some 2.43%",
		"full 0.14%",
		render.LabelCPUFreq,
		"3680 MHz",
		"78%",
		render.LabelCPUTemp,
		"CPU Hot 0",
		render.LabelCPUMap,
		render.LabelRAMAvail,
		"14.6 GiB",
		render.LabelRAMFree,
		"13.6 GiB",
		render.LabelRAMCache,
		testThreeGiB,
		"RAM Buffers",
		"288 MiB",
		"RAM Reclaim",
		"640 MiB",
		"RAM Shared",
		"96 MiB",
		"Swap IO",
		"in 8.0 KiB/s",
		"out 4.0 KiB/s",
		render.LabelMemPSI,
		"some 1.20%",
		"full 0.04%",
		"Disk Source",
		"991.9 GiB free",
		"Inodes /",
		"Disk Latency",
		"Disk Merge",
		"12/s",
		"7/s",
		render.LabelDiskInflight,
		"3 active",
		"FS /mnt/archive",
		"FS /mnt/data",
		"Net eth0 RX",
		"Net eth0 TX",
		"TCP Health",
		"retx 9/s",
		"reset 1/s",
		"1%/1.0G",
		"1.0kpps",
		"512pps",
		"d2/e0/o1",
		"d0/e1/o0",
		testPythonCommand,
		testFFmpegCommand,
		"4242",
		"2.0 GiB",
		testThreeGiB,
		testGPU0Temp,
		"GPU0 Power",
		"GPU0 Mem Util",
		testGPU0Encoder,
		testGPU0Decoder,
		testGPU0Fan,
		testGPU0PState,
		"idle / cool",
		testGPU0SM,
		"GPU0 Mem",
		testGPU0Graphics,
		testGPU0Video,
		testGPU0PCIe,
		"Gen3 x8",
		"max Gen4 x16",
		testGPU0Throttle,
		"POWER CAP",
		"210 / 2100 MHz",
		"810 / 7501 MHz",
	}
}

func wideFrameUnwantedText() []string {
	return []string{
		"CPU + System",
		"Mem Avail",
		"PSI CPU",
		"PSI IO",
		"PSI MEMORY",
		"GPU0 Therm",
		"GPU NVIDIA GeForce RTX 3060",
		"14.6 GiB •",
		"3.0 GiB •",
	}
}

func assertTextContainsAll(t *testing.T, label, text string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("%s missing %q", label, want)
		}
	}
}

func assertTextOmitsAll(t *testing.T, label, text string, unwanted []string) {
	t.Helper()
	for _, item := range unwanted {
		if strings.Contains(text, item) {
			t.Fatalf("%s unexpectedly contains %q", label, item)
		}
	}
}

func assertWideFrameTrackStyling(t *testing.T, frame string) {
	t.Helper()
	lines := strings.Split(frame, "\n")
	for _, label := range []string{render.LabelCPUActive, render.LabelCPUImbalance, render.LabelCPUFreq, render.LabelCPUTemp, render.LabelCPUUser, render.LabelCPUPSI, render.LabelRAMAvail, render.LabelRAMCache, render.LabelRAMFree, render.LabelMemPSI, "Net eth0 RX", "Net eth0 TX", "GPU0 Mem Util", testGPU0Encoder, testGPU0Temp, "GPU0 Power", testGPU0SM, "GPU0 Mem", testGPU0PCIe} {
		assertRenderedTrackRow(t, lines, label)
	}
}

func assertRenderedTrackRow(t *testing.T, lines []string, label string) {
	t.Helper()
	for _, line := range lines {
		if !strings.Contains(ansi.StripANSI(line), label) {
			continue
		}
		if !strings.Contains(line, ansi.TrackBg) {
			t.Fatalf("rendered %q row missing dense track styling: %q", label, line)
		}

		return
	}
	t.Fatalf("did not find rendered row %q", label)
}

func assertWideFrameAvoidsRepeatedGPUName(t *testing.T, cleaned string) {
	t.Helper()
	for line := range strings.SplitSeq(cleaned, "\n") {
		if strings.Contains(line, testGPU0Load) && strings.Contains(line, "NVIDIA") {
			t.Fatalf("gpu load row should not repeat gpu name: %q", line)
		}
	}
}

type wideFrameRowIndexes struct {
	cpuMap       int
	cpuImbalance int
	firstCPUHot  int
	diskInflight int
	firstFS      int
	storageRule  int
}

func assertWideFrameRowOrder(t *testing.T, cleaned string) {
	t.Helper()
	indexes := findWideFrameRowIndexes(cleaned)
	if indexes.cpuMap == -1 || indexes.cpuImbalance == -1 || indexes.firstCPUHot == -1 {
		t.Fatalf("expected CPU Map, CPU Imbalance, and CPU Hot rows in rendered frame")
	}
	if indexes.cpuMap > indexes.cpuImbalance {
		t.Fatalf("expected CPU Imbalance below CPU Map, got map=%d imbalance=%d", indexes.cpuMap, indexes.cpuImbalance)
	}
	if indexes.cpuMap > indexes.firstCPUHot {
		t.Fatalf("expected CPU Hot rows below CPU Map, got map=%d firstHot=%d", indexes.cpuMap, indexes.firstCPUHot)
	}
	if indexes.diskInflight == -1 || indexes.firstFS == -1 || indexes.storageRule == -1 {
		t.Fatalf("expected Disk Inflight, filesystem rows, and a separator in rendered storage section")
	}
	if indexes.diskInflight >= indexes.storageRule || indexes.storageRule >= indexes.firstFS {
		t.Fatalf("expected storage separator between Disk Inflight and filesystem rows, got inflight=%d rule=%d firstFS=%d", indexes.diskInflight, indexes.storageRule, indexes.firstFS)
	}
}

func findWideFrameRowIndexes(cleaned string) wideFrameRowIndexes {
	indexes := wideFrameRowIndexes{
		cpuMap:       -1,
		cpuImbalance: -1,
		firstCPUHot:  -1,
		diskInflight: -1,
		firstFS:      -1,
		storageRule:  -1,
	}
	for idx, line := range strings.Split(cleaned, "\n") {
		indexes = updateWideFrameRowIndexes(indexes, idx, line)
	}

	return indexes
}

func updateWideFrameRowIndexes(indexes wideFrameRowIndexes, idx int, line string) wideFrameRowIndexes {
	switch {
	case strings.Contains(line, render.LabelCPUMap):
		indexes.cpuMap = idx
	case strings.Contains(line, render.LabelCPUImbalance):
		indexes.cpuImbalance = idx
	case strings.Contains(line, "CPU Hot ") && indexes.firstCPUHot == -1:
		indexes.firstCPUHot = idx
	case strings.Contains(line, render.LabelDiskInflight):
		indexes.diskInflight = idx
	case strings.Contains(line, testFSLabelPrefix) && indexes.firstFS == -1:
		indexes.firstFS = idx
	case indexes.diskInflight != -1 && indexes.firstFS == -1 && strings.Contains(line, "├") && strings.Contains(line, "┼"):
		indexes.storageRule = idx
	}

	return indexes
}

func assertTallWideFrameContent(t *testing.T, state core.AppState) {
	t.Helper()
	tallFrame := render.Frame(state, 176, 120)
	tallCleaned := ansi.StripANSI(tallFrame)
	assertTextContainsAll(t, "tall frame", tallCleaned, []string{"Top Processes", "GPU Processes", testPythonCommand, testFFmpegCommand, "2.0 GiB", testThreeGiB, "CPU FREQ", "CPU TEMP", "RAM AVAIL", "DISK LAT", "NET ISSUES", "GPU TEMP", "NET RX", "NET TX", "POWER"})
	assertHistorySpacer(t, tallCleaned)
}

func assertHistorySpacer(t *testing.T, tallCleaned string) {
	t.Helper()
	cleanLines := strings.Split(strings.TrimRight(tallCleaned, "\n"), "\n")
	historyTitleIdx := firstLineContaining(cleanLines, "History (newest on the right, rolling samples)")
	if historyTitleIdx < 2 {
		t.Fatalf("did not find history title in rendered frame")
	}
	if historyTitleIdx < 3 {
		t.Fatalf("history should have room for gap and system border")
	}
	if !isBlankOrAuroraSpacer(cleanLines[historyTitleIdx-2]) {
		t.Fatalf("expected blank or aurora spacer before history, got %q", cleanLines[historyTitleIdx-2])
	}
	if !strings.Contains(cleanLines[historyTitleIdx-3], "╰") || !strings.Contains(cleanLines[historyTitleIdx-3], "╯") {
		t.Fatalf("expected a table bottom border immediately before spacer, got %q", cleanLines[historyTitleIdx-3])
	}
}

func isBlankOrAuroraSpacer(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}

	return strings.Trim(trimmed, "▀") == ""
}

func assertWideFrameStoragePlacement(t *testing.T, cleaned string) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(cleaned, "\n"), "\n")
	memPSIIdx := firstLineContaining(lines, render.LabelMemPSI)
	storageIdx := firstLineContaining(lines, "│ Storage")
	if memPSIIdx == -1 || storageIdx == -1 {
		t.Fatalf("expected both Memory and Storage sections in wide frame")
	}
	if storageIdx > memPSIIdx {
		t.Fatalf("expected Storage to rise into freed left-column space before Memory finished, got storage line %d after Mem PSI line %d", storageIdx, memPSIIdx)
	}
}

func firstLineContaining(lines []string, needle string) int {
	for idx, line := range lines {
		if strings.Contains(line, needle) {
			return idx
		}
	}

	return -1
}

func TestRenderMediumTwoColumnLayoutStacksMemoryUnderShortGPU(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	state.Current.GPUs = nil
	state.Current.GPUProcesses = nil

	frame := ansi.StripANSI(render.Frame(state, 176, 92))
	memoryIdx := -1
	cpuMapIdx := -1
	for idx, line := range strings.Split(strings.TrimRight(frame, "\n"), "\n") {
		switch {
		case strings.Contains(line, "│ Memory"):
			memoryIdx = idx
		case strings.Contains(line, render.LabelCPUMap):
			cpuMapIdx = idx
		}
	}

	if memoryIdx == -1 || cpuMapIdx == -1 {
		t.Fatalf("expected both Memory and CPU Map sections in medium frame")
	}
	if memoryIdx > cpuMapIdx {
		t.Fatalf("expected Memory to stack under the shorter GPU column before the CPU column finished, got memory line %d after CPU Map line %d", memoryIdx, cpuMapIdx)
	}
}

func TestRenderFrameUnavailableGPUMessageMentionsSupportedCollectors(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	state.Current.GPUs = nil
	state.Current.GPUProcesses = nil

	frame := ansi.StripANSI(render.Frame(state, 176, 92))
	if !strings.Contains(frame, "supported GPU") {
		t.Fatalf("expected unavailable GPU text to mention supported GPU collectors, got %q", frame)
	}
	if strings.Contains(frame, "nvidia-smi unavailable") {
		t.Fatalf("unavailable GPU text should not imply NVIDIA-only support, got %q", frame)
	}
}

func TestRenderFrameAddsPowerSectionOnlyWhenPowerDataExists(t *testing.T) {
	t.Parallel()

	state := testWideFrameState()
	withPowerSample(&state.Current)

	cleaned := ansi.StripANSI(render.Frame(state, 176, 120))
	assertTextContainsAll(t, "power frame", cleaned, []string{"│ Power", "External", "online", "Battery", "BAT0 83%", testBatteryStatusDischarging, "UPS", "present"})

	state.Current = testWideFrameSample()
	cleaned = ansi.StripANSI(render.Frame(state, 176, 120))
	if strings.Contains(cleaned, "│ Power") {
		t.Fatalf("frame should omit power section when no power data exists, got %q", cleaned)
	}
}

func TestRenderNonInteractivePowerSummaryMatchesTextOutputBehavior(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Current = testSample(withPowerSample)
		state.HasSample = true
		state.RuntimeState = core.StatusLive
	})

	got := render.NonInteractive(state)
	if !strings.Contains(got, "Power AC online") || !strings.Contains(got, "BAT0 83% Discharging") || !strings.Contains(got, "12.34W") || !strings.Contains(got, "UPS present") {
		t.Fatalf("text output missing power summary: %q", got)
	}

	state.Current = testSample()
	got = render.NonInteractive(state)
	if strings.Contains(got, "Power") {
		t.Fatalf("text output should omit power summary when no power data exists: %q", got)
	}
}

func withPowerSample(smp *core.Sample) {
	smp.ExternalPowerOnline = 1
	smp.BatteryPercent = 83
	smp.BatteryStatus = testBatteryStatusDischarging
	smp.PowerDrawWatts = 12.34
	smp.UPSPresent = 1
	smp.PowerSourceName = "BAT0"
	smp.PowerSupplies = []core.PowerSupplyStat{
		{Name: "AC0", Type: "Mains", Online: 1, CapacityPercent: -1, Status: "", PowerDrawWatts: -1, Present: -1},
		{Name: "BAT0", Type: "Battery", Online: -1, CapacityPercent: 83, Status: testBatteryStatusDischarging, PowerDrawWatts: 12.34, Present: 1},
		{Name: "UPS0", Type: "UPS", Online: 0, CapacityPercent: 55, Status: "Full", PowerDrawWatts: -1, Present: 1},
	}
}

func TestRenderFrameLineWidthsConsistent(t *testing.T) {
	t.Parallel()

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

	for _, tc := range []struct {
		width int
		want  int
	}{
		{width: 176, want: 174},
		{width: 240, want: 238},
	} {
		frame := render.Frame(state, tc.width, 92)
		lines := strings.SplitSeq(strings.TrimRight(frame, "\n"), "\n")
		for line := range lines {
			if ansi.VisibleLen(line) == 0 {
				continue
			}
			if got := ansi.VisibleLen(line); got != tc.want {
				t.Fatalf("width %d: line width = %d, want %d for %q", tc.width, got, tc.want, line)
			}
		}
	}
}

func TestRenderWideFrameCompactModeStacksTablesAndSkipsBanner(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) {
			cfg.Host = testHost
			cfg.Interval = time.Second
			cfg.HistoryLimit = 240
			cfg.StaleAfter = 4 * time.Second
			cfg.Compact = true
			cfg.Theme = core.ThemeAurora
		})
		state.Current = testSample(func(smp *core.Sample) {
			smp.RemoteName = testRemoteName
			smp.UptimeSeconds = 3600
			smp.RemoteTimestamp = testRemoteTimestamp
			smp.Load1 = 0.04
			smp.Load5 = 0.04
			smp.Load15 = 1.07
			smp.CPUCores = 12
			smp.CPUPercent = 1
			smp.RAMUsedMiB = 1039
			smp.RAMTotalMiB = 15967
		})
		state.HasSample = true
		state.RuntimeState = core.StatusLive
		state.StreamAlive = true
		state.LastRx = time.Now()
	})

	cleaned := ansi.StripANSI(render.FullFrame(state, 176, 40))
	if strings.Contains(cleaned, "██████╗") {
		t.Fatal("expected compact mode to skip the large banner")
	}
	if !strings.Contains(cleaned, "REMOTE MONITOR") {
		t.Fatal("expected compact mode to keep the compact title")
	}
	if strings.Count(cleaned, "│ Metric") < 4 {
		t.Fatalf("expected compact mode to stack the grouped tables, got frame %q", cleaned)
	}
}
