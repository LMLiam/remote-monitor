package render

const (
	cpuRowsCapacity       = 18
	cpuPostMapRowsCap     = 6
	memoryRowsCapacity    = 12
	storageBaseRowsCap    = 10
	gpuRowsPerDevice      = 16
	hotCorePreviewLimit   = 4
	summaryStateChipMin   = 12
	summaryStateChipDiv   = 2
	summaryAlertChipMin   = 10
	summaryAlertChipDiv   = 3
	diskInflightScale     = 12
	filesystemLastWidth   = 9
	filesystemParentWidth = 5
	filesystemLabelMax    = 18
	filesystemFallbackMax = 12
	ppsMillionScale       = decimalMegaScale
	ppsThousandScale      = decimalKiloScale
	coreMultiRowMinWidth  = 16
	coreGroupSize         = 4
	coreMinGridColumns    = 2
	gridSearchScoreCeil   = 1 << 30
	gridShapeScoreWeight  = 10
	centerLineDivisor     = 2
	coreLevelLow          = 5
	coreLevelMild         = 15
	coreLevelMedium       = 30
	coreLevelRaised       = 45
	coreLevelHigh         = 60
	coreLevelVeryHigh     = 80
)

// TableRowSpec describes one metrics row in a three-column table.
type TableRowSpec struct {
	Divider       bool
	LabelText     string
	LabelColor    string
	ValueText     string
	ValueColor    string
	ValueCell     string
	ActivityText  string
	ActivityColor string
	ActivityCell  string
}

// StatusSummaryRowSpec describes one compact status summary row.
type StatusSummaryRowSpec struct {
	LabelText   string
	LabelColor  string
	SummaryCell string
}

// ProcessListRowSpec describes one row in a process list table.
type ProcessListRowSpec struct {
	ProcessText  string
	ProcessColor string
	PIDText      string
	PIDColor     string
	UsageText    string
	UsageColor   string
	MemoryText   string
	MemoryColor  string
}

// TableFullRow builds a populated metric table row.
func TableFullRow(
	labelText string,
	labelColor string,
	valueText string,
	valueColor string,
	valueCell string,
	activityText string,
	activityColor string,
	activityCell string,
) TableRowSpec {
	return TableRowSpec{
		Divider:       false,
		LabelText:     labelText,
		LabelColor:    labelColor,
		ValueText:     valueText,
		ValueColor:    valueColor,
		ValueCell:     valueCell,
		ActivityText:  activityText,
		ActivityColor: activityColor,
		ActivityCell:  activityCell,
	}
}

func tableDividerRow() TableRowSpec {
	return TableRowSpec{
		Divider:       true,
		LabelText:     "",
		LabelColor:    "",
		ValueText:     "",
		ValueColor:    "",
		ValueCell:     "",
		ActivityText:  "",
		ActivityColor: "",
		ActivityCell:  "",
	}
}

func statusSummaryRow(labelText, labelColor, summaryCell string) StatusSummaryRowSpec {
	return StatusSummaryRowSpec{
		LabelText:   labelText,
		LabelColor:  labelColor,
		SummaryCell: summaryCell,
	}
}

func processRow(
	processText string,
	processColor string,
	pidText string,
	pidColor string,
	usageText string,
	usageColor string,
	memoryText string,
	memoryColor string,
) ProcessListRowSpec {
	return ProcessListRowSpec{
		ProcessText:  processText,
		ProcessColor: processColor,
		PIDText:      pidText,
		PIDColor:     pidColor,
		UsageText:    usageText,
		UsageColor:   usageColor,
		MemoryText:   memoryText,
		MemoryColor:  memoryColor,
	}
}
