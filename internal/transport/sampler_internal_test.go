package transport

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
)

const (
	samplerScriptMode       fs.FileMode = 0o600
	samplerModulesDir                   = "sampler"
	samplerManifestPath                 = "sampler/manifest.txt"
	testPathEnv                         = "PATH"
	testWSLDistroEnv                    = "WSL_DISTRO_NAME"
	testWSLDistroName                   = "Ubuntu"
	testIntelDRMClassEnv                = "REMOTE_MONITOR_DRM_CLASS_DIR"
	testIntelVendorFile                 = "device/vendor"
	testIntelDeviceFile                 = "device/device"
	testIntelUeventFile                 = "device/uevent"
	testIntelVendorValue                = "0x8086\n"
	samplerJSONModule                   = "json.sh"
	samplerNVIDIAModule                 = "gpu_nvidia.sh"
	samplerIntelModule                  = "gpu_intel.sh"
	readWSLHostMetricsLine              = `wsl_host_metrics_json="$(read_wsl_windows_host_metrics_json)"`
	applyWSLHostMetricsLine             = `apply_wsl_host_metrics`
	allHostMetricsPrintLine             = `printf '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s' "${remote_cpu_name}" "${remote_cpu_cores}" "${cpu_freq_mhz}" "${cpu_max_freq_mhz}" "${cpu_temp_c}" "${ram_used}" "${ram_total}" "${ram_available}" "${ram_free}" "${ram_cache}" "${ram_buffers}" "${ram_reclaimable}" "${ram_shared}" "${mem_pressure_some}" "${mem_pressure_full}"`
)

func TestRemoteSamplerMatchesAssembledModules(t *testing.T) {
	t.Parallel()

	assembled := assembleSamplerModulesForTest(t)
	if !bytes.Equal([]byte(remoteSampler), assembled) {
		t.Fatalf("embedded sampler.sh is not the deterministic assembly of %s; run the sampler assembly step", samplerManifestPath)
	}
}

func TestRemoteSamplerModuleManifestCoversExpectedCollectors(t *testing.T) {
	t.Parallel()

	manifest := readSamplerManifestForTest(t)
	expected := expectedSamplerModules()
	if strings.Join(manifest, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("sampler manifest changed unexpectedly\nwant %s\n got %s", expected, manifest)
	}
	for _, module := range manifest {
		if _, err := os.Stat(filepath.Join(samplerModulesDir, module)); err != nil {
			t.Fatalf("sampler module %s is not available: %v", module, err)
		}
	}
}

func TestRemoteSamplerPressureModuleCanBeSourcedIndependently(t *testing.T) {
	t.Parallel()

	pressureFile := writeSamplerTestFile(t, "some avg10=1.23 avg60=2.00 avg300=3.00 total=4\nfull avg10=0.45 avg60=1.00 avg300=2.00 total=3\n")
	got := runSamplerModuleSnippet(t, []string{samplerJSONModule, "pressure.sh"}, "read_pressure_avg10 "+shellQuote(pressureFile), nil)
	if got != "1.23|0.45" {
		t.Fatalf("expected pressure module to parse avg10 values, got %q", got)
	}
}

func TestRemoteSamplerShellSyntax(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required to validate the remote sampler script")
	}
	if strings.TrimSpace(remoteSampler) == "" {
		t.Fatal("embedded remote sampler script is empty")
	}

	path := filepath.Join(t.TempDir(), "sampler.sh")
	if err := os.WriteFile(path, []byte(remoteSampler), samplerScriptMode); err != nil {
		t.Fatalf("write sampler script: %v", err)
	}

	cmd := exec.Command("bash", "-n", "sampler.sh")
	cmd.Dir = filepath.Dir(path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sampler script failed bash syntax check: %v\n%s", err, output)
	}
}

