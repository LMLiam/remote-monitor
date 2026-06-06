package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/parser"
)

const (
	samplerScriptMode           fs.FileMode = 0o600
	samplerModulesDir                       = "sampler"
	samplerManifestPath                     = "sampler/manifest.txt"
	testPathEnv                             = "PATH"
	testWSLDistroEnv                        = "WSL_DISTRO_NAME"
	testWSLDistroName                       = "Ubuntu"
	samplerConfigModule                     = "config.sh"
	samplerCPUModule                        = "cpu.sh"
	samplerProcessesModule                  = "processes.sh"
	testIntelDRMClassEnv                    = "REMOTE_MONITOR_DRM_CLASS_DIR"
	testIntelVendorFile                     = "device/vendor"
	testIntelDeviceFile                     = "device/device"
	testIntelUeventFile                     = "device/uevent"
	testIntelVendorValue                    = "0x8086\n"
	testAMDVendorValue                      = "0x1002\n"
	testAMDSysfsUUID                        = "amd-0000:0b:00.0"
	testVRAMTotalFile                       = "device/mem_info_vram_total"
	testVRAMUsedFile                        = "device/mem_info_vram_used"
	samplerJSONModule                       = "json.sh"
	samplerPowerModule                      = "power.sh"
	samplerGPUCommonModule                  = "gpu_common.sh"
	samplerNVIDIAModule                     = "gpu_nvidia.sh"
	samplerIntelModule                      = "gpu_intel.sh"
	samplerAMDModule                        = "gpu_amd.sh"
	testDiskstatsEnv                        = "REMOTE_MONITOR_DISKSTATS_FILE"
	testBlockSysEnv                         = "REMOTE_MONITOR_BLOCK_SYS_DIR"
	testPowerSupplyEnv                      = "REMOTE_MONITOR_POWER_SUPPLY_DIR"
	testPowerSupplyTypeFile                 = "type"
	testPowerSupplyCapacityFile             = "capacity"
	testPowerSupplyBattery0                 = "BAT0"
	testPowerSupplyBattery1                 = "BAT1"
	testPowerSupplyUPS0                     = "UPS0"
	readWSLHostMetricsLine                  = `wsl_host_metrics_json="$(read_wsl_windows_host_metrics_json)"`
	applyWSLHostMetricsLine                 = `apply_wsl_host_metrics`
	allHostMetricsPrintLine                 = `printf '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s' "${remote_cpu_name}" "${remote_cpu_cores}" "${cpu_freq_mhz}" "${cpu_max_freq_mhz}" "${cpu_temp_c}" "${ram_used}" "${ram_total}" "${ram_available}" "${ram_free}" "${ram_cache}" "${ram_buffers}" "${ram_reclaimable}" "${ram_shared}" "${mem_pressure_some}" "${mem_pressure_full}"`
	buildNetJSONLine                        = `build_net_json`
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

func TestRemoteSamplerRefreshesNetworkInterfacesDuringSampling(t *testing.T) {
	t.Parallel()

	awkPath, err := exec.LookPath("awk")
	if err != nil {
		t.Skip("awk is required to fake sampler network discovery")
	}

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "ip"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(binDir, "awk"), `#!/usr/bin/env bash
last="${@: -1}"
if [ "${last}" = "/proc/net/dev" ]; then
  tr ',' '\n' <<< "${REMOTE_MONITOR_TEST_NET_IFACES:-}"
  exit 0
fi
exec `+shellQuote(awkPath)+` "$@"
`)

	got := parseNetworkJSONLinesForTest(t, runSamplerModuleSnippet(t, []string{samplerConfigModule, samplerJSONModule, samplerCPUModule, "network.sh"}, strings.Join([]string{
		`sample_elapsed_ms=1000`,
		`network_refresh_samples=2`,
		`sample_index=0`,
		`export REMOTE_MONITOR_TEST_NET_IFACES=eth0`,
		`declare -A net_samples=(`,
		`  [eth0]="100|50|10|5|0|0|0|0|0|0"`,
		`  [wg0]="10|5|1|1|0|0|0|0|0|0"`,
		`)`,
		`read_net_sample() { printf '%s\n' "${net_samples[$1]:--1|-1|-1|-1|-1|-1|-1|-1|-1|-1}"; }`,
		`read_net_speed_mbps() { printf '1000\n'; }`,
		`discover_net_ifaces`,
		`prime_net_baselines`,
		`sample_index=1`,
		`net_samples[eth0]="200|100|20|10|0|0|0|0|0|0"`,
		buildNetJSONLine,
		`printf '\n'`,
		`export REMOTE_MONITOR_TEST_NET_IFACES=eth0,wg0`,
		`sample_index=2`,
		`net_samples[eth0]="300|150|30|15|0|0|0|0|0|0"`,
		buildNetJSONLine,
		`printf '\n'`,
		`export REMOTE_MONITOR_TEST_NET_IFACES=wg0`,
		`sample_index=4`,
		`net_samples[wg0]="20|15|2|2|0|0|0|0|0|0"`,
		buildNetJSONLine,
	}, "\n"), map[string]string{
		testPathEnv: prependTestPath(binDir),
	}))

	if len(got) != 3 {
		t.Fatalf("expected three network samples, got %#v", got)
	}
	if len(got[0]) != 1 || got[0][0].Iface != "eth0" || got[0][0].RXBps != 100 || got[0][0].TXBps != 50 {
		t.Fatalf("unexpected initial network sample: %#v", got[0])
	}
	if len(got[1]) != 2 || got[1][0].Iface != "eth0" || got[1][0].RXBps != 100 || got[1][1].Iface != "wg0" || got[1][1].RXBps != -1 {
		t.Fatalf("expected rediscovered wg0 with eth0 state preserved, got %#v", got[1])
	}
	if len(got[2]) != 1 || got[2][0].Iface != "wg0" || got[2][0].RXBps != 10 || got[2][0].TXBps != 10 {
		t.Fatalf("expected removed eth0 to stop emitting and wg0 state to continue, got %#v", got[2])
	}
}

