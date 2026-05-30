package monitor_test

import (
	"bufio"
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/input"
	"github.com/lmliam/remote-monitor/internal/render"
	"os"
	"strings"
	"testing"
	"time"
)

func TestReadTTYCommandParsesScrollKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  input.TTYCommand
	}{
		{name: "down rune", input: "j", want: testTTYCommand(func(cmd *input.TTYCommand) { cmd.LineDelta = 1 })},
		{name: "up arrow", input: "\x1b[A", want: testTTYCommand(func(cmd *input.TTYCommand) { cmd.LineDelta = -1 })},
		{name: "page down", input: "\x1b[6~", want: testTTYCommand(func(cmd *input.TTYCommand) { cmd.PageDelta = 1 })},
		{name: "home", input: "\x1b[H", want: testTTYCommand(func(cmd *input.TTYCommand) { cmd.ToTop = true })},
		{name: "end", input: "\x1b[F", want: testTTYCommand(func(cmd *input.TTYCommand) { cmd.ToBottom = true })},
		{name: "quit", input: "q", want: testTTYCommand(func(cmd *input.TTYCommand) { cmd.Quit = true })},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, matched, err := input.ReadTTYCommand(bufio.NewReader(strings.NewReader(tc.input)))
			if err != nil {
				t.Fatalf("ReadTTYCommand returned error: %v", err)
			}
			if !matched {
				t.Fatal("ReadTTYCommand did not match a command")
			}
			if got != tc.want {
				t.Fatalf("command = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func testTTYTranscriptState() core.AppState {
	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) {
			cfg.Host = testHost
			cfg.Interval = time.Second
			cfg.HistoryLimit = 240
			cfg.StaleAfter = 4 * time.Second
			cfg.Theme = core.ThemeAurora
		})
		state.Current = testTTYTranscriptSample()
		state.HasSample = true
		state.RuntimeState = core.StatusLive
		state.RuntimeDetail = core.DetailStreamHealthy
		state.StreamAlive = true
		state.SampleCount = 343
		state.ReconnectCount = 0
		state.LastRx = time.Now()
		state.CPUHistory = []int{12, 24, 18, 38}
		state.CPUFreqHistory = []int{0, 0, 0, 0}
		state.CPUTempHistory = []int{63, 64, 65, 66}
		state.RAMHistory = []int{58, 59, 59, 60}
		state.RAMAvailHistory = []int{42, 41, 41, 40}
		state.DiskHistory = []int{0, 0, 0, 0}
		state.DiskLatencyHistory = []int{1, 1, 1, 1}
		state.GPUHistory = []int{5, 9, 3, 7}
		state.VRAMHistory = []int{34, 34, 34, 34}
		state.TempHistory = []int{63, 64, 64, 64}
		state.PowerHistory = []int{57, 59, 60, 61}
		state.NetRXHistory = []int64{12000, 25000, 30000, 42500}
		state.NetTXHistory = []int64{48000, 70000, 95000, 135000}
		state.NetIssueHistory = []int{0, 0, 0, 0}
	})
	state.Current.CPUCoresUsage = []core.CPUCore{
		{Index: 0, Percent: 11}, {Index: 1, Percent: 93}, {Index: 2, Percent: 99}, {Index: 3, Percent: 3},
		{Index: 4, Percent: 39}, {Index: 5, Percent: 1}, {Index: 6, Percent: 62}, {Index: 7, Percent: 2},
		{Index: 8, Percent: 15}, {Index: 9, Percent: 27}, {Index: 10, Percent: 12}, {Index: 11, Percent: 99},
	}

	return state
}

func testTTYTranscriptSample() core.Sample {
	return testSample(func(smp *core.Sample) {
		smp.RemoteName = testRemoteName
		smp.UptimeSeconds = 3600
		smp.RemoteTimestamp = "2026-05-29 03:00:00"
		smp.Load1 = 3.83
		smp.Load5 = 3.90
		smp.Load15 = 3.28
		smp.CPUCores = 12
		smp.CPUName = testCPUName
		smp.CPUPercent = 38
		smp.CPUUserPercent = 31
		smp.CPUSystemPercent = 5
		smp.CPUIOWaitPercent = 1
		smp.CPUStealPercent = 0
		smp.RAMUsedMiB = 9634
		smp.RAMTotalMiB = 15967
		smp.RAMAvailableMiB = 6333
		smp.RAMFreeMiB = 5920
		smp.RAMCacheMiB = 1650
		smp.RAMBuffersMiB = 128
		smp.RAMReclaimableMiB = 165
		smp.RAMSharedMiB = 448
		smp.CPUFreqMHz = 0
		smp.CPUMaxFreqMHz = 4700
		smp.CPUTempC = 66
		smp.CPUPressureSomeAvg10 = 0.0
		smp.CPUPressureFullAvg10 = 0.0
		smp.MemPressureSomeAvg10 = 0.0
		smp.MemPressureFullAvg10 = 0.0
		smp.SwapFreeKiB = 4194304
		smp.SwapTotalKiB = 4194304
		smp.SwapInBps = 0
		smp.SwapOutBps = 0
		smp.RootSource = testDiskSource
		smp.RootUsedKiB = 15664464
		smp.RootTotalKiB = 1055762868
		smp.RootUsedPercent = 2
		smp.DiskDevice = testDiskDevice
		smp.DiskReadBps = 0
		smp.DiskWriteBps = 136000
		smp.DiskReadMergedPerSec = 0
		smp.DiskWriteMergedPerSec = 2
		smp.DiskUtil = 0
		smp.DiskAwaitMS = 0.20
		smp.DiskQueueDepth = 0.02
		smp.DiskInflight = 0
		smp.Net = []core.NetStat{
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceEth0
				net.RXBps = 33300
				net.TXBps = 85400
				net.RXPps = 250
				net.TXPps = 510
				net.SpeedMbps = 10000
			}),
			testNetStat(func(net *core.NetStat) {
				net.Iface = testIfaceTailscale
				net.RXBps = 9200
				net.TXBps = 49700
				net.RXPps = 100
				net.TXPps = 310
				net.SpeedMbps = -1
			}),
		}
		smp.Filesystems = []core.FilesystemStat{
			{Source: testDiskSource, Mount: "/", UsedKiB: 15664464, TotalKiB: 1055762868, UsedPercent: 2, InodesUsedPercent: 17},
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
			gpu.Util = 7
			gpu.MemUtil = 34
			gpu.MemUsed = 4204
			gpu.MemTotal = 12288
			gpu.Temp = 64
			gpu.PowerDraw = 103.12
			gpu.PowerLimit = 170.0
			gpu.Fan = 45
			gpu.SMClock = 1935
			gpu.MaxSMClock = 2100
			gpu.MemClock = 7301
			gpu.MaxMemClock = 7501
			gpu.GraphicsClock = 1740
			gpu.VideoClock = 1620
			gpu.PCIeGenCurrent = 4
			gpu.PCIeGenMax = 4
			gpu.PCIeWidthCurrent = 16
			gpu.PCIeWidthMax = 16
			gpu.ThrottleReasons = "none"
			gpu.PState = "P2"
		})}
	})
}

func TestTTYTranscriptSmoke(t *testing.T) {
	if os.Getenv("MONITOR_TTY_SMOKE") != "1" {
		t.Skip("set MONITOR_TTY_SMOKE=1 to emit a real TTY transcript for manual inspection")
	}

	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")

	state := testTTYTranscriptState()

	fmt.Print(render.TTYFrame(state, 176, 40))
	state.ScrollOffset = 16
	fmt.Print(render.TTYFrame(state, 176, 40))
}
