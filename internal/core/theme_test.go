package core_test

import (
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
)

func TestCanonicalThemeNameAcceptsWindowsXPAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "windows-xp", want: core.ThemeWindowsXP},
		{input: "Windows-XP", want: core.ThemeWindowsXP},
		{input: " xp ", want: core.ThemeWindowsXP},
		{input: "winxp", want: core.ThemeWindowsXP},
		{input: "WINXP", want: core.ThemeWindowsXP},
	}

	for _, tt := range tests {
		got := core.CanonicalThemeName(tt.input)
		if got != tt.want {
			t.Fatalf("CanonicalThemeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
