package render

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
)

func buildMemoryRows(state core.AppState, activityWidth int, condensed bool) []TableRowSpec {
	s := state.Current
	rows := make([]TableRowSpec, 0, memoryRowsCapacity)

	ramPct := metrics.PercentOf(s.RAMUsedMiB, s.RAMTotalMiB)
	ramSeverity := memorySeverity(ramPct)
	rows = append(rows, TableFullRow("RAM", SeverityColor(ramSeverity), formatMiBPair(s.RAMUsedMiB, s.RAMTotalMiB), SeverityColor(ramSeverity), "", "", "", gaugeCell(ramPct, activityWidth, ramSeverity)))

	if s.RAMAvailableMiB >= 0 {
		availPct := metrics.PercentOf(s.RAMAvailableMiB, s.RAMTotalMiB)
		availSeverity := availabilitySeverity(availPct)
		rows = append(rows, TableFullRow(LabelRAMAvail, SeverityColor(availSeverity), formatMiBValue(s.RAMAvailableMiB), SeverityColor(availSeverity), "", "", "", gaugeBarCell(availPct, activityWidth, SeverityColor(availSeverity), percentDisplay(availPct))))
	}

	if s.RAMFreeMiB >= 0 {
		freePct := metrics.PercentOf(s.RAMFreeMiB, s.RAMTotalMiB)
		freeSeverity := availabilitySeverity(freePct)
		rows = append(rows, TableFullRow(LabelRAMFree, SeverityColor(freeSeverity), formatMiBValue(s.RAMFreeMiB), SeverityColor(freeSeverity), "", "", "", gaugeBarCell(freePct, activityWidth, SeverityColor(freeSeverity), percentDisplay(freePct))))
	}

	if s.RAMCacheMiB >= 0 {
		cachePct := metrics.PercentOf(s.RAMCacheMiB, s.RAMTotalMiB)
		rows = append(rows, TableFullRow(LabelRAMCache, ansi.Blue, formatMiBValue(s.RAMCacheMiB), ansi.Blue, "", "", "", gaugeBarCell(cachePct, activityWidth, ansi.Blue, percentDisplay(cachePct))))
	}

	if !condensed && s.RAMBuffersMiB >= 0 {
		buffersPct := metrics.PercentOf(s.RAMBuffersMiB, s.RAMTotalMiB)
		rows = append(rows, TableFullRow("RAM Buffers", ansi.Cyan, formatMiBValue(s.RAMBuffersMiB), ansi.Cyan, "", "", "", gaugeBarCell(buffersPct, activityWidth, ansi.Cyan, percentDisplay(buffersPct))))
	}

	if !condensed && s.RAMReclaimableMiB >= 0 {
		reclaimPct := metrics.PercentOf(s.RAMReclaimableMiB, s.RAMTotalMiB)
		rows = append(rows, TableFullRow("RAM Reclaim", ansi.Green, formatMiBValue(s.RAMReclaimableMiB), ansi.Green, "", "", "", gaugeBarCell(reclaimPct, activityWidth, ansi.Green, percentDisplay(reclaimPct))))
	}

	if !condensed && s.RAMSharedMiB >= 0 {
		sharedPct := metrics.PercentOf(s.RAMSharedMiB, s.RAMTotalMiB)
		sharedSeverity := UtilSeverity(sharedPct)
		rows = append(rows, TableFullRow("RAM Shared", SeverityColor(sharedSeverity), formatMiBValue(s.RAMSharedMiB), SeverityColor(sharedSeverity), "", "", "", gaugeBarCell(sharedPct, activityWidth, SeverityColor(sharedSeverity), percentDisplay(sharedPct))))
	}

	if s.SwapTotalKiB > 0 {
		swapUsed := s.SwapTotalKiB - s.SwapFreeKiB
		swapPct := metrics.PercentOf(swapUsed, s.SwapTotalKiB)
		swapSeverity := memorySeverity(swapPct)
		rows = append(rows, TableFullRow("Swap", SeverityColor(swapSeverity), formatKiBPair(swapUsed, s.SwapTotalKiB), SeverityColor(swapSeverity), "", "", "", gaugeCell(swapPct, activityWidth, swapSeverity)))
	}

	if !condensed && (s.SwapInBps >= 0 || s.SwapOutBps >= 0) {
		rows = append(rows, TableFullRow("Swap IO", ansi.Lav, "in "+formatBps(s.SwapInBps), ansi.Lav, "", "out "+formatBps(s.SwapOutBps), ansi.Lav, ""))
	}

	if s.MemPressureSomeAvg10 >= 0 || s.MemPressureFullAvg10 >= 0 {
		pressurePeak := s.MemPressureSomeAvg10
		if s.MemPressureFullAvg10 > pressurePeak {
			pressurePeak = s.MemPressureFullAvg10
		}
		pressureSeverity := mergeSeverity(psiSeverity(s.MemPressureSomeAvg10), psiSeverity(s.MemPressureFullAvg10))
		rows = append(rows, TableFullRow(LabelMemPSI, SeverityColor(pressureSeverity), "some "+formatPSIValue(s.MemPressureSomeAvg10), SeverityColor(pressureSeverity), "", "", "", gaugeBarCell(psiPercent(pressurePeak), activityWidth, SeverityColor(pressureSeverity), "full "+formatPSIValue(s.MemPressureFullAvg10))))
	}

	return rows
}
