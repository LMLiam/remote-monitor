package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"math"
)

func psiPercent(v float64) int {
	if v < 0 {
		return -1
	}

	return clamp(int(math.Round(v)), percentMin, percentMax)
}

func formatPSIValue(v float64) string {
	if v < 0 {
		return TextNA
	}

	return fmt.Sprintf("%.2f%%", v)
}

func formatOpsPerSec(v int64) string {
	if v < 0 {
		return TextNA
	}
	value := float64(v)
	suffix := "/s"
	switch {
	case value >= decimalMegaScale:
		return fmt.Sprintf("%.1f M%s", value/decimalMegaScale, suffix)
	case value >= decimalKiloScale:
		return fmt.Sprintf("%.1f k%s", value/decimalKiloScale, suffix)
	default:
		return fmt.Sprintf("%d%s", v, suffix)
	}
}

func inodeUsageSeverity(v int, thresholds core.Thresholds) string {
	return diskUsageSeverity(v, thresholds)
}
