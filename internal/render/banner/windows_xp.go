package banner

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

const (
	windowsXPTitleRed      = 235
	windowsXPTitleGreen    = 245
	windowsXPTitleBlue     = 255
	windowsXPBorderRed     = 42
	windowsXPBorderGreen   = 96
	windowsXPBorderBlue    = 216
	windowsXPPanelRed      = 14
	windowsXPPanelGreen    = 52
	windowsXPPanelBlue     = 118
	windowsXPAltRed        = 22
	windowsXPAltGreen      = 73
	windowsXPAltBlue       = 148
	windowsXPHeaderRed     = 33
	windowsXPHeaderGreen   = 106
	windowsXPHeaderBlue    = 214
	windowsXPTrackRed      = 6
	windowsXPTrackGreen    = 33
	windowsXPTrackBlue     = 94
	windowsXPFallbackTitle = "\x1b[1;38;5;195m"
	windowsXPFallbackBlue  = "\x1b[38;5;33m"
	windowsXPFallbackPanel = "\x1b[48;5;18m"
	windowsXPFallbackAlt   = "\x1b[48;5;19m"
	windowsXPFallbackHead  = "\x1b[48;5;27m"
	windowsXPFallbackTrack = "\x1b[48;5;17m"
)

// ApplyWindowsXPFrame gives the standard dashboard an XP-inspired chrome without changing metric severity colors.
func ApplyWindowsXPFrame(frame string, cfg core.Config) string {
	replacer := strings.NewReplacer(
		ansi.TitleColor, windowsXPTitle(cfg),
		ansi.BorderColor, windowsXPBorder(cfg),
		ansi.PanelBg, windowsXPPanel(cfg),
		ansi.PanelAltBg, windowsXPPanelAlt(cfg),
		ansi.HeaderBg, windowsXPHeader(cfg),
		ansi.TrackBg, windowsXPTrack(cfg),
	)

	return replacer.Replace(frame)
}

func windowsXPInk(cfg core.Config) string {
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		return ansiTrueColor(windowsXPTitleRed, windowsXPTitleGreen, windowsXPTitleBlue)
	}

	return windowsXPFallbackTitle
}

func windowsXPTitle(cfg core.Config) string {
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		return ansi.Bold + ansiTrueColor(windowsXPTitleRed, windowsXPTitleGreen, windowsXPTitleBlue)
	}

	return windowsXPFallbackTitle
}

func windowsXPBorder(cfg core.Config) string {
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		return ansiTrueColor(windowsXPBorderRed, windowsXPBorderGreen, windowsXPBorderBlue)
	}

	return windowsXPFallbackBlue
}

func windowsXPPanel(cfg core.Config) string {
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		return ansiTrueColorBackground(windowsXPPanelRed, windowsXPPanelGreen, windowsXPPanelBlue)
	}

	return windowsXPFallbackPanel
}

func windowsXPPanelAlt(cfg core.Config) string {
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		return ansiTrueColorBackground(windowsXPAltRed, windowsXPAltGreen, windowsXPAltBlue)
	}

	return windowsXPFallbackAlt
}

func windowsXPHeader(cfg core.Config) string {
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		return ansiTrueColorBackground(windowsXPHeaderRed, windowsXPHeaderGreen, windowsXPHeaderBlue)
	}

	return windowsXPFallbackHead
}

func windowsXPTrack(cfg core.Config) string {
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		return ansiTrueColorBackground(windowsXPTrackRed, windowsXPTrackGreen, windowsXPTrackBlue)
	}

	return windowsXPFallbackTrack
}