func TestRemoteSamplerDetectsWSLFromProcAndEnvironment(t *testing.T) {
	t.Parallel()

	wslOSRelease := writeSamplerTestFile(t, "5.15.90.1-microsoft-standard-WSL2\n")
	nativeVersion := writeSamplerTestFile(t, "Linux version 6.8.0-generic\n")
	got := runSamplerSnippet(t, "if is_wsl_environment "+shellQuote(wslOSRelease)+" "+shellQuote(nativeVersion)+"; then printf wsl; else printf native; fi", nil)
	if got != "wsl" {
		t.Fatalf("expected WSL from osrelease, got %q", got)
	}

	nativeOSRelease := writeSamplerTestFile(t, "6.8.0-31-generic\n")
	got = runSamplerSnippet(t, "if is_wsl_environment "+shellQuote(nativeOSRelease)+" "+shellQuote(nativeVersion)+"; then printf wsl; else printf native; fi", nil)
	if got != "native" {
		t.Fatalf("expected native Linux from proc files, got %q", got)
	}

	got = runSamplerSnippet(t, "if is_wsl_environment "+shellQuote(nativeOSRelease)+" "+shellQuote(nativeVersion)+"; then printf wsl; else printf native; fi", map[string]string{
		"WSL_INTEROP": "/run/WSL/123_interop",
	})
	if got != "wsl" {
		t.Fatalf("expected WSL from WSL_INTEROP, got %q", got)
	}
}

func TestRemoteSamplerReadsWindowsHostMetricsFromPowerShellJSON(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	argsPath := filepath.Join(t.TempDir(), "powershell.args")
	writeExecutable(t, filepath.Join(binDir, "timeout"), "#!/bin/sh\nshift\nexec \"$@\"\n")
	writeExecutable(t, filepath.Join(binDir, "powershell.exe"), "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"${POWERSHELL_ARGS_FILE}\"\nprintf '%s\\n' '{\"cpu_temp_c\":67,\"cpu_name\":\"AMD Host CPU\",\"cpu_cores\":16,\"cpu_freq_mhz\":3600,\"cpu_max_freq_mhz\":4700,\"ram_used_mib\":8192,\"ram_total_mib\":32768,\"ram_available_mib\":24576,\"ram_free_mib\":24576}'\n")

	got := runSamplerSnippet(t, wslHostMetricSnippet("WSL CPU", "-1", "-1", "-1", allHostMetricsPrintLine), map[string]string{
		testPathEnv:                       binDir,
		testWSLDistroEnv:                  testWSLDistroName,
		"POWERSHELL_ARGS_FILE":            argsPath,
		"REMOTE_MONITOR_WSL_HOST_METRICS": "1",
	})
	const want = "AMD Host CPU|16|3600|4700|67|8192|32768|24576|24576|-1|-1|-1|-1|-1|-1"
	if got != want {
		t.Fatalf("expected Windows host metrics\nwant %q\n got %q", want, got)
	}

	// #nosec G304 -- argsPath is a test-controlled temporary file.
	argsBytes, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read captured PowerShell args: %v", err)
	}
	args := string(argsBytes)
	if strings.Contains(args, "MSAcpi_ThermalZoneTemperature") {
		t.Fatalf("expected PowerShell args to avoid ACPI thermal zones as CPU temperature, got %q", args)
	}
	if strings.Contains(args, `$_.Identifier -match "/cpu/|/temperature/"`) {
		t.Fatalf("expected PowerShell args to avoid accepting any hardware-monitor temperature identifier as CPU temp, got %q", args)
	}
	for _, want := range []string{"-NoProfile", "-NonInteractive", "-Command", "Get-CimInstance", "Win32_Processor", "Win32_OperatingSystem", "root/LibreHardwareMonitor", "root/OpenHardwareMonitor"} {
		if !strings.Contains(args, want) {
			t.Fatalf("expected PowerShell args to contain %q, got %q", want, args)
		}
	}
	if want := `$_.Identifier -match "/(cpu|intelcpu|amdcpu)(/|$)" -and $_.Identifier -match "/temperature/"`; !strings.Contains(args, want) {
		t.Fatalf("expected PowerShell args to require CPU and temperature identifier components, got %q", args)
	}
}

