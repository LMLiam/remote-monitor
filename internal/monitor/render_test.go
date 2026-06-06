package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-runewidth"
)

const (
	clearBelow = "\x1b[J"
	clearLine  = "\x1b[2K"
	cursorHome = "\x1b[H"
)

var ansiEscapeRE = regexp.MustCompile("\x1b\\[[0-9;?]*[- !\"#$%&'()*+,./]*(?:[@ABCDEFGHIJKLMNOPQRSTUVWXYZ\\[\\\\\\]^_`abcdefghijklmnopqrstuvwxyz{|}~])")

type ttyScreenCursor struct {
	row int
	col int
}

func applyTTYSequence(screen []string, sequence string) []string {
	updated := append([]string(nil), screen...)
	cursor := ttyScreenCursor{
		row: 0,
		col: 0,
	}
	for sequence != "" {
		switch {
		case strings.HasPrefix(sequence, cursorHome):
			cursor.row = 0
			cursor.col = 0
			sequence = sequence[len(cursorHome):]
		case strings.HasPrefix(sequence, clearLine):
			clearTTYLineForTest(updated, cursor.row)
			sequence = sequence[len(clearLine):]
		case strings.HasPrefix(sequence, clearBelow):
			clearTTYBelowForTest(updated, cursor)
			sequence = sequence[len(clearBelow):]
		case sequence[0] == '\r':
			cursor.col = 0
			sequence = sequence[1:]
		case sequence[0] == '\n':
			cursor.row = max(cursor.row+1, 0)
			sequence = sequence[1:]
		case sequence[0] == '\x1b':
			sequence = consumeANSIEscapeForTest(sequence)
		default:
			sequence = applyTTYTextForTest(updated, &cursor, sequence)
		}
	}

	return updated
}

func clearTTYLineForTest(screen []string, row int) {
	if row >= 0 && row < len(screen) {
		screen[row] = ""
	}
}

func clearTTYBelowForTest(screen []string, cursor ttyScreenCursor) {
	if cursor.row >= 0 && cursor.row < len(screen) {
		screen[cursor.row] = visiblePrefix(screen[cursor.row], cursor.col)
	}
	for row := max(cursor.row+1, 0); row < len(screen); row++ {
		screen[row] = ""
	}
}

func consumeANSIEscapeForTest(sequence string) string {
	match := ansiEscapeRE.FindStringIndex(sequence)
	if len(match) == 2 && match[0] == 0 {
		return sequence[match[1]:]
	}

	return sequence[1:]
}

func applyTTYTextForTest(screen []string, cursor *ttyScreenCursor, sequence string) string {
	next := strings.IndexAny(sequence, "\r\n\x1b")
	if next == -1 {
		next = len(sequence)
	}
	text := sequence[:next]
	if cursor.row >= 0 && cursor.row < len(screen) {
		screen[cursor.row] = overlayTTYTextForTest(screen[cursor.row], cursor.col, text)
	}
	cursor.col += ansi.VisibleLen(text)

	return sequence[next:]
}

func overlayTTYTextForTest(line string, col int, text string) string {
	lineWidth := ansi.VisibleLen(line)
	if col > lineWidth {
		line += strings.Repeat(" ", col-lineWidth)
	}
	if col >= ansi.VisibleLen(line) {
		return line + text
	}

	return ansi.FitText(line, col) + text
}

func firstBannerColorEscape(s string) string {
	for _, prefix := range []string{"\x1b[38;2;", "\x1b[38;5;"} {
		start := strings.Index(s, prefix)
		if start < 0 {
			continue
		}
		rest := s[start:]
		end := strings.IndexByte(rest, 'm')
		if end < 0 {
			return ""
		}

		return rest[:end+1]
	}

	return ""
}

func lineHasAll(line string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(line, needle) {
			return false
		}
	}

	return true
}

func maxSectionTitlesOnLine(lines []string) int {
	titles := []string{
		"│ CPU",
		"│ GPU",
		"│ System",
		"│ Memory",
		"│ Storage",
		"│ Network",
		"│ Top Processes",
		"│ GPU Processes",
	}

	maxCount := 0
	for _, line := range lines {
		count := 0
		for _, title := range titles {
			if strings.Contains(line, title) {
				count++
			}
		}
		if count > maxCount {
			maxCount = count
		}
	}

	return maxCount
}

func visiblePrefix(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.VisibleLen(s) <= width {
		return s
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			rw = 1
		}
		if used+rw > width {
			break
		}
		b.WriteRune(r)
		used += rw
	}

	return b.String()
}

