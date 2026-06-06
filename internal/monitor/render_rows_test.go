package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
	"testing"
)

func TestNetSpeedCeilingHelpers(t *testing.T) {
	t.Parallel()

	net := testNetStat(func(net *core.NetStat) { net.SpeedMbps = 1000 })
	if got := metrics.NetLinkBps(net); got != 125000000 {
		t.Fatalf("NetLinkBps = %d", got)
	}
	if got := render.FormatLinkSpeed(1000); got != "1.0 Gbps" {
		t.Fatalf("FormatLinkSpeed = %q", got)
	}
	if got := metrics.RatePercent(4096, metrics.NetLinkBps(net)); got != 1 {
		t.Fatalf("RatePercent = %d", got)
	}
}

func TestCoreHeatmapCellUsesSquareMultiRowGrid(t *testing.T) {
	t.Parallel()

	got := ansi.StripANSI(render.CoreHeatmapCell([]core.CPUCore{
		{Index: 7, Percent: 99},
		{Index: 0, Percent: 0},
		{Index: 1, Percent: 4},
		{Index: 2, Percent: 18},
		{Index: 3, Percent: 44},
		{Index: 4, Percent: 61},
		{Index: 5, Percent: 81},
		{Index: 6, Percent: 2},
	}, 24, core.DefaultThresholds()))

	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("cpu map should use multiple rows when width allows, got %q", got)
	}
	for _, line := range lines {
		if ansi.VisibleLen(line) != 24 {
			t.Fatalf("cpu map line width = %d, want 24 in %q", ansi.VisibleLen(line), line)
		}
	}
	for _, want := range []string{"▁", "▂", "▄", "▅", "▇", "█"} {
		if !strings.Contains(got, want) {
			t.Fatalf("cpu map strip missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "╎") || strings.Contains(got, "▀") {
		t.Fatalf("cpu map should render as a square-ish grid, got legacy strip markers in %q", got)
	}
}

func TestCoreHeatmapCellFallsBackToSingleLineOnNarrowWidths(t *testing.T) {
	t.Parallel()

	cores := make([]core.CPUCore, 0, 32)
	for i := range 32 {
		cores = append(cores, core.CPUCore{Index: i, Percent: (i * 7) % 100})
	}

	got := ansi.StripANSI(render.CoreHeatmapCell(cores, 12, core.DefaultThresholds()))
	if strings.Contains(got, "\n") {
		t.Fatalf("narrow cpu map should stay single-line, got %q", got)
	}
	if ansi.VisibleLen(got) != 12 {
		t.Fatalf("narrow cpu map width = %d, want 12 in %q", ansi.VisibleLen(got), got)
	}
}

func TestRenderTableBoxSupportsMultilineActivityCells(t *testing.T) {
	t.Parallel()

	rows := []render.TableRowSpec{render.TableFullRow(render.LabelCPUMap, ansi.Sand, "12 cores • peak 99%", ansi.Sand, "", "", "", strings.Repeat("A", 10)+"\n"+strings.Repeat("B", 10))}

	lines := render.TableBox("CPU", rows, 12, 18, 10)
	rendered := ansi.StripANSI(strings.Join(lines, "\n"))
	if strings.Count(rendered, render.LabelCPUMap) != 1 {
		t.Fatalf("expected single CPU Map label on multiline row, got %q", rendered)
	}
	if !strings.Contains(rendered, strings.Repeat("A", 10)) || !strings.Contains(rendered, strings.Repeat("B", 10)) {
		t.Fatalf("expected multiline activity cell content in rendered table, got %q", rendered)
	}
}

func TestComputeTableWidthsForRowsBalancesGaugeHeavyTables(t *testing.T) {
	t.Parallel()

	state := testTUIState()

	_, cpuValueWidth, cpuActivityWidth := render.ComputeTableWidthsForRows(86, func(valueWidth, activityWidth int) []render.TableRowSpec {
		return render.BuildCPURows(state, valueWidth, activityWidth, false)
	})
	if cpuActivityWidth < 16 {
		t.Fatalf("cpu activity width should preserve multi-row cpu map, got %d", cpuActivityWidth)
	}
	if cpuValueWidth >= cpuActivityWidth {
		t.Fatalf("cpu activity width should dominate the value column, got value=%d activity=%d", cpuValueWidth, cpuActivityWidth)
	}
	if cpuValueWidth > 28 {
		t.Fatalf("cpu value width should stay on a short leash for Gauge-heavy rows, got %d", cpuValueWidth)
	}
	if cpuValueWidth-cpuActivityWidth > 6 {
		t.Fatalf("cpu value width should not dominate activity width, got value=%d activity=%d", cpuValueWidth, cpuActivityWidth)
	}

	_, netValueWidth, netActivityWidth := render.ComputeTableWidthsForRows(86, func(valueWidth, activityWidth int) []render.TableRowSpec {
		_ = valueWidth

		return render.BuildNetworkRows(state, activityWidth, false)
	})
	if netValueWidth >= netActivityWidth {
		t.Fatalf("network activity width should dominate the value column, got value=%d activity=%d", netValueWidth, netActivityWidth)
	}
	if netValueWidth-netActivityWidth > 6 {
		t.Fatalf("network value width should not dominate activity width, got value=%d activity=%d", netValueWidth, netActivityWidth)
	}

	_, storageValueWidth, storageActivityWidth := render.ComputeTableWidthsForRows(86, func(_ int, activityWidth int) []render.TableRowSpec {
		return render.BuildStorageRows(state, activityWidth, false)
	})
	if storageValueWidth >= storageActivityWidth {
		t.Fatalf("storage activity width should dominate the value column, got value=%d activity=%d", storageValueWidth, storageActivityWidth)
	}
	if storageValueWidth > 30 {
		t.Fatalf("storage value width should stay compact enough to feed graphs, got %d", storageValueWidth)
	}

	_, cpuWideValueWidth, cpuWideActivityWidth := render.ComputeTableWidthsForRows(108, func(valueWidth, activityWidth int) []render.TableRowSpec {
		return render.BuildCPURows(state, valueWidth, activityWidth, false)
	})
	if cpuWideValueWidth > 30 {
		t.Fatalf("wide one-column cpu value width should stay compact enough to feed graphs, got %d", cpuWideValueWidth)
	}
	if cpuWideValueWidth >= cpuWideActivityWidth {
		t.Fatalf("wide one-column cpu activity width should dominate the value column, got value=%d activity=%d", cpuWideValueWidth, cpuWideActivityWidth)
	}
}

func TestMediumCPUSectionKeepsMultilineCPUMap(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	_, valueWidth, activityWidth := render.ComputeTableWidthsForRows(86, func(valueWidth, activityWidth int) []render.TableRowSpec {
		return render.BuildCPURows(state, valueWidth, activityWidth, false)
	})
	rows := render.BuildCPURows(state, valueWidth, activityWidth, false)
	if len(rows) == 0 {
		t.Fatalf("expected cpu rows")
	}
	activity := ""
	for _, row := range rows {
		if row.LabelText == render.LabelCPUMap {
			activity = row.ActivityCell

			break
		}
	}
	if activity == "" {
		t.Fatalf("expected CPU Map row")
	}
	if !strings.Contains(activity, "\n") {
		t.Fatalf("expected medium-width cpu map to stay multiline, got %q", ansi.StripANSI(activity))
	}
}

func TestBuildCPURowsPlacesImbalanceAndHotRowsBelowCPUMap(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	rows := render.BuildCPURows(state, 24, 40, false)

	indexOf := func(label string) int {
		for i, row := range rows {
			if row.LabelText == label {
				return i
			}
		}

		return -1
	}

	mapIdx := indexOf(render.LabelCPUMap)
	imbalanceIdx := indexOf(render.LabelCPUImbalance)
	hotIdx := indexOf("CPU Hot 0")
	if mapIdx == -1 || imbalanceIdx == -1 || hotIdx == -1 {
		t.Fatalf("expected CPU Map, CPU Imbalance, and CPU Hot rows in cpu section")
	}
	if mapIdx > imbalanceIdx {
		t.Fatalf("expected CPU Imbalance below CPU Map, got map=%d imbalance=%d", mapIdx, imbalanceIdx)
	}
	if mapIdx > hotIdx {
		t.Fatalf("expected CPU Hot rows below CPU Map, got map=%d hot=%d", mapIdx, hotIdx)
	}
	if imbalanceIdx-mapIdx < 2 || !rows[mapIdx+1].Divider {
		t.Fatalf("expected divider between CPU Map and post-map CPU rows")
	}
}

func TestFilesystemLabelCompactsLongMountsWithoutEllipsis(t *testing.T) {
	t.Parallel()

	for mount, wantParts := range map[string][]string{
		"/mnt/c":                  {"FS /mnt/c"},
		"/mnt/wslg/versions.txt":  {testFSLabelPrefix, "wslg/", ".txt"},
		"/run/credentials/system": {testFSLabelPrefix, "system"},
		"/usr/lib/modules":        {testFSLabelPrefix, "lib/", "modules"},
		"/usr/lib/wsl/drivers":    {testFSLabelPrefix, "wsl/", "drivers"},
		"/":                       {"FS /"},
	} {
		got := render.FilesystemLabel(mount)
		for _, wantPart := range wantParts {
			if !strings.Contains(got, wantPart) {
				t.Fatalf("FilesystemLabel(%q) = %q, want to contain %q", mount, got, wantPart)
			}
		}
		if strings.Contains(got, "…") {
			t.Fatalf("FilesystemLabel(%q) should avoid ellipsis, got %q", mount, got)
		}
		if ansi.VisibleLen(got) > 18 {
			t.Fatalf("FilesystemLabel(%q) should fit metric column, got %q (%d)", mount, got, ansi.VisibleLen(got))
		}
	}
}

func TestRenderMediumFrameAvoidsEllipsisInTableContent(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	frame := ansi.StripANSI(render.Frame(state, 176, 92))
	if strings.Contains(frame, "…") {
		t.Fatalf("medium frame should avoid ellipsis in table content, got %q", frame)
	}
}

func TestBuildStorageRowsSeparatesFilesystemRowsFromDiskInflight(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	state.Current.Filesystems = []core.FilesystemStat{
		{Source: testDiskSource, Mount: "/", UsedKiB: state.Current.RootUsedKiB, TotalKiB: state.Current.RootTotalKiB, UsedPercent: state.Current.RootUsedPercent, InodesUsedPercent: 17},
		{Source: "/dev/sdc", Mount: testDataMount, UsedKiB: 2048000, TotalKiB: 10485760, UsedPercent: 20, InodesUsedPercent: 11},
		{Source: "/dev/sdb", Mount: "/mnt/archive", UsedKiB: 52428800, TotalKiB: 104857600, UsedPercent: 50, InodesUsedPercent: 44},
	}
	rows := render.BuildStorageRows(state, 40, false)

	inflightIdx := -1
	fsIdx := -1
	for idx, row := range rows {
		switch {
		case row.LabelText == render.LabelDiskInflight:
			inflightIdx = idx
		case strings.HasPrefix(row.LabelText, testFSLabelPrefix):
			fsIdx = idx
		}
	}
	if inflightIdx == -1 || fsIdx == -1 {
		t.Fatalf("expected Disk Inflight and filesystem rows in storage rows")
	}
	if fsIdx-inflightIdx < 2 {
		t.Fatalf("expected divider between Disk Inflight and filesystem rows, got inflight=%d fs=%d", inflightIdx, fsIdx)
	}
	if !rows[inflightIdx+1].Divider {
		t.Fatalf("expected divider row between Disk Inflight and filesystem rows")
	}
}

func TestBuildStorageRowsRendersMultipleDiskDevices(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	state.Current.Disks = []core.DiskStat{
		testDiskStat(func(disk *core.DiskStat) {
			disk.Device = testDiskDevice
			disk.ReadBps = 4096
			disk.WriteBps = 8192
			disk.Util = 3
			disk.AwaitMS = 1.37
			disk.QueueDepth = 0.21
			disk.Inflight = 3
		}),
		testDiskStat(func(disk *core.DiskStat) {
			disk.Device = testNVMeDiskDevice
			disk.ReadBps = 1048576
			disk.WriteBps = 524288
			disk.ReadMergedPerSec = 12
			disk.WriteMergedPerSec = 7
			disk.Util = 63
			disk.AwaitMS = 2.4
			disk.QueueDepth = 0.4
			disk.Inflight = 1
		}),
	}

	rows := render.BuildStorageRows(state, 40, false)
	assertRowLabelExists(t, rows, "Disk IO "+testDiskDevice)
	assertRowLabelExists(t, rows, "Disk IO "+testNVMeDiskDevice)
	assertRowLabelExists(t, rows, "Disk Lat "+testDiskDevice)
	assertRowLabelExists(t, rows, "Disk Lat "+testNVMeDiskDevice)
	assertRowLabelExists(t, rows, "Disk Merge "+testNVMeDiskDevice)
	assertRowLabelExists(t, rows, "Inflight "+testNVMeDiskDevice)
}

func TestBuildStorageRowsKeepsSingleDiskRowsUnchanged(t *testing.T) {
	t.Parallel()

	state := testTUIState()
	state.Current.Disks = []core.DiskStat{
		testDiskStat(func(disk *core.DiskStat) {
			disk.Device = testDiskDevice
			disk.ReadBps = state.Current.DiskReadBps
			disk.WriteBps = state.Current.DiskWriteBps
			disk.ReadMergedPerSec = state.Current.DiskReadMergedPerSec
			disk.WriteMergedPerSec = state.Current.DiskWriteMergedPerSec
			disk.Util = state.Current.DiskUtil
			disk.AwaitMS = state.Current.DiskAwaitMS
			disk.QueueDepth = state.Current.DiskQueueDepth
			disk.Inflight = state.Current.DiskInflight
		}),
	}

	rows := render.BuildStorageRows(state, 40, false)
	assertRowLabelExists(t, rows, "Disk IO "+testDiskDevice)
	assertRowLabelExists(t, rows, "Disk Latency")
	assertRowLabelExists(t, rows, render.LabelDiskInflight)
	assertRowLabelMissing(t, rows, "Disk Lat "+testDiskDevice)
	assertRowLabelMissing(t, rows, "Inflight "+testDiskDevice)
}

func TestCPUHelpersSurfaceActiveAndImbalanceMetrics(t *testing.T) {
	t.Parallel()

	cores := []core.CPUCore{
		{Index: 0, Percent: 0},
		{Index: 1, Percent: 7},
		{Index: 2, Percent: 33},
		{Index: 3, Percent: 90},
	}
	if got := metrics.CPUActiveCoreCount(cores, 5); got != 3 {
		t.Fatalf("CPUActiveCoreCount = %d", got)
	}
	if got := metrics.CPUAveragePercent(cores); got != 43 {
		t.Fatalf("CPUAveragePercent = %d", got)
	}
	if got := metrics.CPUPeakCore(cores); got.Index != 3 || got.Percent != 90 {
		t.Fatalf("CPUPeakCore = %#v", got)
	}
	if got := metrics.CPUImbalancePercent(cores); got != 47 {
		t.Fatalf("CPUImbalancePercent = %d", got)
	}
}

func TestDiskAndNetSummariesUseCollectedData(t *testing.T) {
	t.Parallel()

	s := testSample(func(smp *core.Sample) {
		smp.RootSource = ""
		smp.DiskDevice = testDiskDevice
		smp.RootUsedKiB = 10
		smp.RootTotalKiB = 100
	})
	if got := metrics.DiskSourceText(s); got != testDiskSource {
		t.Fatalf("DiskSourceText = %q", got)
	}
	if got := render.FormatKiBValue(metrics.DiskFreeKiB(s)); got != "90 KiB" {
		t.Fatalf("DiskFreeKiB/FormatKiBValue = %q", got)
	}

	known := render.NetUtilSummary(1250000, 0, testNetStat(func(net *core.NetStat) { net.Iface = testIfaceEth0; net.SpeedMbps = 1000 }))
	if !strings.Contains(known, "1% / 1.0G") {
		t.Fatalf("known NetUtilSummary = %q", known)
	}

	unknown := render.NetUtilSummary(512*1024, 1024*1024, testNetStat(func(net *core.NetStat) { net.Iface = testIfaceTailscale; net.SpeedMbps = -1 }))
	if !strings.Contains(unknown, "50% auto") {
		t.Fatalf("unknown NetUtilSummary = %q", unknown)
	}

	if got := render.NetDirectionHealthSummary(2, 1); got != "d2/e1" {
		t.Fatalf("NetDirectionHealthSummary = %q", got)
	}
	if got := render.FormatMillisValue(1.37); got != "1.37 ms" {
		t.Fatalf("FormatMillisValue = %q", got)
	}
	if got := render.FormatQueueDepth(0.21); got != "0.21x" {
		t.Fatalf("FormatQueueDepth = %q", got)
	}
}

func assertRowLabelExists(t *testing.T, rows []render.TableRowSpec, label string) {
	t.Helper()

	if !rowLabelExists(rows, label) {
		t.Fatalf("expected row label %q in %#v", label, rows)
	}
}

func assertRowLabelMissing(t *testing.T, rows []render.TableRowSpec, label string) {
	t.Helper()

	if rowLabelExists(rows, label) {
		t.Fatalf("unexpected row label %q in %#v", label, rows)
	}
}

func rowLabelExists(rows []render.TableRowSpec, label string) bool {
	for _, row := range rows {
		if row.LabelText == label {
			return true
		}
	}

	return false
}
