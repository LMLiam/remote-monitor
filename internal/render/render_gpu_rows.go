package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

func buildGPURows(state core.AppState, valueWidth, activityWidth int, condensed bool) []TableRowSpec {
	s := state.Current
	if len(s.GPUs) == 0 {
		return []TableRowSpec{TableFullRow(LabelGPU, ansi.Yellow, "unavailable", ansi.Yellow, "", "supported GPU unavailable/no GPUs exposed", ansi.Yellow, "")}
	}

	rows := make([]TableRowSpec, 0, len(s.GPUs)*gpuRowsPerDevice)
	for idx, gpu := range s.GPUs {
		if idx > 0 {
			rows = append(rows, tableDividerRow())
		}

		rows = appendGPUUtilRows(rows, gpu, activityWidth, condensed)
		rows = appendGPUSensorRows(rows, gpu, activityWidth, condensed, valueWidth)
		rows = appendGPUClockRows(rows, gpu, activityWidth, condensed)
		rows = appendGPULinkRows(rows, gpu, activityWidth, condensed)
		rows = appendGPUThrottleRows(rows, gpu, valueWidth, condensed)
	}

	return rows
}

func appendGPUUtilRows(rows []TableRowSpec, gpu core.GPUStat, activityWidth int, condensed bool) []TableRowSpec {
	utilSeverityValue := UtilSeverity(gpu.Util)
	rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Load", gpu.Index), ansi.Cyan, percentDisplay(gpu.Util), SeverityColor(utilSeverityValue), "", "", "", gaugeCell(gpu.Util, activityWidth, utilSeverityValue)))

	memUtilSeverity := UtilSeverity(gpu.MemUtil)
	rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Mem Util", gpu.Index), ansi.Blue, percentDisplay(gpu.MemUtil), SeverityColor(memUtilSeverity), "", "", "", gaugeCell(gpu.MemUtil, activityWidth, memUtilSeverity)))

	if !condensed && gpu.EncoderUtil >= 0 {
		encSeverity := UtilSeverity(gpu.EncoderUtil)
		rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Encoder", gpu.Index), SeverityColor(encSeverity), percentDisplay(gpu.EncoderUtil), SeverityColor(encSeverity), "", "", "", gaugeCell(gpu.EncoderUtil, activityWidth, encSeverity)))
	}

	if !condensed && gpu.DecoderUtil >= 0 {
		decSeverity := UtilSeverity(gpu.DecoderUtil)
		rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Decoder", gpu.Index), SeverityColor(decSeverity), percentDisplay(gpu.DecoderUtil), SeverityColor(decSeverity), "", "", "", gaugeCell(gpu.DecoderUtil, activityWidth, decSeverity)))
	}

	return rows
}

func appendGPUSensorRows(rows []TableRowSpec, gpu core.GPUStat, activityWidth int, condensed bool, valueWidth int) []TableRowSpec {
	vramPct := metrics.PercentOf(gpu.MemUsed, gpu.MemTotal)
	vramSeverity := memorySeverity(vramPct)
	rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d VRAM", gpu.Index), ansi.Lav, formatMiBPair(gpu.MemUsed, gpu.MemTotal), SeverityColor(vramSeverity), "", "", "", gaugeCell(vramPct, activityWidth, vramSeverity)))

	tempSeverityValue := temperatureSeverity(gpu.Temp)
	rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Temp", gpu.Index), ansi.Amber, tempDisplay(gpu.Temp), SeverityColor(tempSeverityValue), "", "", "", gaugeBarCell(metrics.TemperaturePercent(gpu.Temp), activityWidth, SeverityColor(tempSeverityValue), tempDisplay(gpu.Temp))))

	powerPct := metrics.PowerPercent(gpu.PowerDraw, gpu.PowerLimit)
	powerSeverityValue := powerSeverity(gpu.PowerDraw, gpu.PowerLimit)
	rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Power", gpu.Index), ansi.Amber, formatPowerPair(gpu.PowerDraw, gpu.PowerLimit), SeverityColor(powerSeverityValue), "", "", "", gaugeBarCell(powerPct, activityWidth, SeverityColor(powerSeverityValue), percentDisplay(powerPct))))

	fanSeverityValue := fanSeverity(gpu.Fan)
	if !condensed && gpu.Fan >= 0 {
		rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Fan", gpu.Index), SeverityColor(fanSeverityValue), percentDisplay(gpu.Fan), SeverityColor(fanSeverityValue), "", "", "", gaugeBarCell(gpu.Fan, activityWidth, SeverityColor(fanSeverityValue), percentDisplay(gpu.Fan))))
	}

	if strings.TrimSpace(gpu.PState) != "" {
		rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d PState", gpu.Index), ansi.Lav, "", "", chipCell(strings.ToUpper(gpu.PState), valueWidth, pStateBackground(gpu.PState)), pStateMeaning(gpu.PState), ansi.Lav, ""))
	}

	return rows
}