func TestRemoteSamplerBuildsDiskJSONForMountedBlockDevices(t *testing.T) {
	t.Parallel()

	diskstatsPath := filepath.Join(t.TempDir(), "diskstats")
	writeDiskstatsFixture(t, diskstatsPath, "8 0 sda 100 5 1000 200 50 3 2000 100 1 500 800\n"+
		"259 0 nvme0n1 200 10 4000 300 100 4 8000 180 2 700 1000\n")

	blockSysDir := writeBlockDeviceFixture(t, []string{"sda", "sda1", "nvme0n1", "nvme0n1p1"})
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "df"), `#!/bin/sh
cat <<'DF'
Filesystem     1024-blocks  Used Available Capacity Mounted on
/dev/sda1           100000 40000     60000      40% /
tmpfs                10000    20      9980       1% /run
/dev/nvme0n1p1      200000 80000    120000      40% /mnt/data
DF
`)

	got := parseDiskJSONForTest(t, runSamplerModuleSnippet(t, []string{samplerConfigModule, samplerJSONModule, samplerCPUModule, "disk.sh"}, strings.Join([]string{
		`sample_elapsed_ms=1000`,
		`tracked_disk_devices=()`,
		`declare -A prev_disk_sample=()`,
		`discover_disk_devices`,
		`prime_disk_baselines`,
		`cat > "${REMOTE_MONITOR_DISKSTATS_FILE}" <<'DISKSTATS'`,
		`8 0 sda 110 9 1200 260 70 8 2300 220 4 550 950`,
		`259 0 nvme0n1 250 12 5000 400 130 9 9000 330 1 830 1250`,
		`DISKSTATS`,
		`build_disks_json`,
	}, "\n"), map[string]string{
		testPathEnv:      prependTestPath(binDir),
		testDiskstatsEnv: diskstatsPath,
		testBlockSysEnv:  blockSysDir,
	}))

	if len(got) != 2 {
		t.Fatalf("expected two mounted block devices, got %#v", got)
	}
	assertDiskJSONStat(t, got[0], core.DiskStat{
		Device:            "sda",
		ReadBps:           102400,
		WriteBps:          153600,
		ReadMergedPerSec:  4,
		WriteMergedPerSec: 5,
		Util:              5,
		AwaitMS:           6,
		QueueDepth:        0.15,
		Inflight:          4,
	})
	assertDiskJSONStat(t, got[1], core.DiskStat{
		Device:            "nvme0n1",
		ReadBps:           512000,
		WriteBps:          512000,
		ReadMergedPerSec:  2,
		WriteMergedPerSec: 5,
		Util:              13,
		AwaitMS:           3.12,
		QueueDepth:        0.25,
		Inflight:          1,
	})
}

func TestRemoteSamplerBuildsLaptopPowerJSONFromSysfs(t *testing.T) {
	t.Parallel()

	powerDir := writePowerSupplyFixture(t, map[string]map[string]string{
		"AC0": {
			testPowerSupplyTypeFile: "Mains\n",
			"online":                "0\n",
		},
		testPowerSupplyBattery0: {
			testPowerSupplyTypeFile:     "Battery\n",
			"present":                   "1\n",
			testPowerSupplyCapacityFile: "83\n",
			"status":                    "Discharging\n",
			"current_now":               "1500000\n",
			"voltage_now":               "12000000\n",
		},
	})

	got := parsePowerJSONForTest(t, runSamplerModuleSnippet(t, powerSamplerModules(), powerJSONSnippet(), map[string]string{
		testPowerSupplyEnv: powerDir,
	}))
	if got.ExternalPowerOnline != 0 || got.BatteryPercent != 83 || got.BatteryStatus != "Discharging" || got.PowerDrawWatts != 18 || got.UPSPresent != 0 || got.PowerSourceName != testPowerSupplyBattery0 {
		t.Fatalf("unexpected laptop power summary: %#v", got)
	}
	if len(got.Supplies) != 2 {
		t.Fatalf("expected AC and battery supplies, got %#v", got.Supplies)
	}
	if got.Supplies[0].Name != "AC0" || got.Supplies[0].Type != "Mains" || got.Supplies[0].Online != 0 {
		t.Fatalf("unexpected AC supply: %#v", got.Supplies[0])
	}
	if got.Supplies[1].Name != testPowerSupplyBattery0 || got.Supplies[1].CapacityPercent != 83 || got.Supplies[1].Status != "Discharging" || got.Supplies[1].PowerDrawWatts != 18 || got.Supplies[1].Present != 1 {
		t.Fatalf("unexpected battery supply: %#v", got.Supplies[1])
	}
}

