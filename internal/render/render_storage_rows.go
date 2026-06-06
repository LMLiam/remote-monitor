package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"path"
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"
)

// BuildStorageRows builds storage, filesystem, and disk IO rows for the dashboard.
func BuildStorageRows(state core.AppState, activityWidth int, condensed bool) []TableRowSpec {
	s := state.Current
	thresholds := thresholdsOrDefaults(state.Cfg.Thresholds)
	rows := make([]TableRowSpec, 0, storageBaseRowsCap+len(s.Filesystems)+(len(s.Disks)*storageRowsPerDisk))

	if s.RootTotalKiB > 0 {
		freePct := metrics.DiskFreePercent(s)
		rows = append(rows, TableFullRow("Disk Source", ansi.Cyan, fmt.Sprintf("%s %s free", metrics.DiskSourceText(s), FormatKiBValue(metrics.DiskFreeKiB(s))), ansi.Cyan, "", percentDisplay(freePct)+" free", ansi.Green, ""))

		diskSeverity := diskUsageSeverity(s.RootUsedPercent, thresholds)
		rows = append(rows, TableFullRow("Disk /", SeverityColor(diskSeverity), formatKiBPair(s.RootUsedKiB, s.RootTotalKiB), SeverityColor(diskSeverity), "", "", "", gaugeCell(s.RootUsedPercent, activityWidth, diskSeverity)))

		if rootFS, ok := metrics.RootFilesystem(s); ok && rootFS.InodesUsedPercent >= 0 {
			inodeSeverity := inodeUsageSeverity(rootFS.InodesUsedPercent, thresholds)
			rows = append(rows, TableFullRow("Inodes /", SeverityColor(inodeSeverity), percentDisplay(rootFS.InodesUsedPercent)+" used", SeverityColor(inodeSeverity), "", "", "", gaugeBarCell(rootFS.InodesUsedPercent, activityWidth, SeverityColor(inodeSeverity), percentDisplay(rootFS.InodesUsedPercent))))
		}
	}

	if len(s.Disks) > 1 {
		for _, disk := range sortedDiskStats(s.Disks) {
			rows = appendDiskRows(rows, disk, activityWidth, condensed, true, thresholds)
		}
	} else {
		rows = appendLegacyDiskRows(rows, s, activityWidth, condensed, thresholds)
	}

	if !condensed {
		extraFS := metrics.ExtraFilesystems(s)
		sort.Slice(extraFS, func(i, j int) bool { return extraFS[i].Mount < extraFS[j].Mount })
		if len(extraFS) > 0 && len(rows) > 0 {
			rows = append(rows, tableDividerRow())
		}
		for _, fs := range extraFS {
			if fs.TotalKiB <= 0 {
				continue
			}
			fsSeverity := diskUsageSeverity(fs.UsedPercent, thresholds)
			fsSuffix := percentDisplay(fs.UsedPercent)
			if fs.InodesUsedPercent >= 0 {
				fsSuffix = "inode " + percentDisplay(fs.InodesUsedPercent)
			}
			rows = append(rows, TableFullRow(FilesystemLabel(fs.Mount), SeverityColor(fsSeverity), formatKiBPair(fs.UsedKiB, fs.TotalKiB), SeverityColor(fsSeverity), "", "", "", gaugeBarCell(fs.UsedPercent, activityWidth, SeverityColor(fsSeverity), fsSuffix)))
		}
	}

	return rows
}

func appendLegacyDiskRows(rows []TableRowSpec, s core.Sample, activityWidth int, condensed bool, thresholds core.Thresholds) []TableRowSpec {
	disk := core.DiskStat{
		Device:            s.DiskDevice,
		ReadBps:           s.DiskReadBps,
		WriteBps:          s.DiskWriteBps,
		ReadMergedPerSec:  s.DiskReadMergedPerSec,
		WriteMergedPerSec: s.DiskWriteMergedPerSec,
		Util:              s.DiskUtil,
		AwaitMS:           s.DiskAwaitMS,
		QueueDepth:        s.DiskQueueDepth,
		Inflight:          s.DiskInflight,
	}

	return appendDiskRows(rows, disk, activityWidth, condensed, false, thresholds)
}

