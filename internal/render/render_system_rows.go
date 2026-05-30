package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

func buildSystemSummaryRows(state core.AppState, summaryWidth int) []StatusSummaryRowSpec {
	status := currentStatus(state)
	statusDetail := fmt.Sprintf("samples %d • reconnects %d", state.SampleCount, state.ReconnectCount)
	alertSeverity, alertText := AlertSummary(state)
	stateChipWidth := min(summaryWidth, max(summaryStateChipMin, summaryWidth/summaryStateChipDiv))
	alertChipWidth := min(summaryWidth, max(summaryAlertChipMin, summaryWidth/summaryAlertChipDiv))

	return []StatusSummaryRowSpec{
		statusSummaryRow("State", statusColor(status), ansi.Pad(
			chipCell(strings.ToUpper(statusValueText(state)), stateChipWidth, statusBackground(status))+
				" "+ansi.Colorize(statusColor(status), ansi.FitText(statusDetail, max(0, summaryWidth-stateChipWidth-1))),
			summaryWidth,
		)),
		statusSummaryRow("Alert", SeverityColor(alertSeverity), ansi.Pad(
			chipCell(strings.ToUpper(alertSeverity), alertChipWidth, severityBackground(alertSeverity))+
				" "+ansi.Colorize(SeverityColor(alertSeverity), ansi.FitText(alertText, max(0, summaryWidth-alertChipWidth-1))),
			summaryWidth,
		)),
	}
}
