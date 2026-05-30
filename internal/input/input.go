package input

import (
	"bufio"
)

const (
	asciiCtrlC           = 3
	asciiEscape          = 0x1b
	escapeSequenceBuffer = 8
)

// TTYCommand describes one keyboard navigation action from the terminal.
type TTYCommand struct {
	LineDelta int
	PageDelta int
	ToTop     bool
	ToBottom  bool
	Quit      bool
}

func newTTYCommand(lineDelta, pageDelta int, toTop, toBottom, quit bool) TTYCommand {
	return TTYCommand{
		LineDelta: lineDelta,
		PageDelta: pageDelta,
		ToTop:     toTop,
		ToBottom:  toBottom,
		Quit:      quit,
	}
}

// ReadTTYCommand reads and decodes one supported terminal input sequence.
func ReadTTYCommand(reader *bufio.Reader) (TTYCommand, bool, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return newTTYCommand(0, 0, false, false, false), false, err
	}

	switch b {
	case asciiCtrlC, 'q', 'Q':
		return newTTYCommand(0, 0, false, false, true), true, nil
	case 'j', 'J':
		return newTTYCommand(1, 0, false, false, false), true, nil
	case 'k', 'K':
		return newTTYCommand(-1, 0, false, false, false), true, nil
	case 'g':
		return newTTYCommand(0, 0, true, false, false), true, nil
	case 'G':
		return newTTYCommand(0, 0, false, true, false), true, nil
	case asciiEscape:
		return readTTYEscapeCommand(reader)
	default:
		return newTTYCommand(0, 0, false, false, false), false, nil
	}
}

func readTTYEscapeCommand(reader *bufio.Reader) (TTYCommand, bool, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return newTTYCommand(0, 0, false, false, false), false, err
	}

	switch b {
	case '[':
		return readTTYCSICommand(reader)
	case 'O':
		final, err := reader.ReadByte()
		if err != nil {
			return newTTYCommand(0, 0, false, false, false), false, err
		}
		switch final {
		case 'H':
			return newTTYCommand(0, 0, true, false, false), true, nil
		case 'F':
			return newTTYCommand(0, 0, false, true, false), true, nil
		default:
			return newTTYCommand(0, 0, false, false, false), false, nil
		}
	default:
		return newTTYCommand(0, 0, false, false, false), false, nil
	}
}

func readTTYCSICommand(reader *bufio.Reader) (TTYCommand, bool, error) {
	sequence := make([]byte, 0, escapeSequenceBuffer)
	for len(sequence) < escapeSequenceBuffer {
		b, err := reader.ReadByte()
		if err != nil {
			return newTTYCommand(0, 0, false, false, false), false, err
		}
		sequence = append(sequence, b)
		if b >= '@' && b <= '~' {
			break
		}
	}

	switch string(sequence) {
	case "A":
		return newTTYCommand(-1, 0, false, false, false), true, nil
	case "B":
		return newTTYCommand(1, 0, false, false, false), true, nil
	case "5~":
		return newTTYCommand(0, -1, false, false, false), true, nil
	case "6~":
		return newTTYCommand(0, 1, false, false, false), true, nil
	case "H", "1~", "7~":
		return newTTYCommand(0, 0, true, false, false), true, nil
	case "F", "4~", "8~":
		return newTTYCommand(0, 0, false, true, false), true, nil
	default:
		return newTTYCommand(0, 0, false, false, false), false, nil
	}
}