func TestRemoteSamplerFindsWindowsPowerShellOutsidePath(t *testing.T) {
	t.Parallel()

	fakePowerShellDir := filepath.Join(t.TempDir(), "WindowsPowerShell")
	if err := os.MkdirAll(fakePowerShellDir, 0o700); err != nil {
		t.Fatalf("create fake PowerShell directory: %v", err)
	}
	fakePowerShell := filepath.Join(fakePowerShellDir, "powershell.exe")
	writeExecutable(t, fakePowerShell, "#!/bin/sh\nprintf '{}'\n")

	got := runSamplerSnippet(t, "find_wsl_powershell_from_candidates powershell.exe "+shellQuote(fakePowerShell), map[string]string{
		testPathEnv: t.TempDir(),
	})
	if got != fakePowerShell {
		t.Fatalf("expected absolute PowerShell fallback path, got %q", got)
	}
}

func TestRemoteSamplerKeepsLinuxCPUTemperatureWhenAvailable(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "timeout"), "#!/bin/sh\nshift\nexec \"$@\"\n")
	writeExecutable(t, filepath.Join(binDir, "powershell.exe"), "#!/bin/sh\nprintf '%s\\n' '{\"cpu_temp_c\":67}'\n")

	got := runSamplerSnippet(t, wslHostMetricSnippet("Linux CPU", "2400", "3600", "55", `printf '%s' "${cpu_temp_c}"`), map[string]string{
		testPathEnv:      binDir,
		testWSLDistroEnv: testWSLDistroName,
	})
	if got != "55" {
		t.Fatalf("expected Linux CPU temperature to win, got %q", got)
	}
}

func TestRemoteSamplerLeavesWindowsHostMetricsUnchangedWithoutUsablePowerShellData(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"missing-powershell": "",
		"null-metrics":       "#!/bin/sh\nprintf '%s\\n' '{\"cpu_temp_c\":null,\"cpu_name\":null,\"cpu_cores\":null,\"ram_total_mib\":null}'\n",
		"bad-metrics":        "#!/bin/sh\nprintf '%s\\n' '{\"cpu_temp_c\":200,\"cpu_name\":\"\",\"cpu_cores\":0,\"cpu_freq_mhz\":0,\"cpu_max_freq_mhz\":0,\"ram_used_mib\":900,\"ram_total_mib\":0,\"ram_available_mib\":-1,\"ram_free_mib\":-1}'\n",
		"command-failure":    "#!/bin/sh\nexit 1\n",
		"timeout":            "#!/bin/sh\nprintf '%s\\n' '{\"cpu_temp_c\":70,\"cpu_name\":\"AMD Host CPU\",\"cpu_cores\":16,\"ram_total_mib\":32768}'\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			binDir := t.TempDir()
			timeoutBody := "#!/bin/sh\nshift\nexec \"$@\"\n"
			if name == "timeout" {
				timeoutBody = "#!/bin/sh\nexit 124\n"
			}
			writeExecutable(t, filepath.Join(binDir, "timeout"), timeoutBody)
			if body != "" {
				writeExecutable(t, filepath.Join(binDir, "powershell.exe"), body)
			}

			got := runSamplerSnippet(t, wslHostMetricSnippet("Linux CPU", "2400", "3600", "55", allHostMetricsPrintLine), map[string]string{
				testPathEnv:      binDir,
				testWSLDistroEnv: testWSLDistroName,
			})
			const want = "Linux CPU|8|2400|3600|55|100|200|50|40|30|20|10|5|1.20|0.10"
			if got != want {
				t.Fatalf("expected unchanged Linux metrics\nwant %q\n got %q", want, got)
			}
		})
	}
}