func TestRemoteSamplerBuildsUPSPowerJSONFromSysfs(t *testing.T) {
	t.Parallel()

	powerDir := writePowerSupplyFixture(t, map[string]map[string]string{
		testPowerSupplyUPS0: {
			testPowerSupplyTypeFile:     "UPS\n",
			"online":                    "1\n",
			"present":                   "1\n",
			testPowerSupplyCapacityFile: "55\n",
			"status":                    "Full\n",
			"power_now":                 "24000000\n",
		},
	})

	got := parsePowerJSONForTest(t, runSamplerModuleSnippet(t, powerSamplerModules(), powerJSONSnippet(), map[string]string{
		testPowerSupplyEnv: powerDir,
	}))
	if got.ExternalPowerOnline != 1 || got.BatteryPercent != -1 || got.PowerDrawWatts != 24 || got.UPSPresent != 1 || got.PowerSourceName != testPowerSupplyUPS0 {
		t.Fatalf("unexpected UPS power summary: %#v", got)
	}
	if len(got.Supplies) != 1 {
		t.Fatalf("expected one UPS supply, got %#v", got.Supplies)
	}
	ups := got.Supplies[0]
	if ups.Name != testPowerSupplyUPS0 || ups.Type != "UPS" || ups.Online != 1 || ups.CapacityPercent != 55 || ups.Status != "Full" || ups.PowerDrawWatts != 24 || ups.Present != 1 {
		t.Fatalf("unexpected UPS supply: %#v", ups)
	}
}

func TestRemoteSamplerPowerJSONUsesSentinelsForMissingAndUnreadableFields(t *testing.T) {
	t.Parallel()

	powerDir := writePowerSupplyFixture(t, map[string]map[string]string{
		testPowerSupplyBattery1: {
			testPowerSupplyTypeFile:     "Battery\n",
			testPowerSupplyCapacityFile: "83\n",
		},
	})
	capacityPath := filepath.Join(powerDir, testPowerSupplyBattery1, testPowerSupplyCapacityFile)
	if err := os.Remove(capacityPath); err != nil {
		t.Fatalf("remove capacity file: %v", err)
	}
	// A directory is a root-safe stand-in for an unreadable scalar sysfs field.
	if err := os.Mkdir(capacityPath, 0o700); err != nil {
		t.Fatalf("replace capacity with directory: %v", err)
	}

	got := parsePowerJSONForTest(t, runSamplerModuleSnippet(t, powerSamplerModules(), powerJSONSnippet(), map[string]string{
		testPowerSupplyEnv: powerDir,
	}))
	if got.ExternalPowerOnline != -1 || got.BatteryPercent != -1 || got.BatteryStatus != "" || got.PowerDrawWatts != -1 || got.UPSPresent != 0 || got.PowerSourceName != testPowerSupplyBattery1 {
		t.Fatalf("unexpected partial power summary: %#v", got)
	}
	if len(got.Supplies) != 1 {
		t.Fatalf("expected one partial battery supply, got %#v", got.Supplies)
	}
	supply := got.Supplies[0]
	if supply.Name != testPowerSupplyBattery1 || supply.Type != "Battery" || supply.Online != -1 || supply.CapacityPercent != -1 || supply.Status != "" || supply.PowerDrawWatts != -1 || supply.Present != -1 {
		t.Fatalf("expected sentinels for unreadable and missing fields, got %#v", supply)
	}
}

func TestRemoteSamplerEscapesJSONControlCharacters(t *testing.T) {
	t.Parallel()

	got := runSamplerModuleSnippet(t, []string{samplerJSONModule}, "json_escape "+bashANSIControlLiteralForTest(), nil)
	want := "cpu" + escapedASCIIControlsForTest() + `\\\"name`
	if got != want {
		t.Fatalf("escaped JSON string mismatch\nwant %q\n got %q", want, got)
	}

	var parsed struct {
		Value string `json:"value"`
	}
	raw := `{"value":"` + got + `"}`
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("expected escaped control characters to produce valid JSON %q: %v", raw, err)
	}
	parsedWant := "cpu" + asciiControlsForTest() + "\\\"name"
	if parsed.Value != parsedWant {
		t.Fatalf("parsed escaped value mismatch\nwant %q\n got %q", parsedWant, parsed.Value)
	}
}

