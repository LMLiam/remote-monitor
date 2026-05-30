package render

import (
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

func headerLine(left, middle, right string, width int) string {
	sep := "  "
	line := left + sep + middle + sep + right

	return ansi.Pad(line, width)
}

// HeaderPair renders one labeled header chip and value cell.
func HeaderPair(key, value, valueColor string, width int) string {
	if width < headerMinWidth {
		width = headerMinWidth
	}
	labelText := strings.ToUpper(strings.TrimSpace(key))
	labelWidth := min(headerMaxLabelWidth, max(headerMinLabelWidth, width/headerLabelDivisor))
	if labelWidth >= width {
		labelWidth = max(headerNarrowLabelWidth, width/headerLabelDivisor)
	}
	valueWidth := max(width-labelWidth-1, 1)
	labelPart := chipCell(labelText, labelWidth, ansi.HeaderBg)
	valuePart := fillBlock(" "+ansi.FitText(value, max(1, valueWidth-headerValuePadding))+" ", valueWidth, valueColor, ansi.PanelAltBg, false)

	return ansi.Pad(labelPart+" "+valuePart, width)
}

func fillBlock(text string, width int, fg, bg string, makeBold bool) string {
	if width <= 0 {
		return ""
	}
	content := ansi.FitText(text, width)
	if makeBold {
		return ansi.StyledText(fg, bg, content, ansi.Bold)
	}

	return ansi.StyledText(fg, bg, content)
}

func chipCell(text string, width int, bg string) string {
	if width <= 0 {
		return ""
	}
	if width <= chipPaddedMinWidth {
		return fillBlock(text, width, ansi.Ink, bg, true)
	}

	return fillBlock(" "+ansi.FitText(text, width-chipPaddedMinWidth)+" ", width, ansi.Ink, bg, true)
}

func inlineChip(text, bg string) string {
	label := " " + strings.TrimSpace(text) + " "

	return ansi.StyledText(ansi.Ink, bg, label, ansi.Bold)
}

func accentBackground(color string) string {
	switch color {
	case ansi.Cyan:
		return ansi.CyanBg
	case ansi.Blue:
		return ansi.BlueBg
	case ansi.Green:
		return ansi.GreenBg
	case ansi.Yellow:
		return ansi.YellowBg
	case ansi.Red:
		return ansi.RedBg
	case ansi.Amber:
		return ansi.AmberBg
	case ansi.Lav:
		return ansi.LavBg
	default:
		return ansi.MutedBg
	}
}