func TestRemoteSamplerSkipsPowerShellMetricsOutsideWSL(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	nativeOSRelease := writeSamplerTestFile(t, "6.8.0-31-generic\n")
	nativeVersion := writeSamplerTestFile(t, "Linux version 6.8.0-generic\n")
	writeExecutable(t, filepath.Join(binDir, "timeout"), "#!/bin/sh\nshift\nexec \"$@\"\n")
	writeExecutable(t, filepath.Join(binDir, "powershell.exe"), "#!/bin/sh\nprintf '%s\\n' '{\"cpu_temp_c\":67,\"cpu_name\":\"AMD Host CPU\"}'\n")

	got := runSamplerSnippet(t, `metrics="$(read_wsl_windows_host_metrics_json `+shellQuote(nativeOSRelease)+" "+shellQuote(nativeVersion)+`)"; if [ -n "${metrics}" ]; then printf '%s' "${metrics}"; else printf empty; fi`, map[string]string{
		testPathEnv: binDir,
	})
	if got != "empty" {
		t.Fatalf("expected native Linux to skip PowerShell metrics, got %q", got)
	}
}

func TestRemoteSamplerCanDisableWSLHostMetrics(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "timeout"), "#!/bin/sh\nshift\nexec \"$@\"\n")
	writeExecutable(t, filepath.Join(binDir, "powershell.exe"), "#!/bin/sh\nprintf '%s\\n' '{\"cpu_temp_c\":67,\"cpu_name\":\"AMD Host CPU\"}'\n")

	got := runSamplerSnippet(t, `metrics="$(read_wsl_windows_host_metrics_json)"; if [ -n "${metrics}" ]; then printf '%s' "${metrics}"; else printf empty; fi`, map[string]string{
		testPathEnv:                       binDir,
		testWSLDistroEnv:                  testWSLDistroName,
		"REMOTE_MONITOR_WSL_HOST_METRICS": "0",
	})
	if got != "empty" {
		t.Fatalf("expected disabled WSL host metrics to skip PowerShell, got %q", got)
	}
}

func TestRemoteSamplerBuildsIntelGPUJSONFromIntelGPUTop(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "intel_gpu_top"), `#!/bin/sh
cat <<'JSON'
[
{
  "period": {"duration": 1000.000000, "unit": "ms"},
  "frequency": {"requested": 1300.000000, "actual": 1016.400000, "unit": "MHz"},
  "power": {"GPU": 14.750000, "Package": 31.500000, "unit": "W"},
  "engines": {
    "Render/3D": {"busy": 42.800000, "sema": 0.000000, "wait": 0.000000, "unit": "%"},
    "Compute": {"busy": 61.200000, "sema": 0.000000, "wait": 0.000000, "unit": "%"},
    "Video": {"busy": 7.400000, "sema": 0.000000, "wait": 0.000000, "unit": "%"}
  },
  "clients": {}
}
]
JSON
	`)
	drmDir := writeIntelDRMFixture(t, "card0", map[string]string{
		testIntelVendorFile:               testIntelVendorValue,
		testIntelDeviceFile:               "0x56a5\n",
		testIntelUeventFile:               "PCI_ID=8086:56A5\nPCI_SLOT_NAME=0000:03:00.0\n",
		"device/hwmon/hwmon0/temp1_input": "53000\n",
		"device/hwmon/hwmon0/power1_cap":  "45000000\n",
		"device/mem_info_vram_total":      "8589934592\n",
		"device/mem_info_vram_used":       "3221225472\n",
	})

	got := parseGPUJSONForTest(t, runSamplerModuleSnippet(t, intelGPUSamplerModules(), intelGPUJSONSnippet(), map[string]string{
		testPathEnv:          prependTestPath(binDir),
		testIntelDRMClassEnv: drmDir,
	}))
	if len(got) != 1 {
		t.Fatalf("expected one Intel GPU, got %#v", got)
	}
	gpu := got[0]
	if gpu.Index != 0 || gpu.UUID != "intel-0000:03:00.0" || gpu.Name != "Intel GPU 8086:56A5" {
		t.Fatalf("unexpected Intel GPU identity: %#v", gpu)
	}
	if gpu.Util != 61 || gpu.SMClock != 1016 || gpu.GraphicsClock != 1016 {
		t.Fatalf("unexpected Intel utilization/clocks: %#v", gpu)
	}
	if gpu.MemUsed != 3072 || gpu.MemTotal != 8192 || gpu.MemUtil != 38 {
		t.Fatalf("unexpected Intel memory values: %#v", gpu)
	}
	if gpu.Temp != 53 || gpu.PowerDraw != 14.75 || gpu.PowerLimit != 45 {
		t.Fatalf("unexpected Intel thermal/power values: %#v", gpu)
	}
	if gpu.EncoderUtil != -1 || gpu.DecoderUtil != 7 || gpu.Fan != -1 || gpu.PState != "" {
		t.Fatalf("unexpected Intel sentinel/source values: %#v", gpu)
	}
}

