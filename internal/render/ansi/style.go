package ansi

import (
	"os"
	"regexp"
	"strings"
)

// Bold and Dim are reusable ANSI text attributes.
const (
	reset = "\x1b[0m"
	Bold  = "\x1b[1m"
	Dim   = "\x1b[2m"
)

// TitleColor and related constants define the dashboard ANSI color palette.
const (
	TitleColor  = "\x1b[1;38;5;159m"
	BorderColor = "\x1b[38;5;117m"
	Ink         = "\x1b[38;5;255m"
	Cyan        = "\x1b[38;5;159m"
	Blue        = "\x1b[38;5;117m"
	Green       = "\x1b[38;5;121m"
	Yellow      = "\x1b[38;5;186m"
	Red         = "\x1b[38;5;211m"
	Amber       = "\x1b[38;5;219m"
	Sand        = "\x1b[38;5;225m"
	Lav         = "\x1b[38;5;183m"
	Muted       = "\x1b[38;5;246m"
	PanelBg     = "\x1b[48;5;233m"
	PanelAltBg  = "\x1b[48;5;235m"
	HeaderBg    = "\x1b[48;5;23m"
	TrackBg     = "\x1b[48;5;234m"
	CyanBg      = "\x1b[48;5;37m"
	BlueBg      = "\x1b[48;5;25m"
	GreenBg     = "\x1b[48;5;29m"
	YellowBg    = "\x1b[48;5;101m"
	RedBg       = "\x1b[48;5;89m"
	AmberBg     = "\x1b[48;5;97m"
	LavBg       = "\x1b[48;5;60m"
	MutedBg     = "\x1b[48;5;238m"
)

var ansiRE = regexp.MustCompile(ansiPattern())

func ansiPattern() string {
	intermediateBytes := regexp.QuoteMeta(" !\"#$%&'()*+,./-")
	finalBytes := regexp.QuoteMeta("@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~")

	return "\x1b\\[[0-9;?]*[" + intermediateBytes + "]*(?:[" + finalBytes + "])"
}

// StripANSI removes ANSI escape sequences from rendered text.
func StripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

// HasANSI reports whether s contains an ANSI escape sequence.
func HasANSI(s string) bool {
	return ansiRE.MatchString(s)
}

// LeadingEscape returns the leading ANSI escape sequence from s.
func LeadingEscape(s string) (string, bool) {
	match := ansiRE.FindStringIndex(s)
	if len(match) != 2 || match[0] != 0 {
		return "", false
	}

	return s[:match[1]], true
}

// Colorize wraps text in an ANSI color escape when color is non-empty.
func Colorize(color, text string) string {
	if color == "" {
		return text
	}

	return color + text + reset
}

// SupportsTrueColor reports whether the current terminal appears to support 24-bit color.
func SupportsTrueColor(disabled bool) bool {
	if disabled {
		return false
	}
	colorTerm := strings.ToLower(strings.TrimSpace(os.Getenv("COLORTERM")))
	if strings.Contains(colorTerm, "truecolor") || strings.Contains(colorTerm, "24bit") {
		return true
	}
	termName := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))

	return strings.Contains(termName, "direct") || strings.Contains(termName, "truecolor")
}

// StyledText applies foreground, background, and attribute escapes to text.
func StyledText(fg, bg, text string, attrs ...string) string {
	if fg == "" && bg == "" && len(attrs) == 0 {
		return text
	}
	var b strings.Builder
	for _, attr := range attrs {
		if attr != "" {
			b.WriteString(attr)
		}
	}
	if fg != "" {
		b.WriteString(fg)
	}
	if bg != "" {
		b.WriteString(bg)
	}
	b.WriteString(text)
	b.WriteString(reset)

	return b.String()
}
