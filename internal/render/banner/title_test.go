//nolint:testpackage // These tests intentionally cover unexported banner helper behavior.
package banner

import (
	"strings"
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
)

func TestStatusLineColorsPreserveExistingThemeDefaults(t *testing.T) {
	t.Parallel()

	var basic core.Config
	basic.Theme = core.ThemeBasic
	if got := fallbackStatusLineColor(basic); got != ansi.Ink {
		t.Fatalf("basic fallback status color = %q, want %q", got, ansi.Ink)
	}
	if got := fullBannerStatusLineColor(basic); got != ansi.Cyan {
		t.Fatalf("basic full-banner status color = %q, want %q", got, ansi.Cyan)
	}

	var xp core.Config
	xp.Theme = core.ThemeWindowsXP
	xp.DisableTrueColor = true
	if got := fallbackStatusLineColor(xp); got != windowsXPFallbackTitle {
		t.Fatalf("windows-xp fallback status color = %q, want %q", got, windowsXPFallbackTitle)
	}
	if got := fullBannerStatusLineColor(xp); got != windowsXPFallbackTitle {
		t.Fatalf("windows-xp full-banner status color = %q, want %q", got, windowsXPFallbackTitle)
	}
}

func TestTitleStatusLineColorsPlainRightJustifiedStatus(t *testing.T) {
	t.Parallel()

	got := titleStatusLine("", ansi.Cyan, "READY", 12)
	if !strings.Contains(got, ansi.Cyan) {
		t.Fatalf("plain right-justified status line did not use requested color: %q", got)
	}
	if ansi.VisibleLen(got) != 12 {
		t.Fatalf("plain right-justified status line width = %d, want 12 in %q", ansi.VisibleLen(got), got)
	}

	styledStatus := ansi.Colorize(ansi.Green, "READY")
	got = titleStatusLine("", ansi.Cyan, styledStatus, 12)
	if strings.Contains(got, ansi.Cyan) {
		t.Fatalf("pre-styled status line should keep its own color without an outer cyan wrapper: %q", got)
	}
	if !strings.Contains(got, ansi.Green) {
		t.Fatalf("pre-styled status line lost its original color: %q", got)
	}
}

func TestWindowsXPBannerLinesUseDistinctVisibleArt(t *testing.T) {
	t.Parallel()

	xp := strings.Join(trimBannerArtLines(WindowsXPBannerLines()), "\n")
	basic := strings.Join(trimBannerArtLines(BasicBannerLines()), "\n")

	if xp == basic {
		t.Fatal("windows-xp banner visible art matches basic banner art")
	}
}

func trimBannerArtLines(lines []string) []string {
	trimmed := make([]string, len(lines))
	for idx, line := range lines {
		trimmed[idx] = strings.TrimRight(line, " ")
	}

	return trimmed
}