func TestRemoteSamplerBuildsIntelGPUJSONFromSysfsWhenToolIsMissing(t *testing.T) {
	t.Parallel()

	drmDir := writeIntelDRMFixture(t, "card1", map[string]string{
		testIntelVendorFile:               testIntelVendorValue,
		testIntelDeviceFile:               "0x9a49\n",
		testIntelUeventFile:               "PCI_ID=8086:9A49\nPCI_SLOT_NAME=0000:00:02.0\n",
		"device/hwmon/hwmon2/temp1_input": "44000\n",
	})

	got := parseGPUJSONForTest(t, runSamplerModuleSnippet(t, intelGPUSamplerModules(), intelGPUJSONSnippet(), map[string]string{
		testPathEnv:          prependTestPath(t.TempDir()),
		testIntelDRMClassEnv: drmDir,
		"REMOTE_MONITOR_TEST_DISABLE_INTEL_TOOLS": "1",
	}))
	if len(got) != 1 {
		t.Fatalf("expected one sysfs Intel GPU, got %#v", got)
	}
	gpu := got[0]
	if gpu.Index != 0 || gpu.UUID != "intel-0000:00:02.0" || gpu.Name != "Intel GPU 8086:9A49" {
		t.Fatalf("unexpected sysfs Intel identity: %#v", gpu)
	}
	if gpu.Util != -1 || gpu.MemUsed != -1 || gpu.MemTotal != -1 || gpu.Temp != 44 || gpu.PowerDraw != -1 {
		t.Fatalf("expected sysfs Intel sentinel metrics with temperature, got %#v", gpu)
	}
	if gpu.PState != "" {
		t.Fatalf("expected unavailable Intel p-state to stay hidden, got %#v", gpu)
	}
}

func TestRemoteSamplerBuildsIntelGPUJSONFromXPUSMI(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "xpu-smi"), `#!/bin/sh
if [ "$1" = "discovery" ] && [ "$2" = "--dump" ]; then
  cat <<'CSV'
Device ID,Device Name,UUID,PCI BDF Address,Memory Physical Size
0,"Intel(R) Data Center GPU Flex 170","00000000-0000-0000-0000-56c000008086","0000:4d:00.0","16384.00 MiB"
CSV
  exit 0
fi
if [ "$1" = "dump" ]; then
  cat <<'CSV'
Timestamp, DeviceId, Average % utilization of all GPU Engines, GPU Power (W), GPU Frequency (MHz), GPU Core Temperature (Celsius Degree), GPU Memory Used (MiB), Compute engine utilizations (%), Render engine utilizations (%), Media decoder engine utilizations (%), Media encoder engine utilizations (%), Throttle reason, Media Engine Frequency (MHz)
06:14:46.000, 0, 55.25, 88.50, 1450, 64, 8192, 61.00, 48.00, 14.00, 9.00, "power cap", 950
CSV
  exit 0
fi
exit 1
`)

	got := parseGPUJSONForTest(t, runSamplerModuleSnippet(t, intelGPUSamplerModules(), intelGPUJSONSnippet(), map[string]string{
		testPathEnv:          prependTestPath(binDir),
		testIntelDRMClassEnv: t.TempDir(),
	}))
	if len(got) != 1 {
		t.Fatalf("expected one xpu-smi Intel GPU, got %#v", got)
	}
	gpu := got[0]
	if gpu.Index != 0 || gpu.UUID != "00000000-0000-0000-0000-56c000008086" || gpu.Name != "Intel(R) Data Center GPU Flex 170" {
		t.Fatalf("unexpected xpu-smi Intel identity: %#v", gpu)
	}
	if gpu.Util != 55 || gpu.MemUtil != 50 || gpu.MemUsed != 8192 || gpu.MemTotal != 16384 {
		t.Fatalf("unexpected xpu-smi Intel utilization/memory: %#v", gpu)
	}
	if gpu.PowerDraw != 88.5 || gpu.Temp != 64 || gpu.SMClock != 1450 || gpu.VideoClock != 950 {
		t.Fatalf("unexpected xpu-smi Intel power/temp/clocks: %#v", gpu)
	}
	if gpu.EncoderUtil != 9 || gpu.DecoderUtil != 14 || gpu.ThrottleReasons != "power cap" {
		t.Fatalf("unexpected xpu-smi Intel media/throttle: %#v", gpu)
	}
}

