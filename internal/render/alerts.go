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

	cpuCriticalPercent = 95
	cpuCriticalTempC   = 85
	cpuWarnTempC       = 75

	ramCriticalAvailPercent = 5
	ramWarnAvailPercent     = 15

	gpuCriticalTempC    = 80
	gpuWarnTempC        = 70
	vramCriticalPercent = 95
	vramWarnPercent     = 85

	diskCriticalUsedPercent = 95
	diskWarnUsedPercent     = 90
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
	issues := make([]alertIssue, 0, alertIssueCapacity)
	switch currentStatus(state) {
	case core.StatusDisconnected:
		issues = appendAlertIssue(issues, severityCritical, core.StatusDisconnected)
	case core.StatusStale:
		issues = appendAlertIssue(issues, severityWarn, core.StatusStale)
	}

	issues = appendCPUAlertIssues(issues, s)
	issues = appendMemoryAlertIssues(issues, s)
	issues = appendGPUAlertIssues(issues, s)
	issues = appendDiskAlertIssues(issues, s)

	return appendNetworkAlertIssues(issues, s)
}

func appendAlertIssue(issues []alertIssue, severity, text string) []alertIssue {
	return append(issues, alertIssue{
		severity: severity,
		text:     text,
	})
}

func appendCPUAlertIssues(issues []alertIssue, s core.Sample) []alertIssue {
	if s.CPUPercent >= cpuCriticalPercent {
		issues = appendAlertIssue(issues, severityCritical, "cpu saturated")
	}
	if s.CPUTempC >= cpuCriticalTempC {
		issues = appendAlertIssue(issues, severityCritical, "cpu hot")
	} else if s.CPUTempC >= cpuWarnTempC {
		issues = appendAlertIssue(issues, severityWarn, "cpu warm")
	}

	return issues
}

func appendMemoryAlertIssues(issues []alertIssue, s core.Sample) []alertIssue {
	if ramAvailPct := metrics.RAMAvailablePercent(s); ramAvailPct >= 0 {
		switch {
		case ramAvailPct <= ramCriticalAvailPercent:
			issues = appendAlertIssue(issues, severityCritical, "ram low")
		case ramAvailPct <= ramWarnAvailPercent:
			issues = appendAlertIssue(issues, severityWarn, "ram tight")
		}
	}

	return issues
}

func appendGPUAlertIssues(issues []alertIssue, s core.Sample) []alertIssue {
	if metrics.OverallTempValue(s) >= gpuCriticalTempC {
		issues = appendAlertIssue(issues, severityCritical, "gpu hot")
	} else if metrics.OverallTempValue(s) >= gpuWarnTempC {
		issues = appendAlertIssue(issues, severityWarn, "gpu warm")
	}
	if vramPct := metrics.OverallVRAMPct(s); vramPct >= vramCriticalPercent {
		issues = appendAlertIssue(issues, severityCritical, "vram high")
	} else if vramPct >= vramWarnPercent {
		issues = appendAlertIssue(issues, severityWarn, "vram high")
	}

	return issues
}

func appendDiskAlertIssues(issues []alertIssue, s core.Sample) []alertIssue {
	switch {
	case s.RootUsedPercent >= diskCriticalUsedPercent:
		issues = appendAlertIssue(issues, severityCritical, "disk full")
	case s.RootUsedPercent >= diskWarnUsedPercent:
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
