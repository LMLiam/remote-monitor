package render

import (
	"fmt"
	"math"
	"strings"
)

func pcieLinkPercent(genCurrent, genMax, widthCurrent, widthMax int) int {
	if genCurrent < 0 || genMax <= 0 || widthCurrent < 0 || widthMax <= 0 {
		return -1
	}
	current := genCurrent * widthCurrent
	maxValue := genMax * widthMax
	if maxValue <= 0 {
		return -1
	}

	return clamp(int(math.Round((float64(current)/float64(maxValue))*percentScale)), percentMin, percentMax)
}

func formatPCIeLinkCurrent(genCurrent, widthCurrent int) string {
	switch {
	case genCurrent <= 0 && widthCurrent <= 0:
		return TextNA
	case widthCurrent <= 0:
		return fmt.Sprintf("Gen%d", genCurrent)
	case genCurrent <= 0:
		return fmt.Sprintf("x%d", widthCurrent)
	default:
		return fmt.Sprintf("Gen%d x%d", genCurrent, widthCurrent)
	}
}

func formatPCIeLinkMax(genMax, widthMax int) string {
	switch {
	case genMax <= 0 && widthMax <= 0:
		return "max n/a"
	case widthMax <= 0:
		return fmt.Sprintf("max Gen%d", genMax)
	case genMax <= 0:
		return fmt.Sprintf("max x%d", widthMax)
	default:
		return fmt.Sprintf("max Gen%d x%d", genMax, widthMax)
	}
}

func throttleSeverity(reasons string) string {
	reasons = strings.ToLower(strings.TrimSpace(reasons))
	switch {
	case reasons == "" || reasons == TextNone:
		return "ok"
	case strings.Contains(reasons, "thermal") || strings.Contains(reasons, "hw slow"):
		return severityCritical
	case strings.Contains(reasons, "power cap") || strings.Contains(reasons, "sync boost") || strings.Contains(reasons, "app clocks") || strings.Contains(reasons, "display"):
		return severityWarn
	case strings.Contains(reasons, "idle"):
		return severityInfo
	default:
		return severityWarn
	}
}