func TestRemoteSamplerMergesXPUSMIWithSysfsIntelDevices(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "xpu-smi"), `#!/bin/sh
if [ "$1" = "discovery" ] && [ "$2" = "--dump" ]; then
  cat <<'CSV'
Device ID,Device Name,UUID,PCI BDF Address,Memory Physical Size
0,"Intel(R) Data Center GPU Flex 170","00000000-0000-0000-0000-56c000008086","0000:4d:00.0","16384.00 MiB"
CSV
  exit 0
fi
if [ "$1" = "dump" ]; then
  cat <<'CSV'
Timestamp, DeviceId, Average % utilization of all GPU Engines, GPU Power (W), GPU Frequency (MHz), GPU Core Temperature (Celsius Degree), GPU Memory Used (MiB), Compute engine utilizations (%), Render engine utilizations (%), Media decoder engine utilizations (%), Media encoder engine utilizations (%), Throttle reason, Media Engine Frequency (MHz)
06:14:46.000, 0, 55.25, 88.50, 1450, 64, 8192, 61.00, 48.00, 14.00, 9.00, "power cap", 950
CSV
  exit 0
fi
exit 1
	`)
	drmDir := t.TempDir()
	writeIntelDRMCardFixture(t, drmDir, "card0", map[string]string{
		testIntelVendorFile: testIntelVendorValue,
		testIntelDeviceFile: "0x56c0\n",
		testIntelUeventFile: "PCI_ID=8086:56C0\nPCI_SLOT_NAME=0000:4d:00.0\n",
	})
	writeIntelDRMCardFixture(t, drmDir, "card1", map[string]string{
		testIntelVendorFile:               testIntelVendorValue,
		testIntelDeviceFile:               "0x9a49\n",
		testIntelUeventFile:               "PCI_ID=8086:9A49\nPCI_SLOT_NAME=0000:00:02.0\n",
		"device/hwmon/hwmon2/temp1_input": "44000\n",
		"device/mem_info_vram_total":      "2147483648\n",
		"device/mem_info_vram_used":       "536870912\n",
	})

	got := parseGPUJSONForTest(t, runSamplerModuleSnippet(t, intelGPUSamplerModules(), intelGPUJSONSnippet(), map[string]string{
		testPathEnv:          prependTestPath(binDir),
		testIntelDRMClassEnv: drmDir,
	}))
	if len(got) != 2 {
		t.Fatalf("expected xpu-smi and sysfs Intel GPUs, got %#v", got)
	}
	if got[0].Index != 0 || got[0].UUID != "00000000-0000-0000-0000-56c000008086" {
		t.Fatalf("unexpected xpu-smi Intel GPU: %#v", got[0])
	}
	if got[1].Index != 1 || got[1].UUID != "intel-0000:00:02.0" || got[1].Name != "Intel GPU 8086:9A49" {
		t.Fatalf("unexpected sysfs Intel GPU: %#v", got[1])
	}
	if got[1].MemUsed != 512 || got[1].MemTotal != 2048 || got[1].MemUtil != 25 || got[1].Temp != 44 {
		t.Fatalf("unexpected merged sysfs Intel metrics: %#v", got[1])
	}
}

