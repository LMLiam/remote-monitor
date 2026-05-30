package render

import (
	"os"

	"golang.org/x/term"
)

const (
	cursorHome = "\x1b[H"
	clearLine  = "\x1b[2K"
	clearBelow = "\x1b[J"
)

// StdoutIsTTY reports whether stdout is an interactive terminal.
func StdoutIsTTY() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	return term.IsTerminal(int(os.Stdout.Fd()))
}
