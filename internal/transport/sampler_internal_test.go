package transport

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	samplerScriptMode       fs.FileMode = 0o600
	testPathEnv                         = "PATH"
	testWSLDistroEnv                    = "WSL_DISTRO_NAME"
	testWSLDistroName                   = "Ubuntu"
	readWSLHostMetricsLine              = `wsl_host_metrics_json="$(read_wsl_windows_host_metrics_json)"`
	applyWSLHostMetricsLine             = `apply_wsl_host_metrics`
	allHostMetricsPrintLine             = `printf '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s' "${remote_cpu_name}" "${remote_cpu_cores}" "${cpu_freq_mhz}" "${cpu_max_freq_mhz}" "${cpu_temp_c}" "${ram_used}" "${ram_total}" "${ram_available}" "${ram_free}" "${ram_cache}" "${ram_buffers}" "${ram_reclaimable}" "${ram_shared}" "${mem_pressure_some}" "${mem_pressure_full}"`
)

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

func runSamplerSnippet(t *testing.T, snippet string, env map[string]string) string {
	t.Helper()

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required to validate sampler functions")
	}

	scriptPath := filepath.Join(t.TempDir(), "sampler-functions.sh")
	script := samplerFunctionPreamble(t) + "\n" + snippet + "\n"
	if err := os.WriteFile(scriptPath, []byte(script), samplerScriptMode); err != nil {
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

func samplerFunctionPreamble(t *testing.T) string {
	t.Helper()

	const mainMarker = "\nremote_name=\"$(hostname)\""
	preamble, _, found := strings.Cut(remoteSampler, mainMarker)
	if !found {
		t.Fatalf("sampler script missing main marker %q", mainMarker)
	}

	return preamble
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
