package banner

import "github.com/lmliam/remote-monitor/internal/render/ansi"

func clampFromZero(v, maxV int) int {
	if v < 0 {
		return 0
	}
	if v > maxV {
		return maxV
	}

	return v
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
