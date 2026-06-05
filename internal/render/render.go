package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"github.com/lmliam/remote-monitor/internal/render/banner"
	"strconv"
	"strings"
	"time"
)

const (
	frameHorizontalMargin = 2
	frameMinWidth         = 92

	reconnectCriticalCount = 3

	headerColumnCount  = 3
	headerSideColumns  = 2
	headerRightReserve = 6
)

// NonInteractive returns the compact line-oriented monitor summary.
func NonInteractive(state core.AppState) string {
	status := currentStatus(state)
	s := state.Current

	line := fmt.Sprintf(
		"%s | state %s | CPU %s | RAM %s | NET RX %s TX %s | GPU %s | VRAM %s | TEMP %s",
		fallbackString(s.RemoteTimestamp, time.Now().Format("2006-01-02 15:04:05")),
		status,
		percentDisplay(s.CPUPercent),
		formatMiBPair(s.RAMUsedMiB, s.RAMTotalMiB),
		formatBps(metrics.TotalNetRXBps(s)),
		formatBps(metrics.TotalNetTXBps(s)),
		percentDisplay(metrics.OverallGPUUtil(s)),
		percentDisplay(metrics.OverallVRAMPct(s)),
		tempDisplay(metrics.OverallTempValue(s)),
	)
	if powerText := PowerSummaryText(s); powerText != "" {
		line += " | Power " + powerText
	}

	return line
}

// TTYFrame returns a viewport frame wrapped in terminal repaint escapes.
func TTYFrame(state core.AppState, width, height int) string {
	frame, _ := ViewportFrame(state, width, height, state.ScrollOffset)

	return TTYSequence(frame)
}

// TTYSequence converts a rendered frame into an in-place terminal update.
func TTYSequence(frame string) string {
	lines := strings.Split(strings.TrimRight(frame, "\n"), "\n")
	var b strings.Builder
	b.WriteString(cursorHome)
	for i, line := range lines {
		if i > 0 {
			// Move to the next row and explicitly return to column zero before
			// clearing/repainting. Some terminals treat LF without CR as a pure
			// line-feed, which can cause subsequent rows to start at the prior
			// column and smear the dashboard horizontally.
			b.WriteString("\r\n")
		}
		// Clear each line before repainting so blank spacer rows and shorter
		// updates don't leave stale content behind, without flashing the whole
		// screen between frames.
		b.WriteString(clearLine)
		b.WriteString(line)
	}
	// Clear any leftover rows below the current frame when the layout shrinks.
	b.WriteString(clearBelow)

	return b.String()
}

// Frame returns the top of the dashboard frame for a terminal size.
func Frame(state core.AppState, width, height int) string {
	frame, _ := ViewportFrame(state, width, height, 0)

	return frame
}

// ViewportFrame returns a scrollable slice of the full dashboard frame.
func ViewportFrame(state core.AppState, width, height, scrollOffset int) (frame string, maxScroll int) {
	full := FullFrame(state, width, height)
	lines := SplitRenderedLines(full)
	if len(lines) == 0 {
		return "", 0
	}
	if height < 1 {
		height = 1
	}

	maxScroll = max(0, len(lines)-height)
	offset := clamp(scrollOffset, 0, maxScroll)
	end := min(offset+height, len(lines))

	return strings.Join(lines[offset:end], "\n"), maxScroll
}

// FullFrame renders the complete dashboard without viewport clipping.
func FullFrame(state core.AppState, width, height int) string {
	s := state.Current
	totalWidth := max(width-frameHorizontalMargin, frameMinWidth)

	var b strings.Builder
	now := time.Now()
	reconnectColor := ansi.Green
	if state.ReconnectCount > 0 {
		reconnectColor = ansi.Yellow
	}
	if state.ReconnectCount > reconnectCriticalCount {
		reconnectColor = ansi.Red
	}
	loadColor := SeverityColor(UtilSeverity(s.CPUPercent))
	status := currentStatus(state)
	statusText := inlineChip(strings.ToUpper(status), statusBackground(status))
	titleCfg := state.Cfg
	titleBlock := banner.TitleBlock(totalWidth, statusText, now, titleCfg)
	condensedTables := state.Cfg.Compact
	tableLayout := renderTableLayout(state, totalWidth, true, condensedTables)
	historyBox := HistoryBox(state, totalWidth)
	leftHeaderWidth, middleHeaderWidth, rightHeaderWidth := headerWidths(totalWidth)
	b.WriteString(titleBlock)
	b.WriteString("\n")
	b.WriteString(headerLine(
		HeaderPair("Target", state.Cfg.Host, ansi.Cyan, leftHeaderWidth),
		HeaderPair("Remote", fallbackString(s.RemoteName, TextNA), ansi.Green, middleHeaderWidth),
		HeaderPair("Uptime", formatUptime(s.UptimeSeconds), ansi.Amber, rightHeaderWidth),
		totalWidth,
	))
	b.WriteString("\n")
	b.WriteString(headerLine(
		HeaderPair("Updated", fallbackString(s.RemoteTimestamp, "waiting"), ansi.Lav, leftHeaderWidth),
		HeaderPair("Fresh", freshnessText(state), statusColor(status), middleHeaderWidth),
		HeaderPair("Every", fmt.Sprintf("%ds", int(state.Cfg.Interval/time.Second)), ansi.Blue, rightHeaderWidth),
		totalWidth,
	))
	b.WriteString("\n")
	b.WriteString(headerLine(
		HeaderPair("Load", loadText(s), loadColor, leftHeaderWidth),
		HeaderPair("Samples", strconv.Itoa(state.SampleCount), ansi.Green, middleHeaderWidth),
		HeaderPair("Reconnect", strconv.Itoa(state.ReconnectCount), reconnectColor, rightHeaderWidth),
		totalWidth,
	))
	b.WriteString("\n")
	b.WriteString(headerLine(
		HeaderPair("Terminal", fmt.Sprintf("%dx%d", width, height), ansi.Cyan, leftHeaderWidth),
		HeaderPair("Mode", modeText(state.Cfg), ansi.Lav, middleHeaderWidth),
		HeaderPair("Health", HealthText(state), statusColor(status), rightHeaderWidth),
		totalWidth,
	))
	b.WriteString("\n\n")

	b.WriteString(tableLayout)
	b.WriteString("\n\n")
	b.WriteString(historyBox)

	return banner.DecorateFrame(b.String(), totalWidth, now, state.Cfg)
}

func headerWidths(totalWidth int) (left, middle, right int) {
	left = totalWidth / headerColumnCount
	middle = left
	right = totalWidth - headerSideColumns*left - headerRightReserve

	return left, middle, right
}

// SplitRenderedLines splits a rendered frame into display rows.
func SplitRenderedLines(s string) []string {
	trimmed := strings.TrimRight(s, "\n")
	if trimmed == "" {
		return nil
	}

	return strings.Split(trimmed, "\n")
}