func appendDiskRows(rows []TableRowSpec, disk core.DiskStat, activityWidth int, condensed, includeDeviceInDetails bool, thresholds core.Thresholds) []TableRowSpec {
	device := fallbackString(disk.Device, TextNA)
	if disk.ReadBps >= 0 || disk.WriteBps >= 0 {
		diskIOSeverity := diskUtilSeverity(disk.Util, thresholds)
		rows = append(rows, TableFullRow("Disk IO "+device, SeverityColor(diskIOSeverity), fmt.Sprintf("R %s • W %s", formatBps(disk.ReadBps), formatBps(disk.WriteBps)), SeverityColor(diskIOSeverity), "", "", "", gaugeCell(disk.Util, activityWidth, diskIOSeverity)))

		if disk.AwaitMS >= 0 || disk.QueueDepth >= 0 {
			label := "Disk Latency"
			if includeDeviceInDetails {
				label = "Disk Lat " + device
			}
			latencySeverity := mergeSeverity(diskAwaitSeverity(disk.AwaitMS), diskQueueSeverity(disk.QueueDepth))
			rows = append(rows, TableFullRow(label, SeverityColor(latencySeverity), "await "+FormatMillisValue(disk.AwaitMS), SeverityColor(latencySeverity), "", "queue "+FormatQueueDepth(disk.QueueDepth), SeverityColor(latencySeverity), ""))
		}
	}

	if !condensed && (disk.ReadMergedPerSec >= 0 || disk.WriteMergedPerSec >= 0) {
		label := "Disk Merge"
		if includeDeviceInDetails {
			label += " " + device
		}
		rows = append(rows, TableFullRow(label, ansi.Cyan, fmt.Sprintf("R %s • W %s", formatOpsPerSec(disk.ReadMergedPerSec), formatOpsPerSec(disk.WriteMergedPerSec)), ansi.Cyan, "", "merged ops", ansi.Cyan, ""))
	}

	if !condensed && disk.Inflight >= 0 {
		label := LabelDiskInflight
		if includeDeviceInDetails {
			label = "Inflight " + device
		}
		inflightSeverity := diskQueueSeverity(float64(disk.Inflight))
		inflightPct := clamp(disk.Inflight*diskInflightScale, percentMin, percentMax)
		rows = append(rows, TableFullRow(label, SeverityColor(inflightSeverity), fmt.Sprintf("%d active", disk.Inflight), SeverityColor(inflightSeverity), "", "", "", gaugeBarCell(inflightPct, activityWidth, SeverityColor(inflightSeverity), fmt.Sprintf("%d req", disk.Inflight))))
	}

	return rows
}

func sortedDiskStats(disks []core.DiskStat) []core.DiskStat {
	sorted := append([]core.DiskStat(nil), disks...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Device < sorted[j].Device })

	return sorted
}

// FilesystemLabel returns a compact display label for a mount path.
func FilesystemLabel(mount string) string {
	mount = strings.Trim(strings.TrimSpace(mount), "/")
	if mount == "" {
		return "FS /"
	}

	segments := strings.Split(mount, "/")
	if len(segments) == 2 && segments[0] == "mnt" {
		return "FS /mnt/" + segments[1]
	}

	last := compactPathSegment(segments[len(segments)-1], filesystemLastWidth)
	if len(segments) == 1 {
		return "FS " + last
	}

	prev := compactPathSegment(segments[len(segments)-2], filesystemParentWidth)
	label := "FS " + prev + "/" + last
	if ansi.VisibleLen(label) <= filesystemLabelMax {
		return label
	}

	return "FS " + compactPathSegment(segments[len(segments)-1], filesystemFallbackMax)
}

func compactPathSegment(segment string, maxWidth int) string {
	segment = strings.TrimSpace(segment)
	if maxWidth <= 0 || ansi.VisibleLen(segment) <= maxWidth {
		return segment
	}

	ext := path.Ext(segment)
	base := strings.TrimSuffix(segment, ext)
	if ext != "" {
		extWidth := runewidth.StringWidth(ext)
		if extWidth < maxWidth {
			headWidth := maxWidth - extWidth

			return runewidth.Truncate(base, headWidth, "") + ext
		}
	}

	return runewidth.Truncate(segment, maxWidth, "")
}
