package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"strings"
)

const (
	powerBatteryCriticalPercent = 10
	powerBatteryWarnPercent     = 20
	powerBatteryInfoPercent     = 40
	powerRowsBaseCapacity       = 4
	powerSummaryPartsCapacity   = 4
	powerSupplyValuePartsCap    = 3
	powerSupplyActivityPartsCap = 2
)

// BuildPowerRows builds optional Linux power-supply rows for the dashboard.
func BuildPowerRows(state core.AppState, activityWidth int, condensed bool) []TableRowSpec {
	s := state.Current
	if !HasPowerData(s) {
		return nil
	}

	rows := make([]TableRowSpec, 0, powerRowsBaseCapacity+len(s.PowerSupplies))
	if s.ExternalPowerOnline >= 0 {
		severity := externalPowerSeverity(s.ExternalPowerOnline)
		rows = append(rows, TableFullRow("External", SeverityColor(severity), externalPowerText(s.ExternalPowerOnline), SeverityColor(severity), "", powerSourceActivity(s), ansi.Cyan, ""))
	}
	if s.BatteryPercent >= 0 || strings.TrimSpace(s.BatteryStatus) != "" {
		severity := batteryPercentSeverity(s.BatteryPercent)
		rows = append(rows, TableFullRow("Battery", SeverityColor(severity), batterySummaryValue(s), SeverityColor(severity), "", fallbackString(s.BatteryStatus, TextNA), SeverityColor(severity), batteryGaugeCell(s.BatteryPercent, activityWidth, severity)))
	}
	if s.PowerDrawWatts >= 0 {
		rows = append(rows, TableFullRow("Power Draw", ansi.Amber, formatPowerValue(s.PowerDrawWatts), ansi.Amber, "", "estimated", ansi.Amber, ""))
	}
	if s.UPSPresent == 1 {
		rows = append(rows, TableFullRow("UPS", ansi.Green, "present", ansi.Green, "", fallbackString(upsSupplyName(s), "sysfs"), ansi.Green, ""))
	}

	if !condensed && len(s.PowerSupplies) > 0 {
		if len(rows) > 0 {
			rows = append(rows, tableDividerRow())
		}
		for _, supply := range s.PowerSupplies {
			rows = append(rows, TableFullRow(powerSupplyLabel(supply), ansi.Cyan, powerSupplyValue(supply), ansi.Cyan, "", powerSupplyActivity(supply), ansi.Cyan, ""))
		}
	}

	return rows
}

// HasPowerData reports whether a sample carries any power-supply data.
func HasPowerData(s core.Sample) bool {
	return len(s.PowerSupplies) > 0 ||
		s.ExternalPowerOnline >= 0 ||
		s.BatteryPercent >= 0 ||
		strings.TrimSpace(s.BatteryStatus) != "" ||
		s.PowerDrawWatts >= 0 ||
		s.UPSPresent == 1 ||
		strings.TrimSpace(s.PowerSourceName) != ""
}

// PowerSummaryText returns a compact text-mode power summary.
func PowerSummaryText(s core.Sample) string {
	if !HasPowerData(s) {
		return ""
	}

	parts := make([]string, 0, powerSummaryPartsCapacity)
	if s.ExternalPowerOnline >= 0 {
		parts = append(parts, "AC "+externalPowerText(s.ExternalPowerOnline))
	}
	if s.BatteryPercent >= 0 || strings.TrimSpace(s.BatteryStatus) != "" {
		parts = append(parts, strings.TrimSpace(batterySummaryValue(s)+" "+strings.TrimSpace(s.BatteryStatus)))
	}
	if s.PowerDrawWatts >= 0 {
		parts = append(parts, formatPowerValue(s.PowerDrawWatts))
	}
	if s.UPSPresent == 1 {
		parts = append(parts, "UPS present")
	}
	if len(parts) == 0 && s.PowerSourceName != "" {
		parts = append(parts, s.PowerSourceName)
	}

	return strings.Join(parts, " • ")
}

func externalPowerText(value int) string {
	if value == 1 {
		return "online"
	}

	return "offline"
}

func externalPowerSeverity(value int) string {
	if value == 1 {
		return severityOK
	}

	return severityWarn
}

func batterySummaryValue(s core.Sample) string {
	source := fallbackString(s.PowerSourceName, "battery")
	if s.BatteryPercent >= 0 {
		return fmt.Sprintf("%s %s", source, percentDisplay(s.BatteryPercent))
	}

	return source
}

func batteryGaugeCell(percent, activityWidth int, severity string) string {
	if percent < 0 {
		return ""
	}

	return gaugeBarCell(percent, activityWidth, SeverityColor(severity), percentDisplay(percent))
}

func batteryPercentSeverity(percent int) string {
	if percent < 0 {
		return severityNeutral
	}
	switch {
	case percent <= powerBatteryCriticalPercent:
		return severityCritical
	case percent <= powerBatteryWarnPercent:
		return severityWarn
	case percent <= powerBatteryInfoPercent:
		return severityInfo
	default:
		return severityOK
	}
}

func powerSourceActivity(s core.Sample) string {
	if strings.TrimSpace(s.PowerSourceName) == "" {
		return ""
	}

	return "source " + s.PowerSourceName
}

func upsSupplyName(s core.Sample) string {
	for _, supply := range s.PowerSupplies {
		if strings.EqualFold(supply.Type, "UPS") {
			return supply.Name
		}
	}

	return ""
}

func powerSupplyLabel(supply core.PowerSupplyStat) string {
	return "Supply " + fallbackString(supply.Name, TextNA)
}

func powerSupplyValue(supply core.PowerSupplyStat) string {
	parts := make([]string, 0, powerSupplyValuePartsCap)
	if strings.TrimSpace(supply.Type) != "" {
		parts = append(parts, supply.Type)
	}
	if supply.Online >= 0 {
		parts = append(parts, externalPowerText(supply.Online))
	}
	if supply.CapacityPercent >= 0 {
		parts = append(parts, percentDisplay(supply.CapacityPercent))
	}
	if strings.TrimSpace(supply.Status) != "" {
		parts = append(parts, supply.Status)
	}
	if len(parts) == 0 {
		return TextNA
	}

	return strings.Join(parts, " ")
}

func powerSupplyActivity(supply core.PowerSupplyStat) string {
	parts := make([]string, 0, powerSupplyActivityPartsCap)
	if supply.PowerDrawWatts >= 0 {
		parts = append(parts, formatPowerValue(supply.PowerDrawWatts))
	}
	if supply.Present >= 0 {
		present := "absent"
		if supply.Present == 1 {
			present = "present"
		}
		parts = append(parts, present)
	}

	return strings.Join(parts, " ")
}
