package render

import (
	"fmt"
	"time"
)

func formatMiBPair(used, total int64) string {
	if used < 0 || total <= 0 {
		return TextNA
	}

	return fmt.Sprintf("%d / %d MiB", used, total)
}

func formatMiBValue(v int64) string {
	if v < 0 {
		return TextNA
	}
	if v >= bytesPerKiB {
		return fmt.Sprintf("%.1f GiB", float64(v)/bytesPerKiB)
	}

	return fmt.Sprintf("%d MiB", v)
}

func formatKiBPair(used, total int64) string {
	if used < 0 || total <= 0 {
		return TextNA
	}

	return fmt.Sprintf("%s / %s", FormatKiBValue(used), FormatKiBValue(total))
}

// FormatKiBValue formats a KiB value with binary units.
func FormatKiBValue(v int64) string {
	if v < 0 {
		return TextNA
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(v)
	unit := units[0]
	for i := 1; i < len(units) && value >= bytesPerKiB; i++ {
		value /= bytesPerKiB
		unit = units[i]
	}
	if unit == "KiB" {
		return fmt.Sprintf("%.0f %s", value, unit)
	}

	return fmt.Sprintf("%.1f %s", value, unit)
}

func formatBps(v int64) string {
	if v < 0 {
		return TextNA
	}
	units := []string{"B/s", "KiB/s", "MiB/s", "GiB/s"}
	value := float64(v)
	unit := units[0]
	for i := 1; i < len(units) && value >= bytesPerKiB; i++ {
		value /= bytesPerKiB
		unit = units[i]
	}
	if unit == "B/s" {
		return fmt.Sprintf("%.0f %s", value, unit)
	}

	return fmt.Sprintf("%.1f %s", value, unit)
}

func percentDisplay(v int) string {
	if v < 0 {
		return TextNA
	}

	return fmt.Sprintf("%d%%", v)
}

func tempDisplay(v int) string {
	if v < 0 {
		return TextNA
	}

	return fmt.Sprintf("%dC", v)
}

func formatFloat(v float64) string {
	if v <= 0 {
		return "0.00"
	}

	return fmt.Sprintf("%.2f", v)
}

func formatPowerPair(draw, limit float64) string {
	if draw <= 0 || limit <= 0 {
		return TextNA
	}

	return fmt.Sprintf("%sW / %.2fW", formatFloat(draw), limit)
}

func formatPowerValue(draw float64) string {
	if draw <= 0 {
		return TextNA
	}

	return formatFloat(draw) + "W"
}

func formatClockPair(current, maxValue int) string {
	switch {
	case current >= 0 && maxValue > 0:
		return fmt.Sprintf("%d / %d MHz", current, maxValue)
	case current >= 0:
		return fmt.Sprintf("%d MHz", current)
	case maxValue > 0:
		return fmt.Sprintf("max %d MHz", maxValue)
	default:
		return TextNA
	}
}

func formatClockValue(current int) string {
	if current < 0 {
		return TextNA
	}

	return fmt.Sprintf("%d MHz", current)
}

// FormatMillisValue formats a millisecond value for display.
func FormatMillisValue(v float64) string {
	if v < 0 {
		return TextNA
	}

	return fmt.Sprintf("%.2f ms", v)
}

// FormatQueueDepth formats a disk queue depth multiplier.
func FormatQueueDepth(v float64) string {
	if v < 0 {
		return TextNA
	}

	return fmt.Sprintf("%.2fx", v)
}

func formatUptime(seconds int64) string {
	if seconds <= 0 {
		return TextNA
	}
	hours := seconds / secondsPerHour
	minutes := (seconds % secondsPerHour) / secondsPerMinute
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}

func formatAge(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	seconds := int(d.Round(time.Second) / time.Second)
	if seconds < secondsPerMinute {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / secondsPerMinute
	if minutes < minutesPerHour {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / minutesPerHour

	return fmt.Sprintf("%dh", hours)
}
