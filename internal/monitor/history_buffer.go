package monitor

func appendHistory(values *[]int, v, limit int) {
	*values = append(*values, v)
	if len(*values) > limit {
		*values = append([]int(nil), (*values)[len(*values)-limit:]...)
	}
}

func appendHistory64(values *[]int64, v int64, limit int) {
	*values = append(*values, v)
	if len(*values) > limit {
		*values = append([]int64(nil), (*values)[len(*values)-limit:]...)
	}
}
