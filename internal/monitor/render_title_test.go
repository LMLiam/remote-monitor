package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"github.com/lmliam/remote-monitor/internal/render/banner"
	"strings"
	"testing"
	"time"
)

func TestAnimatedTitleChangesColorButNotText(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")
	for _, theme := range []string{core.ThemeAurora, core.ThemeBasic, core.ThemeWindowsXP} {
		first := banner.TitleBlock(170, status, time.Unix(0, 0), testConfig(func(cfg *core.Config) { cfg.Theme = theme }))
		second := banner.TitleBlock(170, status, time.Unix(0, 100*int64(time.Millisecond)), testConfig(func(cfg *core.Config) { cfg.Theme = theme }))

		if first == second {
			t.Fatalf("expected %s title colors to change over time", theme)
		}
		if ansi.StripANSI(first) != ansi.StripANSI(second) {
			t.Fatalf("expected %s title text content to remain stable", theme)
		}
		if !strings.Contains(first, "\x1b[38;2;") {
			t.Fatalf("expected %s animated title to use truecolor escapes", theme)
		}
	}
}

func TestBannerPaletteIsDenseEnoughForSmoothMotion(t *testing.T) {
	t.Parallel()

	if len(banner.Palette()) < 48 {
		t.Fatalf("banner palette too short: %d", len(banner.Palette()))
	}
}

func TestAnimatedTitleUsesSingleColorPerBannerRow(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")
	title := banner.TitleBlock(170, status, time.Unix(0, 0), testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeBasic }))
	lines := strings.Split(title, "\n")

	if len(lines) < len(banner.BasicBannerLines()) {
		t.Fatalf("title has too few lines: %d", len(lines))
	}

	for idx := range banner.BasicBannerLines() {
		if got := strings.Count(lines[idx], "\x1b[38;2;"); got != 1 {
			t.Fatalf("banner row %d should have one palette color, got %d in %q", idx, got, lines[idx])
		}
	}
}

func TestAnimatedTitleUsesColorBandsAcrossRows(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")
	title := banner.TitleBlock(170, status, time.Unix(0, 0), testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeBasic }))
	lines := strings.Split(title, "\n")

	if len(lines) < 6 {
		t.Fatalf("title has too few lines: %d", len(lines))
	}

	for idx := 0; idx < 6; idx += 2 {
		if firstBannerColorEscape(lines[idx]) != firstBannerColorEscape(lines[idx+1]) {
			t.Fatalf("expected banner rows %d and %d to share the same color band", idx, idx+1)
		}
	}
}

