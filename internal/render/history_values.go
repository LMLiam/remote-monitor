package render

func lastOrZero(values []int) int {
	if len(values) == 0 {
		return 0
	}

	return values[len(values)-1]
}

func lastOrZero64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}

	return values[len(values)-1]
}
