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
	rows := make([]TableRowSpec, 0, storageBaseRowsCap+len(s.Filesystems))

	if s.RootTotalKiB > 0 {
		freePct := metrics.DiskFreePercent(s)
		rows = append(rows, TableFullRow("Disk Source", ansi.Cyan, fmt.Sprintf("%s %s free", metrics.DiskSourceText(s), FormatKiBValue(metrics.DiskFreeKiB(s))), ansi.Cyan, "", percentDisplay(freePct)+" free", ansi.Green, ""))

		diskSeverity := memorySeverity(s.RootUsedPercent)
		rows = append(rows, TableFullRow("Disk /", SeverityColor(diskSeverity), formatKiBPair(s.RootUsedKiB, s.RootTotalKiB), SeverityColor(diskSeverity), "", "", "", gaugeCell(s.RootUsedPercent, activityWidth, diskSeverity)))

		if rootFS, ok := metrics.RootFilesystem(s); ok && rootFS.InodesUsedPercent >= 0 {
			inodeSeverity := inodeUsageSeverity(rootFS.InodesUsedPercent)
			rows = append(rows, TableFullRow("Inodes /", SeverityColor(inodeSeverity), percentDisplay(rootFS.InodesUsedPercent)+" used", SeverityColor(inodeSeverity), "", "", "", gaugeBarCell(rootFS.InodesUsedPercent, activityWidth, SeverityColor(inodeSeverity), percentDisplay(rootFS.InodesUsedPercent))))
		}
	}

	if s.DiskReadBps >= 0 || s.DiskWriteBps >= 0 {
		diskIOSeverity := diskUtilSeverity(s.DiskUtil)
		rows = append(rows, TableFullRow("Disk IO "+fallbackString(s.DiskDevice, TextNA), SeverityColor(diskIOSeverity), fmt.Sprintf("R %s • W %s", formatBps(s.DiskReadBps), formatBps(s.DiskWriteBps)), SeverityColor(diskIOSeverity), "", "", "", gaugeCell(s.DiskUtil, activityWidth, diskIOSeverity)))

		if s.DiskAwaitMS >= 0 || s.DiskQueueDepth >= 0 {
			latencySeverity := mergeSeverity(diskAwaitSeverity(s.DiskAwaitMS), diskQueueSeverity(s.DiskQueueDepth))
			rows = append(rows, TableFullRow("Disk Latency", SeverityColor(latencySeverity), "await "+FormatMillisValue(s.DiskAwaitMS), SeverityColor(latencySeverity), "", "queue "+FormatQueueDepth(s.DiskQueueDepth), SeverityColor(latencySeverity), ""))
		}
	}

	if !condensed && (s.DiskReadMergedPerSec >= 0 || s.DiskWriteMergedPerSec >= 0) {
		rows = append(rows, TableFullRow("Disk Merge", ansi.Cyan, fmt.Sprintf("R %s • W %s", formatOpsPerSec(s.DiskReadMergedPerSec), formatOpsPerSec(s.DiskWriteMergedPerSec)), ansi.Cyan, "", "merged ops", ansi.Cyan, ""))
	}

	if !condensed && s.DiskInflight >= 0 {
		inflightSeverity := diskQueueSeverity(float64(s.DiskInflight))
		inflightPct := clamp(s.DiskInflight*diskInflightScale, percentMin, percentMax)
		rows = append(rows, TableFullRow(LabelDiskInflight, SeverityColor(inflightSeverity), fmt.Sprintf("%d active", s.DiskInflight), SeverityColor(inflightSeverity), "", "", "", gaugeBarCell(inflightPct, activityWidth, SeverityColor(inflightSeverity), fmt.Sprintf("%d req", s.DiskInflight))))
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
			fsSeverity := memorySeverity(fs.UsedPercent)
			fsSuffix := percentDisplay(fs.UsedPercent)
			if fs.InodesUsedPercent >= 0 {
				fsSuffix = "inode " + percentDisplay(fs.InodesUsedPercent)
			}
			rows = append(rows, TableFullRow(FilesystemLabel(fs.Mount), SeverityColor(fsSeverity), formatKiBPair(fs.UsedKiB, fs.TotalKiB), SeverityColor(fsSeverity), "", "", "", gaugeBarCell(fs.UsedPercent, activityWidth, SeverityColor(fsSeverity), fsSuffix)))
		}
	}

	return rows
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
