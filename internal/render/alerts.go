package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"strings"
)

const (
	alertSummaryLimit  = 3
	alertIssueCapacity = 8
)

type alertIssue struct {
	severity string
	text     string
}

// AlertSummary returns the highest alert severity and a compact human summary.
func AlertSummary(state core.AppState) (severity, summary string) {
	if !state.HasSample {
		return severityWarn, "waiting for first Sample"
	}

	issues := alertIssues(state)
	if len(issues) == 0 {
		return severityOK, "nominal"
	}

	severity = alertSeverity(issues)
	texts := alertIssueTexts(issues)
	if len(texts) > alertSummaryLimit {
		return severity, strings.Join(texts[:alertSummaryLimit], " • ") + fmt.Sprintf(" • +%d more", len(texts)-alertSummaryLimit)
	}

	return severity, strings.Join(texts, " • ")
}

func alertIssues(state core.AppState) []alertIssue {
	s := state.Current
	thresholds := thresholdsOrDefaults(state.Cfg.Thresholds)
	issues := make([]alertIssue, 0, alertIssueCapacity)
	switch currentStatus(state) {
	case core.StatusDisconnected:
		issues = appendAlertIssue(issues, severityCritical, core.StatusDisconnected)
	case core.StatusStale:
		issues = appendAlertIssue(issues, severityWarn, core.StatusStale)
	}

	issues = appendCPUAlertIssues(issues, s, thresholds)
	issues = appendMemoryAlertIssues(issues, s, thresholds)
	issues = appendGPUAlertIssues(issues, s, thresholds)
	issues = appendDiskAlertIssues(issues, s, thresholds)

	return appendNetworkAlertIssues(issues, s)
}

func appendAlertIssue(issues []alertIssue, severity, text string) []alertIssue {
	return append(issues, alertIssue{
		severity: severity,
		text:     text,
	})
}

func appendCPUAlertIssues(issues []alertIssue, s core.Sample, thresholds core.Thresholds) []alertIssue {
	if s.CPUPercent >= thresholds.CPUCriticalPercent { // default 95
		issues = appendAlertIssue(issues, severityCritical, "cpu saturated")
	}
	if s.CPUTempC >= thresholds.CPUCriticalTemp { // default 85 C
		issues = appendAlertIssue(issues, severityCritical, "cpu hot")
	} else if s.CPUTempC >= thresholds.CPUWarnTemp { // default 75 C
		issues = appendAlertIssue(issues, severityWarn, "cpu warm")
	}

	return issues
}

func appendMemoryAlertIssues(issues []alertIssue, s core.Sample, thresholds core.Thresholds) []alertIssue {
	if ramAvailPct := metrics.RAMAvailablePercent(s); ramAvailPct >= 0 {
		switch {
		case ramAvailPct <= thresholds.RAMCriticalAvailablePercent: // default 5
			issues = appendAlertIssue(issues, severityCritical, "ram low")
		case ramAvailPct <= thresholds.RAMWarnAvailablePercent: // default 15
			issues = appendAlertIssue(issues, severityWarn, "ram tight")
		}
	}

	return issues
}

func appendGPUAlertIssues(issues []alertIssue, s core.Sample, thresholds core.Thresholds) []alertIssue {
	if metrics.OverallTempValue(s) >= thresholds.GPUCriticalTemp { // default 80 C
		issues = appendAlertIssue(issues, severityCritical, "gpu hot")
	} else if metrics.OverallTempValue(s) >= thresholds.GPUWarnTemp { // default 70 C
		issues = appendAlertIssue(issues, severityWarn, "gpu warm")
	}
	if vramPct := metrics.OverallVRAMPct(s); vramPct >= thresholds.VRAMCriticalPercent { // default 95
		issues = appendAlertIssue(issues, severityCritical, "vram high")
	} else if vramPct >= thresholds.VRAMWarnPercent { // default 85
		issues = appendAlertIssue(issues, severityWarn, "vram high")
	}

	return issues
}

func appendDiskAlertIssues(issues []alertIssue, s core.Sample, thresholds core.Thresholds) []alertIssue {
	switch {
	case s.RootUsedPercent >= thresholds.DiskCriticalPercent: // default 95
		issues = appendAlertIssue(issues, severityCritical, "disk full")
	case s.RootUsedPercent >= thresholds.DiskWarnPercent: // default 90
		issues = appendAlertIssue(issues, severityWarn, "disk high")
	}
	switch mergeSeverity(diskAwaitSeverity(s.DiskAwaitMS), diskQueueSeverity(s.DiskQueueDepth)) {
	case severityCritical:
		issues = appendAlertIssue(issues, severityCritical, "disk latency")
	case severityWarn:
		issues = appendAlertIssue(issues, severityWarn, "disk latency")
	}

	return issues
}

func appendNetworkAlertIssues(issues []alertIssue, s core.Sample) []alertIssue {
	drops, errors := metrics.NetIssueTotals(s)
	if errors > 0 || drops > 0 {
		issues = appendAlertIssue(issues, severityWarn, "net "+NetDirectionHealthSummary(drops, errors))
	}

	return issues
}

func alertSeverity(issues []alertIssue) string {
	severity := severityOK
	for _, issue := range issues {
		severity = mergeSeverity(severity, issue.severity)
	}

	return severity
}

func alertIssueTexts(issues []alertIssue) []string {
	texts := make([]string, 0, len(issues))
	for _, issue := range issues {
		texts = append(texts, issue.text)
	}

	return texts
}
