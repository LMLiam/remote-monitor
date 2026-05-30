package render

func diskLatencyHistorySeverity(v int) string {
	if v < 0 {
		return severityNeutral
	}
	switch {
	case v >= diskHistoryCritical:
		return severityCritical
	case v >= diskHistoryWarn:
		return severityWarn
	case v >= diskHistoryAcceptable:
		return severityOK
	default:
		return severityInfo
	}
}

func diskAwaitSeverity(v float64) string {
	switch {
	case v < 0:
		return severityNeutral
	case v >= diskAwaitCriticalMS:
		return severityCritical
	case v >= diskAwaitWarnMS:
		return severityWarn
	case v >= diskAwaitAcceptableMS:
		return severityOK
	default:
		return severityInfo
	}
}

func diskQueueSeverity(v float64) string {
	switch {
	case v < 0:
		return severityNeutral
	case v >= diskQueueCriticalDepth:
		return severityCritical
	case v >= diskQueueWarnDepth:
		return severityWarn
	case v >= 1:
		return severityOK
	default:
		return severityInfo
	}
}
