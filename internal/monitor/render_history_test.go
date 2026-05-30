package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
	"testing"
)

func TestRenderHistoryBoxAddsGapLinesBetweenMetricRows(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Current = testSample(func(smp *core.Sample) {
			smp.CPUFreqMHz = 3680
			smp.CPUMaxFreqMHz = 4700
			smp.CPUTempC = 66
			smp.RAMTotalMiB = 15967
			smp.RAMAvailableMiB = 14928
			smp.DiskAwaitMS = 1.37
			smp.DiskQueueDepth = 0.21
			smp.Net = []core.NetStat{
				testNetStat(func(net *core.NetStat) {
					net.Iface = testIfaceEth0
					net.RXDrops = 0
					net.RXErrors = 0
					net.TXDrops = 0
					net.TXErrors = 0
				}),
				testNetStat(func(net *core.NetStat) {
					net.Iface = testIfaceTailscale
					net.RXDrops = 2
					net.RXErrors = 0
					net.TXDrops = 0
					net.TXErrors = 1
				}),
			}
			smp.GPUs = []core.GPUStat{testGPUStat(func(gpu *core.GPUStat) { gpu.Temp = 56; gpu.PowerDraw = 26.07; gpu.PowerLimit = 170 })}
		})
		state.CPUHistory = []int{1, 2, 1}
		state.CPUFreqHistory = []int{72, 76, 78}
		state.CPUTempHistory = []int{63, 65, 66}
		state.RAMHistory = []int{7, 7, 7}
		state.RAMAvailHistory = []int{93, 93, 93}
		state.DiskHistory = []int{3, 2, 3}
		state.DiskLatencyHistory = []int{2, 3, 3}
		state.GPUHistory = []int{0, 1, 0}
		state.VRAMHistory = []int{17, 17, 17}
		state.TempHistory = []int{54, 55, 56}
		state.PowerHistory = []int{14, 15, 15}
		state.NetRXHistory = []int64{1024, 2048, 4096}
		state.NetTXHistory = []int64{2048, 4096, 8192}
		state.NetIssueHistory = []int{0, 20, 40}
	})

	history := ansi.StripANSI(render.HistoryBox(state, 170))
	lines := strings.Split(strings.TrimRight(history, "\n"), "\n")
	rowIdx := -1
	for idx, line := range lines {
		if strings.Contains(line, "CPU") && strings.Contains(line, "CPU FREQ") {
			rowIdx = idx

			break
		}
	}
	if rowIdx == -1 {
		t.Fatalf("missing CPU/CPU FREQ history row in %q", history)
	}
	for _, want := range []string{"CPU TEMP", "RAM AVAIL", "DISK LAT", "NET ISSUES", "GPU TEMP"} {
		if !strings.Contains(history, want) {
			t.Fatalf("missing expanded history label %q in %q", want, history)
		}
	}
	if rowIdx+1 >= len(lines) || strings.TrimSpace(strings.Trim(lines[rowIdx+1], "│")) != "" {
		t.Fatalf("expected a blank spacer row after CPU/CPU FREQ history, got %q", lines[rowIdx+1])
	}
}
