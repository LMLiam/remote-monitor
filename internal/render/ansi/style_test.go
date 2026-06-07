package ansi_test

import (
	"testing"

	"github.com/lmliam/remote-monitor/internal/render/ansi"
)

const reset = "\x1b[0m"

func TestStripAndDetectANSI(t *testing.T) {
	t.Parallel()

	input := ansi.Red + "hot" + reset + " " + ansi.Green + "ok" + reset

	if got := ansi.StripANSI(input); got != "hot ok" {
		t.Fatalf("StripANSI() = %q, want %q", got, "hot ok")
	}
	if !ansi.HasANSI(input) {
		t.Fatal("HasANSI() = false, want true for styled text")
	}
	if ansi.HasANSI("plain text") {
		t.Fatal("HasANSI() = true, want false for plain text")
	}
}

func TestLeadingEscape(t *testing.T) {
	t.Parallel()

	got, ok := ansi.LeadingEscape(ansi.Bold + "status")
	if !ok {
		t.Fatal("LeadingEscape() did not find leading escape")
	}
	if got != ansi.Bold {
		t.Fatalf("LeadingEscape() = %q, want %q", got, ansi.Bold)
	}

	if got, ok := ansi.LeadingEscape("ready " + ansi.Green); ok || got != "" {
		t.Fatalf("LeadingEscape() = %q, %v for non-leading escape; want empty false", got, ok)
	}
}

func TestColorize(t *testing.T) {
	t.Parallel()

	if got := ansi.Colorize("", "plain"); got != "plain" {
		t.Fatalf("Colorize() without color = %q, want plain", got)
	}

	want := ansi.Cyan + "text" + reset
	if got := ansi.Colorize(ansi.Cyan, "text"); got != want {
		t.Fatalf("Colorize() = %q, want %q", got, want)
	}
}

func TestStyledText(t *testing.T) {
	t.Parallel()

	if got := ansi.StyledText("", "", "plain"); got != "plain" {
		t.Fatalf("StyledText() without style = %q, want plain", got)
	}

	want := ansi.Bold + ansi.Cyan + ansi.PanelBg + "label" + reset
	if got := ansi.StyledText(ansi.Cyan, ansi.PanelBg, "label", ansi.Bold); got != want {
		t.Fatalf("StyledText() = %q, want %q", got, want)
	}
}
