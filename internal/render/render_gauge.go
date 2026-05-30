package render

import (
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"math"
	"strings"
)

func gaugeCell(percent, width int, severity string) string {
	return gaugeBarCell(percent, width, SeverityColor(severity), percentDisplay(percent))
}

func rateGaugeSummaryCell(value, ceiling int64, width int, color, suffix string) string {
	if value < 0 {
		return fillBlock(TextNA, width, ansi.Muted, ansi.PanelAltBg, false)
	}

	return gaugeBarCell(metrics.RatePercent(value, ceiling), width, color, suffix)
}

func gaugeBarCell(percent, width int, fillColor, suffix string) string {
	plain := gaugeBar(percent, width, fillColor, suffix)

	return ansi.Pad(plain, width)
}

// Gauge renders a fixed-width utilization gauge using a severity palette.
func Gauge(percent, width int, severity string) string {
	return gaugeBar(percent, width, SeverityColor(severity), percentDisplay(percent))
}

func gaugeBar(percent, width int, fillColor, suffix string) string {
	if percent < 0 {
		return fillBlock(TextNA, width, ansi.Muted, ansi.PanelAltBg, false)
	}
	if width < minGaugeWidth {
		if suffix != "" {
			return fillBlock(suffix, width, fillColor, ansi.PanelAltBg, false)
		}

		return fillBlock("", width, fillColor, ansi.PanelAltBg, false)
	}

	suffixText := strings.TrimSpace(suffix)
	suffixReserve := 0
	if suffixText != "" {
		suffixReserve = max(gaugeSuffixMinReserve, ansi.VisibleLen(suffixText))
	}

	bodyWidth := width
	if suffixText != "" {
		bodyWidth -= suffixReserve + 1
	}
	if bodyWidth < minGaugeBodyWidth {
		return fillBlock(suffixText, width, fillColor, ansi.PanelAltBg, false)
	}
	halfUnits := int(math.Round(float64(clamp(percent, percentMin, percentMax)) * float64(bodyWidth*gaugeHalfUnitMultiplier) / percentScale))
	if percent > 0 && halfUnits == 0 {
		halfUnits = 1
	}
	lowSignal := percent > 0 && halfUnits <= 1

	var b strings.Builder
	for range bodyWidth {
		switch {
		case halfUnits >= gaugeHalfUnitMultiplier:
			b.WriteString(ansi.StyledText("", accentBackground(fillColor), " "))
			halfUnits -= gaugeHalfUnitMultiplier
		case halfUnits == 1:
			bg := ansi.TrackBg
			if lowSignal {
				bg = accentBackground(fillColor)
			}
			b.WriteString(ansi.StyledText(fillColor, bg, "▌", ansi.Bold))
			halfUnits = 0
		default:
			b.WriteString(ansi.StyledText("", ansi.TrackBg, " "))
		}
	}
	if suffixText != "" {
		b.WriteString(" ")
		b.WriteString(fillBlock(ansi.RightJustify(suffixText, suffixReserve), suffixReserve, fillColor, ansi.PanelAltBg, false))
	}

	return b.String()
}