func TestRemoteSamplerEscapedControlCharactersParseAsSample(t *testing.T) {
	t.Parallel()

	line := runSamplerModuleSnippet(t, []string{samplerJSONModule}, fmt.Sprintf(strings.Join([]string{
		`cpu_name="$(json_escape %s)"`,
		`command="$(json_escape %s)"`,
		`printf '{"version":1,"cpu_name":"%%s","top_processes":[{"pid":4242,"command":"%%s","cpu_percent":12,"rss_mib":64}]}' "${cpu_name}" "${command}"`,
	}, "\n"), bashANSIControlLiteralForTest(), bashANSIControlLiteralForTest()), nil)

	var p parser.Parser
	got, ok := p.HandleLine(line)
	if !ok || got == nil {
		t.Fatalf("expected sampler-produced escaped controls to parse, line: %q, error: %v", line, p.LastError())
	}
	want := "cpu" + asciiControlsForTest() + "\\\"name"
	if got.CPUName != want {
		t.Fatalf("cpu name round trip mismatch\nwant %q\n got %q", want, got.CPUName)
	}
	if len(got.TopProcesses) != 1 || got.TopProcesses[0].Command != want {
		t.Fatalf("process command round trip mismatch\nwant %q\n got %#v", want, got.TopProcesses)
	}
}

func TestRemoteSamplerNumericNormalizationDoesNotCallTrim(t *testing.T) {
	t.Parallel()

	snippet := strings.Join([]string{
		`trim() {`,
		`  printf 'trim called for %s\n' "$1" >&2`,
		`  return 42`,
		`}`,
		`printf '%s|%s|%s|%s\n' "$(normalize_int ' 42 ')" "$(normalize_int ' n/a ')" "$(normalize_float ' 3.14 ')" "$(normalize_float ' bad ')"`,
	}, "\n")

	got := runSamplerModuleSnippet(t, []string{samplerJSONModule}, snippet, nil)
	want := "42|-1|3.14|-1"
	if got != want {
		t.Fatalf("numeric normalization should trim inline without calling trim\nwant %q\n got %q", want, got)
	}
}

