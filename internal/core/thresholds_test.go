package core_test

import (
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
)

func TestDefaultThresholdsMatchCurrentAlertDefaults(t *testing.T) {
	t.Parallel()

	got := core.DefaultThresholds()
	want := core.Thresholds{
		CPUCriticalPercent:          95,
		CPUWarnTemp:                 75,
		CPUCriticalTemp:             85,
		RAMWarnAvailablePercent:     15,
		RAMCriticalAvailablePercent: 5,
		GPUWarnTemp:                 70,
		GPUCriticalTemp:             80,
		VRAMWarnPercent:             85,
		VRAMCriticalPercent:         95,
		DiskWarnPercent:             90,
		DiskCriticalPercent:         95,
	}

	if got != want {
		t.Fatalf("DefaultThresholds() = %#v, want %#v", got, want)
	}
}
