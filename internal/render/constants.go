package render

const (
	severityCritical = "critical"
	severityHot      = "hot"
	severityInfo     = "info"
	severityNeutral  = "neutral"
	severityOK       = "ok"
	severityWarn     = "warn"
)

// Label constants name rendered dashboard metrics and columns.
const (
	LabelGPU = "GPU"
	LabelPID = "PID"

	LabelCPUActive    = "CPU Active"
	LabelCPUFreq      = "CPU Freq"
	LabelCPUImbalance = "CPU Imbalance"
	LabelCPUMap       = "CPU Map"
	LabelCPUPSI       = "CPU PSI"
	LabelCPUTemp      = "CPU Temp"
	LabelCPUUser      = "CPU User"
	LabelDiskInflight = "Disk Inflight"
	LabelMemPSI       = "Mem PSI"
	LabelProcess      = "Process"
	LabelRAMAvail     = "RAM Avail"
	LabelRAMCache     = "RAM Cache"
	LabelRAMFree      = "RAM Free"

	TextNA   = "n/a"
	TextNone = "none"
)