func TestRemoteSamplerNumericNormalizationOutput(t *testing.T) {
	t.Parallel()

	snippet := strings.Join([]string{
		`printf '%s\n' \`,
		`  "$(normalize_int '')" \`,
		`  "$(normalize_int ' 42 ')" \`,
		`  "$(normalize_int '   ')" \`,
		`  "$(normalize_int '-7')" \`,
		`  "$(normalize_int 'N/A')" \`,
		`  "$(normalize_int '3.5')" \`,
		`  "$(normalize_float '')" \`,
		`  "$(normalize_float ' 3.14 ')" \`,
		`  "$(normalize_float $'\t \t')" \`,
		`  "$(normalize_float '-7')" \`,
		`  "$(normalize_float 'n/a')" \`,
		`  "$(normalize_float '1.2.3')"`,
	}, "\n")

	got := runSamplerModuleSnippet(t, []string{samplerJSONModule}, snippet, nil)
	want := strings.Join([]string{"-1", "42", "-1", "-7", "-1", "-1", "-1", "3.14", "-1", "-7", "-1", "-1"}, "\n")
	if got != want {
		t.Fatalf("numeric normalization output mismatch\nwant:\n%s\n got:\n%s", want, got)
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

func TestRemoteSamplerFiltersProcessesBeforeCountLimit(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "ps"), `#!/bin/sh
cat <<'PS'
101 95.4 102400 postgres postgres: writer process
102 83.2 204800 python app.py
103 12.4 409600 bash helper
104 7.1 512000 Python worker.py
105 1.3 1024 awk -v self=123
106 0.8 2048 python3 manage.py
PS
`)

	got := runSamplerModuleSnippet(t, []string{samplerProcessesModule}, strings.Join([]string{
		`process_sort="cpu"`,
		`process_filter="PYTHON"`,
		`process_count=2`,
		`read_top_process_snapshot`,
	}, "\n"), map[string]string{
		testPathEnv: prependTestPath(binDir),
	})
	const want = "102|83|200|python\n104|7|500|Python"
	if got != want {
		t.Fatalf("process snapshot mismatch\nwant %q\n got %q", want, got)
	}
}

func TestRemoteSamplerUsesMemorySortWhenRequested(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	argsPath := filepath.Join(t.TempDir(), "ps.args")
	writeExecutable(t, filepath.Join(binDir, "ps"), `#!/bin/sh
printf '%s\n' "$*" > "${PS_ARGS_FILE}"
cat <<'PS'
201 2.2 1048576 postgres writer
202 44.8 1024 python worker.py
PS
`)

	got := runSamplerModuleSnippet(t, []string{samplerProcessesModule}, strings.Join([]string{
		`process_sort="mem"`,
		`process_filter=""`,
		`process_count=4`,
		`read_top_process_snapshot`,
	}, "\n"), map[string]string{
		testPathEnv:    prependTestPath(binDir),
		"PS_ARGS_FILE": argsPath,
	})
	if !strings.Contains(readTestFile(t, argsPath), "--sort=-rss,-pcpu") {
		t.Fatalf("expected memory ps sort args, got %q", readTestFile(t, argsPath))
	}
	const want = "201|2|1024|postgres\n202|45|1|python"
	if got != want {
		t.Fatalf("process snapshot mismatch\nwant %q\n got %q", want, got)
	}
}

func TestRemoteSamplerOmitsMalformedProcessRowsFromJSON(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "ps"), `#!/bin/sh
cat <<'PS'
bad-row
301 nope 2048
303 nope 2048 node
304 2.4 nope ruby
302 11.4 4096 python worker.py
PS
`)

	got := runSamplerModuleSnippet(t, []string{samplerJSONModule, samplerProcessesModule}, strings.Join([]string{
		`process_sort="cpu"`,
		`process_filter=""`,
		`process_count=4`,
		`build_top_process_json`,
	}, "\n"), map[string]string{
		testPathEnv: prependTestPath(binDir),
	})
	const want = `[{"pid":302,"command":"python","cpu_percent":11,"rss_mib":4}]`
	if got != want {
		t.Fatalf("process JSON mismatch\nwant %q\n got %q", want, got)
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
		testVRAMTotalFile:                 "8589934592\n",
		testVRAMUsedFile:                  "3221225472\n",
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
		testVRAMTotalFile:                 "2147483648\n",
		testVRAMUsedFile:                  "536870912\n",
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

func TestRemoteSamplerBuildsAMDGPUJSONFromAMDSMI(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "amd-smi"), `#!/bin/sh
if [ "$1" = "metric" ] && [ "$2" = "--json" ]; then
  cat <<'JSON'
{
  "gpu": [
    {
      "gpu_id": 0,
      "asic": {
        "market_name": "AMD Radeon RX 7900 XTX",
        "uuid": "GPU-AMD-123",
        "bdf": "0000:0b:00.0"
      },
      "usage": {
        "gfx_activity": 73,
        "umc_activity": 44,
        "mm_activity": 12
      },
      "memory_usage": {
        "vram_used_mb": 12288,
        "vram_total_mb": 24576
      },
      "temperature": {
        "edge_celsius": 62
      },
      "power": {
        "average_socket_power_w": 315.50,
        "power_cap_w": 355.0
      },
      "fan": {
        "speed_percent": 58
      },
      "clock": {
        "gfxclk_mhz": 2485,
        "gfxclk_max_mhz": 2900,
        "memclk_mhz": 1248,
        "memclk_max_mhz": 1250
      },
      "pcie": {
        "current_gen": 4,
        "max_gen": 4,
        "current_width": 16,
        "max_width": 16
      },
      "performance": {
        "level": "auto",
        "throttle_status": "power cap"
      }
    },
    {
      "gpu_id": 1,
      "asic": {
        "market_name": "AMD Radeon RX 7800 XT",
        "uuid": "GPU-AMD-456",
        "bdf": "0000:0c:00.0"
      },
      "usage": {
        "gfx_activity": 31
      },
      "memory_usage": {
        "vram_used_mb": 4096,
        "vram_total_mb": 16384
      },
      "temperature": {
        "edge_celsius": 55
      }
    }
  ]
}
JSON
  exit 0
fi
exit 1
`)

	got := parseGPUJSONForTest(t, runSamplerModuleSnippet(t, amdGPUSamplerModules(), amdGPUJSONSnippet(), map[string]string{
		testPathEnv:          prependTestPath(binDir),
		testIntelDRMClassEnv: t.TempDir(),
	}))
	if len(got) != 2 {
		t.Fatalf("expected two amd-smi AMD GPUs, got %#v", got)
	}
	assertAMDSMIPrimaryGPU(t, got[0])
	assertAMDSMISecondaryGPU(t, got[1])
}

func assertAMDSMIPrimaryGPU(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	assertAMDSMIPrimaryIdentity(t, gpu)
	assertAMDSMIPrimaryMemoryAndSensors(t, gpu)
	assertAMDSMIPrimaryClocksAndLink(t, gpu)
	assertAMDSMIPrimaryState(t, gpu)
}

func assertAMDSMIPrimaryIdentity(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	if gpu.Index != 0 || gpu.UUID != "GPU-AMD-123" || gpu.Name != "AMD Radeon RX 7900 XTX" {
		t.Fatalf("unexpected amd-smi AMD identity: %#v", gpu)
	}
	if gpu.Util != 73 || gpu.DecoderUtil != 12 {
		t.Fatalf("unexpected amd-smi AMD utilization: %#v", gpu)
	}
}

func assertAMDSMIPrimaryMemoryAndSensors(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	if gpu.MemUsed != 12288 || gpu.MemTotal != 24576 || gpu.MemUtil != 50 {
		t.Fatalf("unexpected amd-smi AMD memory: %#v", gpu)
	}
	if gpu.Temp != 62 || gpu.PowerDraw != 315.5 || gpu.PowerLimit != 355 || gpu.Fan != 58 {
		t.Fatalf("unexpected amd-smi AMD sensors: %#v", gpu)
	}
}

func assertAMDSMIPrimaryClocksAndLink(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	if gpu.SMClock != 2485 || gpu.MaxSMClock != 2900 || gpu.MemClock != 1248 || gpu.MaxMemClock != 1250 || gpu.GraphicsClock != 2485 {
		t.Fatalf("unexpected amd-smi AMD clocks: %#v", gpu)
	}
	if gpu.PCIeGenCurrent != 4 || gpu.PCIeGenMax != 4 || gpu.PCIeWidthCurrent != 16 || gpu.PCIeWidthMax != 16 {
		t.Fatalf("unexpected amd-smi AMD PCIe fields: %#v", gpu)
	}
}

func assertAMDSMIPrimaryState(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	if gpu.ThrottleReasons != "power cap" || gpu.PState != "auto" || gpu.EncoderUtil != -1 {
		t.Fatalf("unexpected amd-smi AMD state/media fields: %#v", gpu)
	}
}

func assertAMDSMISecondaryGPU(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	if gpu.Index != 1 || gpu.UUID != "GPU-AMD-456" || gpu.Name != "AMD Radeon RX 7800 XT" {
		t.Fatalf("unexpected second amd-smi AMD identity: %#v", gpu)
	}
	if gpu.Util != 31 || gpu.MemUsed != 4096 || gpu.MemTotal != 16384 || gpu.MemUtil != 25 || gpu.Temp != 55 {
		t.Fatalf("unexpected second amd-smi AMD metrics: %#v", gpu)
	}
}

func TestRemoteSamplerBuildsAMDGPUJSONFromSysfsWhenToolsAreMissing(t *testing.T) {
	t.Parallel()

	drmDir := writeAMDSysfsFallbackFixture(t)

	got := parseGPUJSONForTest(t, runSamplerModuleSnippet(t, amdGPUSamplerModules(), amdGPUJSONSnippet(), map[string]string{
		testPathEnv:                             prependTestPath(t.TempDir()),
		testIntelDRMClassEnv:                    drmDir,
		"REMOTE_MONITOR_TEST_DISABLE_AMD_TOOLS": "1",
	}))
	if len(got) != 1 {
		t.Fatalf("expected one sysfs AMD GPU, got %#v", got)
	}
	gpu := got[0]
	if gpu.Index != 0 || gpu.UUID != testAMDSysfsUUID || gpu.Name != "AMD GPU 1002:744C" {
		t.Fatalf("unexpected sysfs AMD identity: %#v", gpu)
	}
	if gpu.MemUsed != 12288 || gpu.MemTotal != 24576 || gpu.MemUtil != 50 {
		t.Fatalf("unexpected sysfs AMD memory: %#v", gpu)
	}
	if gpu.Temp != 62 || gpu.PowerDraw != 315.5 || gpu.PowerLimit != 355 {
		t.Fatalf("unexpected sysfs AMD sensors: %#v", gpu)
	}
	if gpu.Util != -1 || gpu.Fan != -1 || gpu.PState != "" {
		t.Fatalf("expected sysfs AMD unavailable metrics to use sentinels, got %#v", gpu)
	}
}

func TestRemoteSamplerFallsBackToSysfsWhenAMDSMIJSONHasNoGPUData(t *testing.T) {
	t.Parallel()

	assertAMDSysfsFallbackAfterEmptyToolJSON(t, "amd-smi")
}

func TestRemoteSamplerBuildsAMDGPUJSONFromROCMSMIWhenAMDSMIIsMissing(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "rocm-smi"), `#!/bin/sh
cat <<'JSON'
{
  "card0": {
    "Card series": "AMD Radeon PRO W7900",
    "Unique ID": "0xabcdef123456",
    "GPU use (%)": "68",
    "GPU Memory Allocated (VRAM%)": "40",
    "VRAM Total Memory (B)": "34359738368",
    "VRAM Total Used Memory (B)": "13743895347",
    "Temperature (Sensor edge) (C)": "59.0",
    "Average Graphics Package Power (W)": "220.5",
    "Max Graphics Package Power (W)": "295.0",
    "Fan Level": "45%",
    "sclk clock level": "0: 500Mhz\n1: 2100Mhz *",
    "mclk clock level": "0: 96Mhz\n1: 1000Mhz *",
    "Performance Level": "auto"
  }
}
JSON
`)

	got := parseGPUJSONForTest(t, runSamplerModuleSnippet(t, amdGPUSamplerModules(), amdGPUJSONSnippet(), map[string]string{
		testPathEnv:          prependTestPath(binDir),
		testIntelDRMClassEnv: t.TempDir(),
	}))
	if len(got) != 1 {
		t.Fatalf("expected one rocm-smi AMD GPU, got %#v", got)
	}
	gpu := got[0]
	if gpu.Index != 0 || gpu.UUID != "0xabcdef123456" || gpu.Name != "AMD Radeon PRO W7900" {
		t.Fatalf("unexpected rocm-smi AMD identity: %#v", gpu)
	}
	if gpu.Util != 68 || gpu.MemUtil != 40 || gpu.MemUsed != 13107 || gpu.MemTotal != 32768 {
		t.Fatalf("unexpected rocm-smi AMD utilization/memory: %#v", gpu)
	}
	if gpu.Temp != 59 || gpu.PowerDraw != 220.5 || gpu.PowerLimit != 295 || gpu.Fan != 45 {
		t.Fatalf("unexpected rocm-smi AMD sensors: %#v", gpu)
	}
	if gpu.SMClock != 2100 || gpu.MemClock != 1000 || gpu.GraphicsClock != 2100 || gpu.PState != "auto" {
		t.Fatalf("unexpected rocm-smi AMD clocks/state: %#v", gpu)
	}
}

func TestRemoteSamplerFallsBackToSysfsWhenROCMSMIJSONHasNoGPUData(t *testing.T) {
	t.Parallel()

	assertAMDSysfsFallbackAfterEmptyToolJSON(t, "rocm-smi")
}

func assertAMDSysfsFallbackAfterEmptyToolJSON(t *testing.T, toolName string) {
	t.Helper()

	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, toolName), `#!/bin/sh
printf '{}'
`)

	got := parseGPUJSONForTest(t, runSamplerModuleSnippet(t, amdGPUSamplerModules(), amdGPUJSONSnippet(), map[string]string{
		testPathEnv:          prependTestPath(binDir),
		testIntelDRMClassEnv: writeAMDSysfsFallbackFixture(t),
	}))
	if len(got) != 1 {
		t.Fatalf("expected sysfs AMD GPU fallback, got %#v", got)
	}
	if got[0].UUID != testAMDSysfsUUID || got[0].MemUsed != 12288 || got[0].Temp != 62 {
		t.Fatalf("unexpected sysfs AMD fallback after empty %s JSON: %#v", toolName, got[0])
	}
}

func TestRemoteSamplerIgnoresAbsentAMDSources(t *testing.T) {
	t.Parallel()

	got := runSamplerModuleSnippet(t, amdGPUSamplerModules(), amdGPUJSONSnippet(), map[string]string{
		testPathEnv:                             prependTestPath(t.TempDir()),
		testIntelDRMClassEnv:                    t.TempDir(),
		"REMOTE_MONITOR_TEST_DISABLE_AMD_TOOLS": "1",
	})
	if got != "[]" {
		t.Fatalf("expected no AMD GPUs without tooling or sysfs devices, got %q", got)
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
	return []string{samplerJSONModule, samplerGPUCommonModule, samplerNVIDIAModule, samplerIntelModule, samplerAMDModule}
}

func amdGPUJSONSnippet() string {
	return strings.Join([]string{
		`nvidia_smi_path=""`,
		`if [ "${REMOTE_MONITOR_TEST_DISABLE_AMD_TOOLS:-0}" = "1" ]; then`,
		`  amd_smi_path=""`,
		`  rocm_smi_path=""`,
		`else`,
		`  amd_smi_path="$(command -v amd-smi 2>/dev/null || true)"`,
		`  rocm_smi_path="$(command -v rocm-smi 2>/dev/null || true)"`,
		`fi`,
		`amd_drm_class_path="${REMOTE_MONITOR_DRM_CLASS_DIR:-/sys/class/drm}"`,
		`build_amd_gpu_json`,
	}, "\n")
}

func amdGPUSamplerModules() []string {
	return []string{samplerJSONModule, samplerGPUCommonModule, samplerAMDModule}
}

type powerJSONForTest struct {
	ExternalPowerOnline int                    `json:"external_power_online"`
	BatteryPercent      int                    `json:"battery_percent"`
	BatteryStatus       string                 `json:"battery_status"`
	PowerDrawWatts      float64                `json:"power_draw_w"`
	UPSPresent          int                    `json:"ups_present"`
	PowerSourceName     string                 `json:"source_name"`
	Supplies            []core.PowerSupplyStat `json:"supplies"`
}

func powerJSONSnippet() string {
	return `power_supply_class_path="${REMOTE_MONITOR_POWER_SUPPLY_DIR:-/sys/class/power_supply}"` + "\n" + `build_power_json`
}

func powerSamplerModules() []string {
	return []string{samplerJSONModule, samplerPowerModule}
}

func parsePowerJSONForTest(t *testing.T, raw string) powerJSONForTest {
	t.Helper()

	var got powerJSONForTest
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse power JSON %q: %v", raw, err)
	}

	return got
}

func parseGPUJSONForTest(t *testing.T, raw string) []core.GPUStat {
	t.Helper()

	var got []core.GPUStat
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse GPU JSON %q: %v", raw, err)
	}

	return got
}

func parseNetworkJSONLinesForTest(t *testing.T, raw string) [][]core.NetStat {
	t.Helper()

	var samples [][]core.NetStat
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var sample []core.NetStat
		if err := json.Unmarshal([]byte(line), &sample); err != nil {
			t.Fatalf("parse network JSON %q: %v", line, err)
		}
		samples = append(samples, sample)
	}

	return samples
}

func parseDiskJSONForTest(t *testing.T, raw string) []core.DiskStat {
	t.Helper()

	var got []core.DiskStat
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse disk JSON %q: %v", raw, err)
	}

	return got
}

func assertDiskJSONStat(t *testing.T, got, want core.DiskStat) {
	t.Helper()

	if got != want {
		t.Fatalf("disk JSON mismatch\nwant %#v\n got %#v", want, got)
	}
}

func asciiControlsForTest() string {
	var value strings.Builder
	for code := 1; code < 0x20; code++ {
		value.WriteByte(byte(code))
	}

	return value.String()
}

func bashANSIControlLiteralForTest() string {
	var value strings.Builder
	value.WriteString("$'cpu")
	for code := 1; code < 0x20; code++ {
		_, _ = fmt.Fprintf(&value, `\%03o`, code)
	}
	value.WriteString(`\\"name'`)

	return value.String()
}

func escapedASCIIControlsForTest() string {
	var value strings.Builder
	for code := 1; code < 0x20; code++ {
		switch code {
		case '\b':
			value.WriteString(`\b`)
		case '\t':
			value.WriteString(`\t`)
		case '\n':
			value.WriteString(`\n`)
		case '\f':
			value.WriteString(`\f`)
		case '\r':
			value.WriteString(`\r`)
		default:
			_, _ = fmt.Fprintf(&value, `\u%04x`, code)
		}
	}

	return value.String()
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

func writeAMDDRMFixture(t *testing.T, card string, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	writeIntelDRMCardFixture(t, root, card, files)

	return root
}

func writeAMDSysfsFallbackFixture(t *testing.T) string {
	t.Helper()

	return writeAMDDRMFixture(t, "card2", map[string]string{
		testIntelVendorFile:                  testAMDVendorValue,
		testIntelDeviceFile:                  "0x744c\n",
		testIntelUeventFile:                  "PCI_ID=1002:744C\nPCI_SLOT_NAME=0000:0b:00.0\n",
		"device/hwmon/hwmon1/temp1_input":    "62000\n",
		"device/hwmon/hwmon1/power1_average": "315500000\n",
		"device/hwmon/hwmon1/power1_cap":     "355000000\n",
		testVRAMTotalFile:                    "25769803776\n",
		testVRAMUsedFile:                     "12884901888\n",
	})
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

func writePowerSupplyFixture(t *testing.T, supplies map[string]map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, files := range supplies {
		supplyRoot := filepath.Join(root, name)
		if err := os.MkdirAll(supplyRoot, 0o700); err != nil {
			t.Fatalf("create power supply fixture: %v", err)
		}
		for file, contents := range files {
			path := filepath.Join(supplyRoot, file)
			if err := os.WriteFile(path, []byte(contents), samplerScriptMode); err != nil {
				t.Fatalf("write power supply fixture %s: %v", path, err)
			}
		}
	}

	return root
}

func writeBlockDeviceFixture(t *testing.T, devices []string) string {
	t.Helper()

	root := t.TempDir()
	for _, device := range devices {
		deviceRoot := filepath.Join(root, device)
		if err := os.MkdirAll(deviceRoot, 0o700); err != nil {
			t.Fatalf("create block device fixture %s: %v", device, err)
		}
		if device == "sda1" || strings.HasSuffix(device, "p1") {
			path := filepath.Join(deviceRoot, "partition")
			if err := os.WriteFile(path, []byte("1\n"), samplerScriptMode); err != nil {
				t.Fatalf("write block device fixture %s: %v", path, err)
			}
		}
	}

	return root
}

func writeDiskstatsFixture(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), samplerScriptMode); err != nil {
		t.Fatalf("write diskstats fixture %s: %v", path, err)
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
		samplerConfigModule,
		samplerJSONModule,
		samplerCPUModule,
		"processes.sh",
		"memory.sh",
		"pressure.sh",
		"wsl.sh",
		"filesystems.sh",
		"disk.sh",
		"network.sh",
		samplerPowerModule,
		samplerGPUCommonModule,
		samplerNVIDIAModule,
		samplerIntelModule,
		samplerAMDModule,
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

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	// #nosec G304 -- path is created inside the calling test's temporary directory.
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(contents)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