func appendGPUClockRows(rows []TableRowSpec, gpu core.GPUStat, activityWidth int, condensed bool) []TableRowSpec {
	smClockPct := metrics.ClockPercent(gpu.SMClock, gpu.MaxSMClock)
	smClockSeverity := clockSeverity(gpu.SMClock, gpu.MaxSMClock)
	rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d SM", gpu.Index), SeverityColor(smClockSeverity), formatClockPair(gpu.SMClock, gpu.MaxSMClock), SeverityColor(smClockSeverity), "", "", "", gaugeBarCell(smClockPct, activityWidth, SeverityColor(smClockSeverity), percentDisplay(smClockPct))))

	memClockPct := metrics.ClockPercent(gpu.MemClock, gpu.MaxMemClock)
	memClockSeverity := clockSeverity(gpu.MemClock, gpu.MaxMemClock)
	if gpu.MemClock >= 0 || gpu.MaxMemClock > 0 {
		rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Mem", gpu.Index), SeverityColor(memClockSeverity), formatClockPair(gpu.MemClock, gpu.MaxMemClock), SeverityColor(memClockSeverity), "", "", "", gaugeBarCell(memClockPct, activityWidth, SeverityColor(memClockSeverity), percentDisplay(memClockPct))))
	}

	if !condensed && gpu.GraphicsClock >= 0 {
		graphicsPct := metrics.ClockPercent(gpu.GraphicsClock, gpu.MaxSMClock)
		graphicsSeverity := clockSeverity(gpu.GraphicsClock, gpu.MaxSMClock)
		rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Graphics", gpu.Index), SeverityColor(graphicsSeverity), formatClockValue(gpu.GraphicsClock), SeverityColor(graphicsSeverity), "", "", "", gaugeBarCell(graphicsPct, activityWidth, SeverityColor(graphicsSeverity), percentDisplay(graphicsPct))))
	}

	if !condensed && gpu.VideoClock >= 0 {
		videoPct := metrics.ClockPercent(gpu.VideoClock, gpu.MaxSMClock)
		videoSeverity := clockSeverity(gpu.VideoClock, gpu.MaxSMClock)
		rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d Video", gpu.Index), SeverityColor(videoSeverity), formatClockValue(gpu.VideoClock), SeverityColor(videoSeverity), "", "", "", gaugeBarCell(videoPct, activityWidth, SeverityColor(videoSeverity), percentDisplay(videoPct))))
	}

	return rows
}

func appendGPULinkRows(rows []TableRowSpec, gpu core.GPUStat, activityWidth int, condensed bool) []TableRowSpec {
	if !condensed && (gpu.PCIeGenCurrent >= 0 || gpu.PCIeWidthCurrent >= 0 || gpu.PCIeGenMax > 0 || gpu.PCIeWidthMax > 0) {
		rows = append(rows, TableFullRow(fmt.Sprintf("GPU%d PCIe", gpu.Index), ansi.Lav, formatPCIeLinkCurrent(gpu.PCIeGenCurrent, gpu.PCIeWidthCurrent), ansi.Lav, "", "", "", gaugeBarCell(pcieLinkPercent(gpu.PCIeGenCurrent, gpu.PCIeGenMax, gpu.PCIeWidthCurrent, gpu.PCIeWidthMax), activityWidth, ansi.Lav, formatPCIeLinkMax(gpu.PCIeGenMax, gpu.PCIeWidthMax))))
	}

	return rows
}

func appendGPUThrottleRows(rows []TableRowSpec, gpu core.GPUStat, valueWidth int, condensed bool) []TableRowSpec {
	if condensed || strings.TrimSpace(gpu.ThrottleReasons) == "" {
		return rows
	}

	reasons := gpu.ThrottleReasons
	throttleSev := throttleSeverity(reasons)
	throttleNote := "no active limiters"
	if strings.TrimSpace(strings.ToLower(reasons)) != TextNone {
		throttleNote = "clock limiters"
	}

	return append(rows, TableFullRow(fmt.Sprintf("GPU%d Throttle", gpu.Index), SeverityColor(throttleSev), "", "", chipCell(strings.ToUpper(reasons), valueWidth, severityBackground(throttleSev)), throttleNote, SeverityColor(throttleSev), ""))
}

func pStateBackground(pstate string) string {
	switch strings.ToUpper(strings.TrimSpace(pstate)) {
	case "P0", "P1":
		return ansi.AmberBg
	case "P2", "P3":
		return ansi.BlueBg
	case "P4", "P5":
		return ansi.LavBg
	default:
		return ansi.MutedBg
	}
}
