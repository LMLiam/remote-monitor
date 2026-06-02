package banner

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"math"
	"strings"
	"time"
)

// TitleBlock renders the themed dashboard banner and status line.
func TitleBlock(totalWidth int, statusText string, now time.Time, cfg core.Config) string {
	spec := bannerThemeSpec(cfg.Theme)
	bannerWidth := themeBannerWidth(spec)
	if cfg.NoBanner || cfg.Compact || totalWidth < bannerWidth+titleStatusPadding {
		titleText := "REMOTE MONITOR"

		return titleStatusLine(titleText, fallbackStatusLineColor(cfg), statusText, totalWidth)
	}

	lines := renderThemeBannerLines(spec, totalWidth, now, cfg)
	lines = append(lines, titleStatusLine("", fullBannerStatusLineColor(cfg), statusText, totalWidth))

	return strings.Join(lines, "\n")
}

// DecorateFrame applies theme-specific frame decoration.
func DecorateFrame(frame string, width int, now time.Time, cfg core.Config) string {
	switch core.CanonicalThemeName(cfg.Theme) {
	case core.ThemeAurora:
		return ApplyAuroraBackdrop(frame, width, now, cfg)
	case core.ThemeWindowsXP:
		return ApplyWindowsXPFrame(frame, cfg)
	default:
		return frame
	}
}

func themeBannerWidth(spec bannerTheme) int {
	if spec.animation == bannerAnimationAurora {
		return auroraBannerWidth()
	}
	width := 0
	for _, line := range spec.lines {
		if lineWidth := ansi.VisibleLen(line); lineWidth > width {
			width = lineWidth
		}
	}

	return width
}

func renderThemeBannerLines(spec bannerTheme, totalWidth int, now time.Time, cfg core.Config) []string {
	if spec.animation == bannerAnimationAurora {
		return renderAuroraBannerLines(totalWidth, now, cfg)
	}
	lines := make([]string, 0, len(spec.lines))
	phase := currentBannerPhase(spec, now)
	for idx, line := range spec.lines {
		lines = append(lines, renderBasicBannerLine(spec, CenterText(line, totalWidth), idx, phase, cfg))
	}

	return lines
}

func currentBannerPhase(spec bannerTheme, now time.Time) int {
	if len(spec.palette) == 0 {
		return 0
	}
	phaseMillis := spec.phaseMillis
	if phaseMillis <= 0 {
		phaseMillis = basicBannerPhaseMillis
	}
	phase := int(now.UnixMilli()/phaseMillis) % len(spec.palette)
	if phase < 0 {
		phase += len(spec.palette)
	}

	return phase
}

func fallbackStatusLineColor(cfg core.Config) string {
	if core.CanonicalThemeName(cfg.Theme) == core.ThemeWindowsXP {
		return windowsXPInk(cfg)
	}

	return ansi.Ink
}

func fullBannerStatusLineColor(cfg core.Config) string {
	if core.CanonicalThemeName(cfg.Theme) == core.ThemeWindowsXP {
		return windowsXPInk(cfg)
	}

	return ansi.Cyan
}

func renderBasicBannerLine(spec bannerTheme, text string, row, phase int, cfg core.Config) string {
	paletteIndex := bannerColorIndex(spec, row, phase, len(spec.palette))

	return ansi.Colorize(bannerColorEscape(spec.palette[paletteIndex], cfg), text)
}

func renderAuroraBannerLines(totalWidth int, now time.Time, cfg core.Config) []string {
	canvas := AuroraBannerCanvas()
	width := auroraBannerWidth()
	lines := make([]string, 0, len(canvas))
	leftPad := max(0, (totalWidth-width)/bannerRowWaveDivisor)
	rightPad := max(0, totalWidth-leftPad-width)
	for row, cells := range canvas {
		var b strings.Builder
		b.WriteString(auroraBackdropSegment(0, row, leftPad, now, cfg))
		for col, cell := range cells {
			visualCol := leftPad + col
			backdrop := AuroraBackdropBandColor(visualCol, row, now)
			bg := bannerBackgroundEscape(backdrop, cfg)
			if cell.Kind == CellFace {
				face := auroraFaceColor(visualCol, row, now)
				b.WriteString(ansi.StyledText(bannerColorEscape(face, cfg), bg, string(cell.Glyph), ansi.Bold))

				continue
			}
			b.WriteString(ansi.StyledText("", bg, " "))
		}
		b.WriteString(auroraBackdropSegment(leftPad+width, row, rightPad, now, cfg))
		lines = append(lines, b.String())
	}

	return lines
}

func titleStatusLine(leftText, leftColor, statusText string, width int) string {
	if leftText == "" {
		return ansi.RightJustify(colorPlainStatusText(statusText, leftColor), width)
	}

	statusWidth := ansi.VisibleLen(statusText)
	leftWidth := width - statusWidth - 1
	if leftWidth < titleStatusMinLeftWidth {
		return ansi.RightJustify(colorPlainStatusText(statusText, leftColor), width)
	}

	leftPart := fillBlock(leftText, leftWidth, leftColor, ansi.PanelBg, true)

	return ansi.Pad(leftPart, leftWidth) + " " + statusText
}

func colorPlainStatusText(text, color string) string {
	if color == "" || ansi.HasANSI(text) {
		return text
	}

	return ansi.Colorize(color, text)
}

func bannerColorIndex(spec bannerTheme, row, phase, paletteLen int) int {
	if paletteLen < 1 {
		return 0
	}
	base := phase + row/bannerRowWaveDivisor
	if spec.animation == bannerAnimationAurora {
		wave := int(math.Round(auroraWaveAmplitude * math.Sin(float64(phase)/auroraWavePhaseDivisor+float64(row)*auroraWaveRowScale)))
		base = phase*auroraPhaseStride + row*auroraRowColorStride + wave
	}
	base %= paletteLen
	if base < 0 {
		base += paletteLen
	}

	return base
}

// CenterText pads text to the requested visible width.
func CenterText(text string, width int) string {
	diff := width - ansi.VisibleLen(text)
	if diff <= 0 {
		return text
	}
	left := diff / centerTextDivisor
	right := diff - left

	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

func positiveMod(value, mod int) int {
	if mod <= 0 {
		return 0
	}
	result := value % mod
	if result < 0 {
		result += mod
	}

	return result
}
