package core

import "strings"

// CanonicalThemeName normalizes configured theme names to supported themes.
func CanonicalThemeName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "default", ThemeAurora:
		return ThemeAurora
	case ThemeBasic:
		return ThemeBasic
	case ThemeWindowsXP, "xp", "winxp":
		return ThemeWindowsXP
	default:
		return ThemeAurora
	}
}
