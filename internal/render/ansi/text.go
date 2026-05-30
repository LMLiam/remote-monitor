package ansi

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Pad right-pads s to the requested visible width.
func Pad(s string, width int) string {
	diff := width - VisibleLen(s)
	if diff <= 0 {
		return s
	}

	return s + strings.Repeat(" ", diff)
}

// RightJustify left-pads s to the requested visible width.
func RightJustify(s string, width int) string {
	diff := width - VisibleLen(s)
	if diff <= 0 {
		return s
	}

	return strings.Repeat(" ", diff) + s
}

// FitText truncates and right-pads text to a fixed visible width.
func FitText(s string, width int) string {
	if width <= 0 {
		return ""
	}
	truncated := runewidth.Truncate(s, width, "…")

	return runewidth.FillRight(truncated, width)
}

// TruncateText truncates text to a fixed visible width.
func TruncateText(s string, width int) string {
	if width <= 0 {
		return ""
	}

	return runewidth.Truncate(s, width, "…")
}

// VisibleLen returns display width after removing ANSI escapes.
func VisibleLen(s string) int {
	clean := ansiRE.ReplaceAllString(s, "")

	return runewidth.StringWidth(clean)
}
