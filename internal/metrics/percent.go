package metrics

import "math"

// ClockPercent returns current clock speed as a percent of its maximum.
func ClockPercent(current, maxValue int) int {
	if current < 0 || maxValue <= 0 {
		return -1
	}

	return Clamp(int(math.Round((float64(current)/float64(maxValue))*percentScale)), percentMin, percentMax)
}

// PowerPercent returns draw as a percent of limit.
func PowerPercent(draw, limit float64) int {
	if draw <= 0 || limit <= 0 {
		return 0
	}

	return Clamp(int(math.Round((draw/limit)*percentScale)), percentMin, percentMax)
}

// TemperaturePercent clamps a Celsius temperature for percent-style displays.
func TemperaturePercent(v int) int {
	if v < 0 {
		return 0
	}

	return Clamp(v, percentMin, percentMax)
}