func TestRemoteSamplerIgnoresAbsentIntelSources(t *testing.T) {
	t.Parallel()

	got := runSamplerModuleSnippet(t, intelGPUSamplerModules(), intelGPUJSONSnippet(), map[string]string{
		testPathEnv:          prependTestPath(t.TempDir()),
		testIntelDRMClassEnv: t.TempDir(),
		"REMOTE_MONITOR_TEST_DISABLE_INTEL_TOOLS": "1",
	})
	if got != "[]" {
		t.Fatalf("expected no Intel GPUs without tooling or sysfs devices, got %q", got)
	}
}

func wslHostMetricSnippet(cpuName, cpuFreqMHz, cpuMaxFreqMHz, cpuTempC, printLine string) string {
	return strings.Join([]string{
		readWSLHostMetricsLine,
		`remote_cpu_name="` + cpuName + `"`,
		`remote_cpu_cores=8`,
		`cpu_freq_mhz=` + cpuFreqMHz,
		`cpu_max_freq_mhz=` + cpuMaxFreqMHz,
		`cpu_temp_c=` + cpuTempC,
		`ram_used=100`,
		`ram_total=200`,
		`ram_available=50`,
		`ram_free=40`,
		`ram_cache=30`,
		`ram_buffers=20`,
		`ram_reclaimable=10`,
		`ram_shared=5`,
		`mem_pressure_some=1.20`,
		`mem_pressure_full=0.10`,
		applyWSLHostMetricsLine,
		printLine,
	}, "\n")
}

func intelGPUJSONSnippet() string {
	return strings.Join([]string{
		`nvidia_smi_path=""`,
		`if [ "${REMOTE_MONITOR_TEST_DISABLE_INTEL_TOOLS:-0}" = "1" ]; then`,
		`  intel_gpu_top_path=""`,
		`  xpu_smi_path=""`,
		`else`,
		`  intel_gpu_top_path="$(command -v intel_gpu_top 2>/dev/null || true)"`,
		`  xpu_smi_path="$(command -v xpu-smi 2>/dev/null || true)"`,
		`fi`,
		`intel_drm_class_path="${REMOTE_MONITOR_DRM_CLASS_DIR:-/sys/class/drm}"`,
		`build_gpu_json`,
	}, "\n")
}

func intelGPUSamplerModules() []string {
	return []string{samplerJSONModule, samplerNVIDIAModule, samplerIntelModule}
}

func parseGPUJSONForTest(t *testing.T, raw string) []core.GPUStat {
	t.Helper()

	var got []core.GPUStat
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse GPU JSON %q: %v", raw, err)
	}

	return got
}

func prependTestPath(dir string) string {
	return dir + string(os.PathListSeparator) + os.Getenv(testPathEnv)
}

func writeIntelDRMFixture(t *testing.T, card string, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	writeIntelDRMCardFixture(t, root, card, files)

	return root
}

