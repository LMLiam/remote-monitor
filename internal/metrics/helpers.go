package metrics

import "math"

const (
	percentMin   = 0
	percentMax   = 100
	percentScale = 100
)

// RatePercent returns value as a clamped percent of ceiling.
func RatePercent(value, ceiling int64) int {
	if value <= 0 || ceiling <= 0 {
		return 0
	}
	pct := max(int(math.Ceil((float64(value)/float64(ceiling))*percentScale)), 1)

	return Clamp(pct, percentMin, percentMax)
}

// PercentOf returns used as a clamped percentage of total.
func PercentOf(used, total int64) int {
	if total <= 0 || used < 0 {
		return 0
	}

	return Clamp(int(math.Round((float64(used)/float64(total))*percentScale)), percentMin, percentMax)
}

// Clamp restricts v to the inclusive range minV..maxV.
func Clamp(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}

	return v
}

// Max64 returns the larger int64 value.
func Max64(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}
