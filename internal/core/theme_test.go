package core_test

import (
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
)

func TestCanonicalThemeNameAcceptsWindowsXPAliases(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"windows-xp": core.ThemeWindowsXP,
		"Windows-XP": core.ThemeWindowsXP,
		" xp ":       core.ThemeWindowsXP,
		"winxp":      core.ThemeWindowsXP,
		"WINXP":      core.ThemeWindowsXP,
	}

	for input, want := range tests {
		got := core.CanonicalThemeName(input)
		if got != want {
			t.Fatalf("CanonicalThemeName(%q) = %q, want %q", input, got, want)
		}
	}
}
