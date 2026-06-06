//nolint:testpackage // Alerts are assembled by unexported helper functions worth covering directly.
package render

import (
	"reflect"
	"testing"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
)

func TestAlertSummaryUsesThresholds(t *testing.T) {
	t.Parallel()

	defaultThresholds := core.DefaultThresholds()
	cpuSample := alertNominalSample()
	cpuSample.CPUPercent = defaultThresholds.CPUCriticalPercent

	customCPUThresholds := defaultThresholds
	customCPUThresholds.CPUCriticalPercent = defaultThresholds.CPUCriticalPercent + 1

	customDiskThresholds := defaultThresholds
	customDiskThresholds.DiskWarnPercent = 40
	customDiskThresholds.DiskCriticalPercent = 90
	diskSample := alertNominalSample()
	diskSample.RootUsedPercent = 45

	tests := []struct {
		name      string
		state     core.AppState
		wantLevel string
		wantText  string
	}{
		{
			name:      "default thresholds report saturated CPU",
			state:     alertTestState(cpuSample, defaultThresholds),
			wantLevel: severityCritical,
			wantText:  "cpu saturated",
		},
		{
			name:      "custom CPU threshold suppresses default CPU alert",
			state:     alertTestState(cpuSample, customCPUThresholds),
			wantLevel: severityOK,
			wantText:  "nominal",
		},
		{
			name:      "custom disk threshold changes alert severity",
			state:     alertTestState(diskSample, customDiskThresholds),
			wantLevel: severityWarn,
			wantText:  "disk high",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotLevel, gotText := AlertSummary(tc.state)
			if gotLevel != tc.wantLevel || gotText != tc.wantText {
				t.Fatalf("AlertSummary() = (%q, %q), want (%q, %q)", gotLevel, gotText, tc.wantLevel, tc.wantText)
			}
		})
	}
}

func TestAppendAlertIssuesUseThresholdBoundaries(t *testing.T) {
	t.Parallel()

	thresholds := core.DefaultThresholds()

	cpuPercentSample := alertNominalSample()
	cpuPercentSample.CPUPercent = thresholds.CPUCriticalPercent
	cpuTempSample := alertNominalSample()
	cpuTempSample.CPUTempC = thresholds.CPUWarnTemp
	memorySample := alertNominalSample()
	memorySample.RAMAvailableMiB = 15
	memorySample.RAMTotalMiB = 100
	gpuTempSample := alertNominalSample()
	gpuTempSample.GPUs = []core.GPUStat{gpuWithTemp(thresholds.GPUCriticalTemp)}
	vramSample := alertNominalSample()
	vramSample.GPUs = []core.GPUStat{gpuWithVRAMPercent(thresholds.VRAMWarnPercent)}
	diskSample := alertNominalSample()
	diskSample.RootUsedPercent = thresholds.DiskCriticalPercent

	tests := []struct {
		name       string
		appendFunc func([]alertIssue, core.Sample, core.Thresholds) []alertIssue
		sample     core.Sample
		want       []alertIssue
	}{
		{
			name:       "cpu critical percent boundary",
			appendFunc: appendCPUAlertIssues,
			sample:     cpuPercentSample,
			want:       []alertIssue{{severity: severityCritical, text: "cpu saturated"}},
		},
		{
			name:       "cpu warn temperature boundary",
			appendFunc: appendCPUAlertIssues,
			sample:     cpuTempSample,
			want:       []alertIssue{{severity: severityWarn, text: "cpu warm"}},
		},
		{
			name:       "memory warn availability boundary",
			appendFunc: appendMemoryAlertIssues,
			sample:     memorySample,
			want:       []alertIssue{{severity: severityWarn, text: "ram tight"}},
		},
		{
			name:       "gpu critical temperature boundary",
			appendFunc: appendGPUAlertIssues,
			sample:     gpuTempSample,
			want:       []alertIssue{{severity: severityCritical, text: "gpu hot"}},
		},
		{
			name:       "vram warn boundary",
			appendFunc: appendGPUAlertIssues,
			sample:     vramSample,
			want:       []alertIssue{{severity: severityWarn, text: "vram high"}},
		},
		{
			name:       "disk critical boundary",
			appendFunc: appendDiskAlertIssues,
			sample:     diskSample,
			want:       []alertIssue{{severity: severityCritical, text: "disk full"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.appendFunc(nil, tc.sample, thresholds)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("issues = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func alertTestState(sample core.Sample, thresholds core.Thresholds) core.AppState {
	var state core.AppState
	state.Cfg.Thresholds = thresholds
	state.Cfg.StaleAfter = time.Hour
	state.RuntimeState = core.StatusLive
	state.Current = sample
	state.HasSample = true
	state.LastRx = time.Now()

	return state
}

func alertNominalSample() core.Sample {
	sample := core.EmptySample()
	sample.RAMAvailableMiB = 80
	sample.RAMTotalMiB = 100
	sample.RootUsedPercent = 0
	sample.DiskAwaitMS = -1
	sample.DiskQueueDepth = -1

	return sample
}

func gpuWithTemp(temp int) core.GPUStat {
	var gpu core.GPUStat
	gpu.Temp = temp

	return gpu
}

func gpuWithVRAMPercent(percent int) core.GPUStat {
	var gpu core.GPUStat
	gpu.MemTotal = 100
	gpu.MemUsed = int64(percent)

	return gpu
}
