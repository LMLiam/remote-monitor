package banner

import "github.com/lmliam/remote-monitor/internal/render/ansi"

func buildBannerCanvas(lines []string) [][]Cell {
	width := 0
	for _, line := range lines {
		if lineWidth := ansi.VisibleLen(line); lineWidth > width {
			width = lineWidth
		}
	}
	height := len(lines)
	canvas := make([][]Cell, height)
	for row := range canvas {
		canvas[row] = make([]Cell, width)
		for col := range canvas[row] {
			canvas[row][col] = Cell{Glyph: ' ', Kind: CellEmpty}
		}
	}
	for row, line := range lines {
		col := 0
		for _, r := range line {
			if r != ' ' {
				canvas[row][col] = Cell{Glyph: r, Kind: CellFace}
			}
			col++
		}
	}

	return canvas
}

func bannerCanvasWidth(canvas [][]Cell) int {
	width := 0
	for _, row := range canvas {
		if len(row) > width {
			width = len(row)
		}
	}

	return width
}
