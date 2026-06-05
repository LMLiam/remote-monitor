package ansi_test

import (
	"testing"

	"github.com/lmliam/remote-monitor/internal/render/ansi"
)

func TestVisibleLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "plain ascii", text: "remote", want: 6},
		{name: "sgr escapes ignored", text: ansi.Green + "ok" + reset, want: 2},
		{name: "wide cjk", text: "界面", want: 4},
		{name: "combining mark", text: "e\u0301", want: 1},
		{name: "zero width joiner sequence", text: "👨\u200d💻", want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ansi.VisibleLen(tt.text); got != tt.want {
				t.Fatalf("VisibleLen(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestPadAndRightJustifyUseVisibleWidth(t *testing.T) {
	t.Parallel()

	if got := ansi.Pad(ansi.Green+"ok"+reset, 4); got != ansi.Green+"ok"+reset+"  " {
		t.Fatalf("Pad() = %q, want styled text plus two spaces", got)
	}

	if got := ansi.RightJustify("界", 4); got != "  界" {
		t.Fatalf("RightJustify() = %q, want two leading spaces before wide rune", got)
	}
}

func TestFitAndTruncateText(t *testing.T) {
	t.Parallel()

	if got := ansi.FitText("abcdef", 4); got != "abc…" {
		t.Fatalf("FitText() = %q, want %q", got, "abc…")
	}

	if got := ansi.FitText("ok", 4); got != "ok  " {
		t.Fatalf("FitText() = %q, want right-padded text", got)
	}

	if got := ansi.TruncateText("abcdef", 4); got != "abc…" {
		t.Fatalf("TruncateText() = %q, want %q", got, "abc…")
	}

	if got := ansi.TruncateText("abcdef", 0); got != "" {
		t.Fatalf("TruncateText() with zero width = %q, want empty", got)
	}
}