func TestHeaderPairUsesFilledChips(t *testing.T) {
	t.Parallel()

	got := render.HeaderPair("Target", testHost, ansi.Cyan, 24)
	if !strings.Contains(got, "TARGET") {
		t.Fatalf("HeaderPair missing uppercase key: %q", got)
	}
	if !strings.Contains(got, testHost) {
		t.Fatalf("HeaderPair missing value: %q", got)
	}
	if !strings.Contains(got, ansi.HeaderBg) || !strings.Contains(got, ansi.PanelAltBg) {
		t.Fatalf("HeaderPair missing filled chip backgrounds: %q", got)
	}
}

func TestHeaderPairKeepsValueColumnAligned(t *testing.T) {
	t.Parallel()

	target := ansi.StripANSI(render.HeaderPair("Target", testHost, ansi.Cyan, 32))
	updated := ansi.StripANSI(render.HeaderPair("Updated", "2026-05-28 22:50:17", ansi.Lav, 32))

	before, _, ok := strings.Cut(target, testHost)
	if !ok {
		t.Fatalf("target header missing value: %q", target)
	}
	before0, _, ok0 := strings.Cut(updated, "2026-05-28 22:50:17")
	if !ok0 {
		t.Fatalf("updated header missing value: %q", updated)
	}

	got := runewidth.StringWidth(before)
	want := runewidth.StringWidth(before0)
	if got != want {
		t.Fatalf("header values start at different Columns: target=%d updated=%d", got, want)
	}
}

func TestGaugeUsesTrackBackgroundAndHalfBlocks(t *testing.T) {
	t.Parallel()

	got := render.Gauge(14, 24, "info")
	if !strings.Contains(got, ansi.Blue) {
		t.Fatalf("Gauge missing info color: %q", got)
	}
	if !strings.Contains(got, ansi.TrackBg) {
		t.Fatalf("Gauge missing track background: %q", got)
	}
	if !strings.Contains(got, "▌") {
		t.Fatalf("Gauge missing half-block dense fill: %q", got)
	}
}

func TestGaugeLowNonZeroPercentUsesVisibleLeadMarker(t *testing.T) {
	t.Parallel()

	got := render.Gauge(1, 24, "info")
	if !strings.Contains(got, ansi.BlueBg) {
		t.Fatalf("low-end Gauge missing accent marker background: %q", got)
	}
	if !strings.Contains(got, "▌") {
		t.Fatalf("low-end Gauge missing visible lead marker Glyph: %q", got)
	}
}

func TestGaugeZeroPercentDoesNotUseAccentMarker(t *testing.T) {
	t.Parallel()

	got := render.Gauge(0, 24, "info")
	if strings.Contains(got, ansi.BlueBg) {
		t.Fatalf("zero Gauge should not render low-end accent marker: %q", got)
	}
}

func TestGPUValueUsesTruncateWithoutMidPadding(t *testing.T) {
	t.Parallel()

	name := ansi.TruncateText(testGPUName, 8)
	got := "X " + name + " • mem 17%"
	if strings.Contains(got, "        •") {
		t.Fatalf("gpu value contains padded gap before bullet: %q", got)
	}
}

func TestLowUtilMapsToInfoBlue(t *testing.T) {
	t.Parallel()

	thresholds := core.DefaultThresholds()
	if got := render.UtilSeverity(1, thresholds); got != "info" {
		t.Fatalf("UtilSeverity(1) = %q", got)
	}
	if got := render.SeverityColor(render.UtilSeverity(1, thresholds)); got != ansi.Blue {
		t.Fatalf("SeverityColor(UtilSeverity(1)) = %q", got)
	}
}

func TestHealthTextStaleDoesNotClaimStreamHealthy(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) { cfg.StaleAfter = time.Second })
		state.HasSample = true
		state.StreamAlive = true
		state.LastRx = time.Now().Add(-3 * time.Second)
		state.RuntimeState = core.StatusLive
	})

	if got := render.HealthText(state); got != "waiting for fresh Sample" {
		t.Fatalf("HealthText(stale) = %q", got)
	}
}

func TestAlertSummaryUsesExpandedSignals(t *testing.T) {
	t.Parallel()

	state := testState(func(state *core.AppState) {
		state.Cfg = testConfig(func(cfg *core.Config) { cfg.StaleAfter = time.Hour })
		state.HasSample = true
		state.RuntimeState = core.StatusLive
		state.StreamAlive = true
		state.LastRx = time.Now()
		state.Current = testSample(func(smp *core.Sample) {
			smp.CPUTempC = 86
			smp.RAMTotalMiB = 15967
			smp.RAMAvailableMiB = 512
			smp.DiskAwaitMS = 55
			smp.DiskQueueDepth = 0.50
		})
	})

	severity, text := render.AlertSummary(state)
	if severity != "critical" {
		t.Fatalf("alert severity = %q", severity)
	}
	for _, want := range []string{"cpu hot", "ram low", "disk latency"} {
		if !strings.Contains(text, want) {
			t.Fatalf("alert text missing %q in %q", want, text)
		}
	}
}