func writeIntelDRMCardFixture(t *testing.T, root, card string, files map[string]string) {
	t.Helper()

	cardRoot := filepath.Join(root, card)
	if err := os.MkdirAll(cardRoot, 0o700); err != nil {
		t.Fatalf("create drm fixture card: %v", err)
	}
	for name, contents := range files {
		path := filepath.Join(cardRoot, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("create drm fixture dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), samplerScriptMode); err != nil {
			t.Fatalf("write drm fixture %s: %v", name, err)
		}
	}
}

func runSamplerSnippet(t *testing.T, snippet string, env map[string]string) string {
	t.Helper()

	script := samplerFunctionPreamble(t) + "\n" + snippet + "\n"

	return runRawSamplerSnippet(t, script, env)
}

func runSamplerModuleSnippet(t *testing.T, modules []string, snippet string, env map[string]string) string {
	t.Helper()

	var script strings.Builder
	for _, module := range modules {
		modulePath := filepath.Join(samplerModulesDir, module)
		if _, err := os.Stat(modulePath); err != nil {
			t.Fatalf("sampler module %s is not available: %v", modulePath, err)
		}
		script.WriteString(". ")
		script.WriteString(shellQuote(modulePath))
		script.WriteByte('\n')
	}
	script.WriteString(snippet)

	return runRawSamplerSnippet(t, script.String(), env)
}

func samplerFunctionPreamble(t *testing.T) string {
	t.Helper()

	const mainMarker = "\nremote_name=\"$(hostname)\""
	preamble, _, found := strings.Cut(remoteSampler, mainMarker)
	if !found {
		t.Fatalf("sampler script missing main marker %q", mainMarker)
	}

	return preamble
}

func assembleSamplerModulesForTest(t *testing.T) []byte {
	t.Helper()

	var assembled bytes.Buffer
	for index, module := range readSamplerManifestForTest(t) {
		if filepath.Base(module) != module {
			t.Fatalf("sampler manifest module %q must be a file name", module)
		}
		path := filepath.Join(samplerModulesDir, module)
		// #nosec G304 -- module is restricted to a file name listed by the local test fixture manifest.
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read sampler module %s: %v", path, err)
		}
		if index > 0 {
			assembled.WriteByte('\n')
		}
		if _, err := assembled.Write(contents); err != nil {
			t.Fatalf("assemble sampler module %s: %v", path, err)
		}
		if len(contents) == 0 || contents[len(contents)-1] != '\n' {
			assembled.WriteByte('\n')
		}
	}

	return assembled.Bytes()
}

func readSamplerManifestForTest(t *testing.T) []string {
	t.Helper()

	contents, err := os.ReadFile(samplerManifestPath)
	if err != nil {
		t.Fatalf("read sampler manifest %s: %v", samplerManifestPath, err)
	}

	var modules []string
	for line := range strings.SplitSeq(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		modules = append(modules, line)
	}
	if len(modules) == 0 {
		t.Fatalf("sampler manifest %s does not list any modules", samplerManifestPath)
	}

	return modules
}

func expectedSamplerModules() []string {
	return []string{
		"config.sh",
		samplerJSONModule,
		"cpu.sh",
		"processes.sh",
		"memory.sh",
		"pressure.sh",
		"wsl.sh",
		"filesystems.sh",
		"disk.sh",
		"network.sh",
		samplerNVIDIAModule,
		samplerIntelModule,
		"main.sh",
	}
}

func runRawSamplerSnippet(t *testing.T, script string, env map[string]string) string {
	t.Helper()

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required to validate sampler functions")
	}

	scriptPath := filepath.Join(t.TempDir(), "sampler-functions.sh")
	if err := os.WriteFile(scriptPath, []byte(script+"\n"), samplerScriptMode); err != nil {
		t.Fatalf("write sampler function script: %v", err)
	}

	pathValue := os.Getenv("PATH")
	if override, ok := env[testPathEnv]; ok {
		pathValue = override
	}
	// #nosec G204 -- scriptPath is a test-controlled temporary script.
	cmd := exec.Command("bash", scriptPath)
	cmd.Env = []string{
		"HOME=" + filepath.Dir(scriptPath),
		"LC_ALL=C",
		testPathEnv + "=" + pathValue,
	}
	for key, value := range env {
		if key == testPathEnv {
			continue
		}
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sampler snippet failed: %v\n%s", err, output)
	}

	return strings.TrimSpace(string(output))
}

func writeSamplerTestFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, []byte(content), samplerScriptMode); err != nil {
		t.Fatalf("write sampler test file: %v", err)
	}

	return path
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()

	// #nosec G306 -- fake commands must be executable by the test shell.
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
