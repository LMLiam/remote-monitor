package metrics_test

import (
	"reflect"
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
)

func TestDiskSourceTextPrefersRootSourceThenDevice(t *testing.T) {
	t.Parallel()

	withRootSource := core.EmptySample()
	withRootSource.RootSource = " /dev/mapper/root "
	withRootSource.DiskDevice = "nvme0n1"
	if got := metrics.DiskSourceText(withRootSource); got != "/dev/mapper/root" {
		t.Fatalf("DiskSourceText with root source = %q, want /dev/mapper/root", got)
	}

	withDevice := core.EmptySample()
	withDevice.DiskDevice = "sda"
	if got := metrics.DiskSourceText(withDevice); got != "/dev/sda" {
		t.Fatalf("DiskSourceText with device = %q, want /dev/sda", got)
	}

	if got := metrics.DiskSourceText(core.EmptySample()); got != "n/a" {
		t.Fatalf("DiskSourceText empty = %q, want n/a", got)
	}
}

func TestDiskFreeKiBAndPercentHandleSentinels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		totalKiB    int64
		usedKiB     int64
		wantFreeKiB int64
		wantPercent int
	}{
		{name: "missing total", totalKiB: 0, usedKiB: 10, wantFreeKiB: -1, wantPercent: 0},
		{name: "unknown used", totalKiB: 100, usedKiB: -1, wantFreeKiB: -1, wantPercent: 0},
		{name: "overused clamps free", totalKiB: 100, usedKiB: 125, wantFreeKiB: 0, wantPercent: 0},
		{name: "normal", totalKiB: 100, usedKiB: 40, wantFreeKiB: 60, wantPercent: 60},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			smp := core.EmptySample()
			smp.RootTotalKiB = tc.totalKiB
			smp.RootUsedKiB = tc.usedKiB
			if got := metrics.DiskFreeKiB(smp); got != tc.wantFreeKiB {
				t.Fatalf("DiskFreeKiB = %d, want %d", got, tc.wantFreeKiB)
			}
			if got := metrics.DiskFreePercent(smp); got != tc.wantPercent {
				t.Fatalf("DiskFreePercent = %d, want %d", got, tc.wantPercent)
			}
		})
	}
}

func TestRootAndExtraFilesystems(t *testing.T) {
	t.Parallel()

	root := testFilesystemStat("/dev/root", " / ", 40, 100, 40, 12)
	data := testFilesystemStat("/dev/sdb1", "/mnt/data", 200, 1000, 20, 5)
	tmp := testFilesystemStat("tmpfs", "/run", 10, 100, 10, 1)
	smp := core.EmptySample()
	smp.Filesystems = []core.FilesystemStat{data, root, tmp}

	gotRoot, ok := metrics.RootFilesystem(smp)
	if !ok {
		t.Fatal("RootFilesystem ok = false, want true")
	}
	if !reflect.DeepEqual(gotRoot, root) {
		t.Fatalf("RootFilesystem = %#v, want %#v", gotRoot, root)
	}

	gotExtra := metrics.ExtraFilesystems(smp)
	wantExtra := []core.FilesystemStat{data, tmp}
	if !reflect.DeepEqual(gotExtra, wantExtra) {
		t.Fatalf("ExtraFilesystems = %#v, want %#v", gotExtra, wantExtra)
	}

	missingRoot := core.EmptySample()
	missingRoot.Filesystems = []core.FilesystemStat{data}
	gotRoot, ok = metrics.RootFilesystem(missingRoot)
	if ok {
		t.Fatalf("RootFilesystem missing ok = true with %#v, want false", gotRoot)
	}
	if !reflect.DeepEqual(gotRoot, core.EmptyFilesystemStat()) {
		t.Fatalf("RootFilesystem missing stat = %#v, want empty", gotRoot)
	}
}

func testFilesystemStat(source, mount string, usedKiB, totalKiB int64, usedPercent, inodesUsedPercent int) core.FilesystemStat {
	return core.FilesystemStat{
		Source:            source,
		Mount:             mount,
		UsedKiB:           usedKiB,
		TotalKiB:          totalKiB,
		UsedPercent:       usedPercent,
		InodesUsedPercent: inodesUsedPercent,
	}
}