func TestThemeBannersUseDistinctArtAndPalette(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")

	rendered := map[string]string{}
	for _, theme := range []string{core.ThemeBasic, core.ThemeAurora, core.ThemeWindowsXP} {
		title := banner.TitleBlock(170, status, time.Unix(0, 0), testConfig(func(cfg *core.Config) { cfg.Theme = theme }))
		rendered[theme] = title
	}

	if !strings.Contains(ansi.StripANSI(rendered[core.ThemeBasic]), "██████╗") {
		t.Fatalf("expected basic banner to keep the current block art, got %q", ansi.StripANSI(rendered[core.ThemeBasic]))
	}

	if !strings.Contains(ansi.StripANSI(rendered[core.ThemeAurora]), "██▀███  ▓█████  ███▄ ▄███▓") {
		t.Fatalf("expected aurora banner to use the custom REMOTE MONITOR wordmark, got %q", ansi.StripANSI(rendered[core.ThemeAurora]))
	}
	if ansi.StripANSI(rendered[core.ThemeAurora]) == ansi.StripANSI(rendered[core.ThemeBasic]) {
		t.Fatal("expected aurora banner art to differ from basic")
	}
	if !strings.Contains(rendered[core.ThemeAurora], "\x1b[48;2;") {
		t.Fatal("expected aurora banner to paint a truecolor backdrop")
	}

	if firstBannerColorEscape(rendered[core.ThemeBasic]) == firstBannerColorEscape(rendered[core.ThemeAurora]) {
		t.Fatal("expected aurora and basic banners to use different starting palette colors")
	}
	if !strings.Contains(ansi.StripANSI(rendered[core.ThemeWindowsXP]), `/\  == \`) {
		t.Fatalf("expected windows-xp banner to use the provided wordmark art, got %q", ansi.StripANSI(rendered[core.ThemeWindowsXP]))
	}
	if ansi.StripANSI(rendered[core.ThemeWindowsXP]) == ansi.StripANSI(rendered[core.ThemeBasic]) {
		t.Fatal("expected windows-xp banner art to differ from basic")
	}
	if ansi.StripANSI(rendered[core.ThemeWindowsXP]) == ansi.StripANSI(rendered[core.ThemeAurora]) {
		t.Fatal("expected windows-xp banner art to differ from aurora")
	}
	if firstBannerColorEscape(rendered[core.ThemeWindowsXP]) == firstBannerColorEscape(rendered[core.ThemeAurora]) {
		t.Fatal("expected windows-xp and aurora banners to use different starting palette colors")
	}
}

func TestWindowsXPBannerUsesProvidedWordmarkArt(t *testing.T) {
	t.Parallel()

	lines := banner.WindowsXPBannerLines()
	if len(lines) != 5 {
		t.Fatalf("windows-xp banner line count = %d, want 5", len(lines))
	}
	if strings.TrimRight(lines[0], " ") != ` ______     ______     __    __     ______     ______   ______        __    __     ______     __   __     __     ______   ______     ______` {
		t.Fatalf("windows-xp banner first line changed: %q", lines[0])
	}
	for rowIdx, line := range lines {
		if got := ansi.VisibleLen(line); got != ansi.VisibleLen(lines[0]) {
			t.Fatalf("windows-xp banner row %d width = %d, want %d", rowIdx, got, ansi.VisibleLen(lines[0]))
		}
	}
}

func TestAuroraBannerCanvasHasNoShadowCells(t *testing.T) {
	t.Parallel()

	if len(banner.AuroraBannerCanvas()) != len(banner.AuroraFaceLines()) {
		t.Fatalf("aurora banner canvas height = %d, want %d", len(banner.AuroraBannerCanvas()), len(banner.AuroraFaceLines()))
	}

	for rowIdx, row := range banner.AuroraBannerCanvas() {
		for colIdx, cell := range row {
			if cell.Kind == banner.CellShadow {
				t.Fatalf("unexpected shadow cell at row %d col %d", rowIdx, colIdx)
			}
		}
	}
}

func TestAuroraBannerKeepsFaceArtReadable(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")
	title := banner.TitleBlock(174, status, time.Unix(0, 0), testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeAurora }))
	lines := strings.Split(title, "\n")

	if len(lines) < len(banner.AuroraFaceLines()) {
		t.Fatalf("title has too few lines: %d", len(lines))
	}

	for idx := range banner.AuroraFaceLines() {
		var expected strings.Builder
		for _, cell := range banner.AuroraBannerCanvas()[idx] {
			if cell.Kind == banner.CellFace {
				expected.WriteRune(cell.Glyph)

				continue
			}
			expected.WriteRune(' ')
		}
		got := ansi.StripANSI(lines[idx])
		wantFace := strings.TrimSpace(expected.String())
		if !strings.Contains(got, wantFace) {
			t.Fatalf("aurora banner row %d lost readability\nwant face %q inside %q", idx, wantFace, got)
		}
		if gotWidth := ansi.VisibleLen(lines[idx]); gotWidth != 174 {
			t.Fatalf("aurora banner row %d width = %d, want 174", idx, gotWidth)
		}
	}
}

func TestAuroraTitleBackgroundUsesHalfBlockCells(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")
	title := banner.TitleBlock(174, status, time.Unix(0, 0), testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeAurora }))
	cleaned := ansi.StripANSI(title)
	expectedHalfBlocks := 0
	for _, line := range banner.AuroraFaceLines() {
		expectedHalfBlocks += strings.Count(line, "▀")
	}

	if got := strings.Count(cleaned, "▀"); got <= expectedHalfBlocks {
		t.Fatalf("expected aurora title background to add half-block cells beyond wordmark art, got %d half-blocks, want more than %d in %q", got, expectedHalfBlocks, cleaned)
	}
	if !strings.Contains(cleaned, "██▀███  ▓█████  ███▄ ▄███▓") {
		t.Fatalf("expected aurora wordmark to remain readable, got %q", cleaned)
	}
}

func TestAuroraFrameAppliesBackdropBehindDashboard(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	state := testTUIState()
	state.Cfg.Theme = core.ThemeAurora
	auroraFrame := render.Frame(state, 176, 92)
	if !strings.Contains(auroraFrame, "\x1b[48;2;") {
		t.Fatal("expected aurora frame to include truecolor backdrop fills")
	}

	state.Cfg.Theme = core.ThemeBasic
	basicFrame := render.Frame(state, 176, 92)
	if strings.Contains(basicFrame, "\x1b[48;2;") {
		t.Fatal("expected basic frame to avoid aurora truecolor backdrop fills")
	}
}

func TestWindowsXPFrameRestylesDashboardSurface(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	state := testTUIState()
	state.Cfg.Theme = core.ThemeWindowsXP
	xpFrame := render.Frame(state, 176, 92)

	if !strings.Contains(xpFrame, "\x1b[48;2;") {
		t.Fatal("expected windows-xp frame to include truecolor surface fills")
	}
	if strings.Contains(xpFrame, ansi.BorderColor) {
		t.Fatal("expected windows-xp frame to replace the default border color")
	}
	if !strings.Contains(ansi.StripANSI(xpFrame), core.ThemeWindowsXP) {
		t.Fatalf("expected frame mode text to include %q", core.ThemeWindowsXP)
	}

	state.Cfg.Theme = core.ThemeBasic
	basicFrame := render.Frame(state, 176, 92)
	if xpFrame == basicFrame {
		t.Fatal("expected windows-xp frame to differ from basic")
	}
}

func TestWindowsXPThemeHonorsNoTrueColorFlag(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")
	title := banner.TitleBlock(170, status, time.Unix(0, 0), testConfig(func(cfg *core.Config) {
		cfg.Theme = core.ThemeWindowsXP
		cfg.DisableTrueColor = true
	}))
	state := testTUIState()
	state.Cfg.Theme = core.ThemeWindowsXP
	state.Cfg.DisableTrueColor = true
	frame := render.Frame(state, 176, 92)

	for name, rendered := range map[string]string{"title": title, "frame": frame} {
		if !strings.Contains(rendered, "\x1b[38;5;") {
			t.Fatalf("expected %s to use 256-color escapes", name)
		}
		if strings.Contains(rendered, "\x1b[38;2;") || strings.Contains(rendered, "\x1b[48;2;") {
			t.Fatalf("expected %s to avoid truecolor escapes", name)
		}
	}
}

func TestWindowsXPTitleTextAndWidthsStayStableAcrossAnimationFrames(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")
	cfg := testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeWindowsXP })
	first := banner.TitleBlock(170, status, time.Unix(0, 0), cfg)
	second := banner.TitleBlock(170, status, time.Unix(0, 180*int64(time.Millisecond)), cfg)

	if first == second {
		t.Fatal("expected windows-xp title colors to change over time")
	}
	if ansi.StripANSI(first) != ansi.StripANSI(second) {
		t.Fatal("expected windows-xp title text content to remain stable")
	}

	assertRenderedLinesWidth(t, first, 170)
	assertRenderedLinesWidth(t, second, 170)
}

func TestWindowsXPFallbackLayoutsStayStable(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")

	for name, cfg := range map[string]core.Config{
		"compact": testConfig(func(cfg *core.Config) {
			cfg.Theme = core.ThemeWindowsXP
			cfg.Compact = true
		}),
		"no-banner": testConfig(func(cfg *core.Config) {
			cfg.Theme = core.ThemeWindowsXP
			cfg.NoBanner = true
		}),
		"narrow": testConfig(func(cfg *core.Config) {
			cfg.Theme = core.ThemeWindowsXP
		}),
	} {
		width := 70
		if name == "narrow" {
			width = 42
		}
		title := banner.TitleBlock(width, status, time.Unix(0, 0), cfg)

		if !strings.Contains(ansi.StripANSI(title), "REMOTE MONITOR") {
			t.Fatalf("%s fallback title lost product name: %q", name, ansi.StripANSI(title))
		}
		assertRenderedLinesWidth(t, title, width)
	}
}

func TestAuroraBackdropUsesHalfBlockCellsAndMovesOverTime(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	cfg := testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeAurora })
	first := banner.AuroraBackdropLine(strings.Repeat(" ", 36), 36, 12, time.Unix(0, 0), cfg)
	second := banner.AuroraBackdropLine(strings.Repeat(" ", 36), 36, 12, time.Unix(0, 220*int64(time.Millisecond)), cfg)

	if first == second {
		t.Fatal("expected aurora backdrop to move over time")
	}
	if got := ansi.StripANSI(first); got != strings.Repeat("▀", 36) {
		t.Fatalf("expected aurora backdrop to use half-block cells, got %q", got)
	}
	if got := ansi.VisibleLen(first); got != 36 {
		t.Fatalf("expected aurora backdrop width to stay 36, got %d in %q", got, first)
	}
	if fgEscapes := strings.Count(first, "\x1b[38;2;"); fgEscapes < 36 {
		t.Fatalf("expected one truecolor foreground escape per half-block cell, got %d in %q", fgEscapes, first)
	}
	if bgEscapes := strings.Count(first, "\x1b[48;2;"); bgEscapes < 36 {
		t.Fatalf("expected one truecolor background escape per half-block cell, got %d in %q", bgEscapes, first)
	}
}

func TestAuroraBackdropSamplesStayVariedAcrossWarmAndCoolBands(t *testing.T) {
	t.Parallel()

	var hasGreen bool
	var hasPurple bool
	var hasWarm bool
	unique := map[banner.Color]struct{}{}

	for _, sampleTime := range []time.Time{
		time.Unix(0, 0),
		time.Unix(0, 860*int64(time.Millisecond)),
	} {
		for row := 0; row < 48; row += 4 {
			for col := 0; col < 176; col += banner.AuroraBackdropBandWidth {
				color := banner.AuroraBackdropBandColor(col, row, sampleTime)
				unique[color] = struct{}{}

				if color.GreenValue >= color.RedValue+16 && color.GreenValue >= color.BlueValue+8 && color.GreenValue >= 72 {
					hasGreen = true
				}
				if color.RedValue >= 90 && color.BlueValue >= 110 && color.BlueValue >= color.GreenValue+12 {
					hasPurple = true
				}
				if color.RedValue >= color.GreenValue+18 && color.RedValue >= 92 {
					hasWarm = true
				}
			}
		}
	}

	if !hasGreen {
		t.Fatal("expected aurora field samples to include Green ribbons")
	}
	if !hasPurple {
		t.Fatal("expected aurora field samples to include purple ribbons")
	}
	if !hasWarm {
		t.Fatal("expected aurora field samples to include warm Red-pink ribbons")
	}
	if len(unique) < 28 {
		t.Fatalf("expected aurora field to stay visually varied, got %d distinct band colors", len(unique))
	}
}

func TestAuroraBackdropDoesNotRenderInsideBoxInterior(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	cfg := testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeAurora })
	got := banner.AuroraBackdropLine("│      │", 8, 6, time.Unix(0, 0), cfg)

	if strings.Contains(got, "\x1b[48;2;") {
		t.Fatalf("expected boxed interior to avoid aurora backdrop fills, got %q", got)
	}
	if !strings.Contains(got, ansi.PanelBg) {
		t.Fatalf("expected boxed interior to use panel background fill, got %q", got)
	}
}

func TestAuroraBackdropKeepsGapBetweenSeparateTablesAnimated(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	cfg := testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeAurora })
	got := banner.AuroraBackdropLine("│      │  │      │", 18, 6, time.Unix(0, 0), cfg)

	if !strings.Contains(got, ansi.PanelBg) {
		t.Fatalf("expected table interiors to keep panel background fill, got %q", got)
	}
	if !strings.Contains(got, "\x1b[48;2;") {
		t.Fatalf("expected gap between separate tables to keep aurora animation, got %q", got)
	}
}

func TestApplyAuroraBackdropKeepsBannerTransitionAligned(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	cfg := testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeAurora })
	now := time.Unix(0, 0)
	width := 18

	lines := make([]string, 0, len(banner.AuroraBannerCanvas())+1)
	for range banner.AuroraBannerCanvas() {
		lines = append(lines, strings.Repeat("X", width))
	}
	lines = append(lines, strings.Repeat(" ", width))

	gotLines := strings.Split(banner.ApplyAuroraBackdrop(strings.Join(lines, "\n"), width, now, cfg), "\n")
	got := gotLines[len(banner.AuroraBannerCanvas())]
	want := banner.AuroraBackdropLine(strings.Repeat(" ", width), width, len(banner.AuroraBannerCanvas()), now, cfg)

	if got != want {
		t.Fatalf("expected first non-banner row to continue the aurora field without a row offset")
	}
}

func TestAnimatedTitleFallsBackTo256ColorWithoutTrueColorSupport(t *testing.T) {
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")
	status := ansi.Colorize(ansi.Green, "● LIVE")
	title := banner.TitleBlock(170, status, time.Unix(0, 0), testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeAurora }))

	if !strings.Contains(title, "\x1b[38;5;") {
		t.Fatal("expected fallback banner to use 256-color escape")
	}
	if strings.Contains(title, "\x1b[38;2;") {
		t.Fatal("expected fallback banner to avoid truecolor escape")
	}
}

func TestAnimatedTitleHonorsNoTrueColorFlag(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-ghostty")
	status := ansi.Colorize(ansi.Green, "● LIVE")
	title := banner.TitleBlock(170, status, time.Unix(0, 0), testConfig(func(cfg *core.Config) { cfg.Theme = core.ThemeAurora; cfg.DisableTrueColor = true }))

	if !strings.Contains(title, "\x1b[38;5;") {
		t.Fatal("expected forced 256-color banner escape")
	}
	if strings.Contains(title, "\x1b[38;2;") {
		t.Fatal("expected no-truecolor flag to suppress truecolor escape")
	}
}

func assertRenderedLinesWidth(t *testing.T, rendered string, width int) {
	t.Helper()

	for idx, line := range strings.Split(rendered, "\n") {
		if got := ansi.VisibleLen(line); got != width {
			t.Fatalf("line %d width = %d, want %d in %q", idx, got, width, line)
		}
	}
}
