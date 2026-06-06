package parser_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/parser"

	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testCPUName       = "AMD Ryzen 5 5600X 6-Core Processor"
	testDataMount     = "/mnt/data"
	testGPUUUID       = "GPU-123"
	testIfaceEth0     = "eth0"
	testPythonCommand = "python"
)

func TestParserBuildsFullSampleFromJSONLine(t *testing.T) {
	t.Parallel()

	var p parser.Parser
	line := `{"version":1,"epoch":1716912345,"timestamp":"2026-05-28 19:35:45","remote":"DESKTOP","uptime_seconds":14340,"load1":10.55,"load5":4.82,"load15":4.09,"cpu_cores":12,"cpu_name":"AMD Ryzen 5 5600X 6-Core Processor","cpu_percent":99,"cpu_user_percent":71,"cpu_system_percent":19,"cpu_iowait_percent":6,"cpu_steal_percent":1,"ram_used_mib":2455,"ram_total_mib":15967,"ram_available_mib":13512,"ram_free_mib":12041,"ram_cache_mib":3120,"ram_buffers_mib":288,"ram_reclaimable_mib":601,"ram_shared_mib":92,"cpu_freq_mhz":3680,"cpu_max_freq_mhz":4700,"cpu_temp_c":66,"cpu_pressure_some_avg10":2.43,"cpu_pressure_full_avg10":0.14,"mem_pressure_some_avg10":1.20,"mem_pressure_full_avg10":0.04,"swap":{"free_kib":0,"total_kib":4194304,"in_bps":8192,"out_bps":4096},"disk":{"root_source":"/dev/sdd","root_used_kib":42000000,"root_total_kib":100000000,"root_used_percent":42,"device":"sdd","read_bps":1048576,"write_bps":524288,"read_merged_per_sec":12,"write_merged_per_sec":7,"util_percent":12,"await_ms":1.37,"queue_depth":0.21,"inflight":3},"net":[{"iface":"eth0","rx_bps":125000,"tx_bps":24000,"rx_pps":1024,"tx_pps":512,"speed_mbps":1000,"rx_drops":2,"rx_errors":1,"rx_overruns":3,"tx_drops":0,"tx_errors":0,"tx_overruns":1},{"iface":"tailscale0","rx_bps":8000,"tx_bps":4000,"rx_pps":64,"tx_pps":32,"speed_mbps":-1,"rx_drops":0,"rx_errors":0,"rx_overruns":0,"tx_drops":0,"tx_errors":0,"tx_overruns":0}],"filesystems":[{"source":"/dev/sdd","mount":"/","used_kib":42000000,"total_kib":100000000,"used_percent":42,"inodes_used_percent":18},{"source":"/dev/sdc","mount":"/mnt/data","used_kib":8000000,"total_kib":20000000,"used_percent":40,"inodes_used_percent":11}],"tcp_retrans_segs_per_sec":9,"tcp_resets_per_sec":1,"cpu_core_usage":[{"index":0,"percent":10}],"top_processes":[{"pid":4242,"command":"python","cpu_percent":88,"rss_mib":2048}],"gpu_processes":[{"gpu_uuid":"GPU-123","pid":4242,"command":"python","used_mem_mib":3072}],"gpus":[{"index":0,"uuid":"GPU-123","name":"NVIDIA GeForce RTX 3060 | Quiet","util_percent":0,"mem_util_percent":34,"encoder_util_percent":12,"decoder_util_percent":8,"mem_used_mib":2003,"mem_total_mib":12288,"temp_c":55,"power_draw_w":26.57,"power_limit_w":170.0,"fan_percent":0,"sm_clock_mhz":210,"sm_clock_max_mhz":2100,"mem_clock_mhz":810,"mem_clock_max_mhz":7501,"graphics_clock_mhz":1740,"video_clock_mhz":1620,"pcie_gen_current":3,"pcie_gen_max":4,"pcie_width_current":8,"pcie_width_max":16,"throttle_reasons":"power cap","p_state":"P5"}]}`

	got, ok := p.HandleLine(line)
	if !ok || got == nil {
		t.Fatalf("expected completed Sample from Parser")
	}
	assertParsedHostAndCPU(t, got)
	assertParsedMemoryAndPressure(t, got)
	assertParsedNetworkAndStorage(t, got)
	assertParsedProcesses(t, got)
	assertParsedGPU(t, got)
}

func assertParsedHostAndCPU(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.RemoteName != "DESKTOP" {
		t.Fatalf("remote name = %q", got.RemoteName)
	}
	if got.CPUCores != 12 || got.CPUPercent != 99 {
		t.Fatalf("cpu cores/percent = %d/%d", got.CPUCores, got.CPUPercent)
	}
	if got.CPUUserPercent != 71 || got.CPUSystemPercent != 19 || got.CPUIOWaitPercent != 6 || got.CPUStealPercent != 1 {
		t.Fatalf("cpu breakdown = %d/%d/%d/%d", got.CPUUserPercent, got.CPUSystemPercent, got.CPUIOWaitPercent, got.CPUStealPercent)
	}
	if got.CPUName != testCPUName {
		t.Fatalf("cpu name = %q", got.CPUName)
	}
	if got.CPUFreqMHz != 3680 || got.CPUMaxFreqMHz != 4700 || got.CPUTempC != 66 {
		t.Fatalf("cpu freq/max/temp = %d/%d/%d", got.CPUFreqMHz, got.CPUMaxFreqMHz, got.CPUTempC)
	}
}

func assertParsedMemoryAndPressure(t *testing.T, got *core.Sample) {
	t.Helper()

	assertParsedSwap(t, got)
	assertParsedMemory(t, got)
	assertParsedPressure(t, got)
}

func assertParsedSwap(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.SwapTotalKiB != 4194304 || got.SwapInBps != 8192 || got.SwapOutBps != 4096 {
		t.Fatalf("swap = total %d io %d/%d", got.SwapTotalKiB, got.SwapInBps, got.SwapOutBps)
	}
}

func assertParsedMemory(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.RAMAvailableMiB != 13512 || got.RAMCacheMiB != 3120 || got.RAMFreeMiB != 12041 || got.RAMBuffersMiB != 288 || got.RAMReclaimableMiB != 601 || got.RAMSharedMiB != 92 {
		t.Fatalf("ram expanded = %#v", got)
	}
}

func assertParsedPressure(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.CPUPressureSomeAvg10 != 2.43 || got.CPUPressureFullAvg10 != 0.14 || got.MemPressureSomeAvg10 != 1.20 || got.MemPressureFullAvg10 != 0.04 {
		t.Fatalf("pressure = %#v", got)
	}
}

func assertParsedNetworkAndStorage(t *testing.T, got *core.Sample) {
	t.Helper()

	assertParsedNetwork(t, got)
	assertParsedFilesystems(t, got)
	assertParsedTCP(t, got)
	assertParsedDisk(t, got)
}

func assertParsedNetwork(t *testing.T, got *core.Sample) {
	t.Helper()

	if len(got.Net) != 2 || got.Net[0].Iface != testIfaceEth0 {
		t.Fatalf("net = %#v", got.Net)
	}
	if got.Net[0].SpeedMbps != 1000 {
		t.Fatalf("net speed = %#v", got.Net[0])
	}
	if got.Net[0].RXPps != 1024 || got.Net[0].TXPps != 512 || got.Net[0].RXDrops != 2 || got.Net[0].RXErrors != 1 || got.Net[0].RXOverruns != 3 || got.Net[0].TXOverruns != 1 {
		t.Fatalf("net drops/errors = %#v", got.Net[0])
	}
}

func assertParsedFilesystems(t *testing.T, got *core.Sample) {
	t.Helper()

	if len(got.Filesystems) != 2 || got.Filesystems[1].Mount != testDataMount || got.Filesystems[1].InodesUsedPercent != 11 {
		t.Fatalf("filesystems = %#v", got.Filesystems)
	}
}

func assertParsedTCP(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.TCPRetransSegsPerSec != 9 || got.TCPResetsPerSec != 1 {
		t.Fatalf("tcp health = %d/%d", got.TCPRetransSegsPerSec, got.TCPResetsPerSec)
	}
}

func assertParsedDisk(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.DiskAwaitMS != 1.37 || got.DiskQueueDepth != 0.21 || got.DiskReadMergedPerSec != 12 || got.DiskWriteMergedPerSec != 7 || got.DiskInflight != 3 {
		t.Fatalf("disk expanded = %#v", got)
	}
}

func assertParsedProcesses(t *testing.T, got *core.Sample) {
	t.Helper()

	if len(got.CPUCoresUsage) != 1 || got.CPUCoresUsage[0].Index != 0 {
		t.Fatalf("cpu cores usage = %#v", got.CPUCoresUsage)
	}
	if len(got.TopProcesses) != 1 || got.TopProcesses[0].PID != 4242 || got.TopProcesses[0].Command != testPythonCommand {
		t.Fatalf("top processes = %#v", got.TopProcesses)
	}
	if len(got.GPUProcesses) != 1 || got.GPUProcesses[0].GPUUUID != testGPUUUID || got.GPUProcesses[0].UsedMemMiB != 3072 {
		t.Fatalf("gpu processes = %#v", got.GPUProcesses)
	}
}

func assertParsedGPU(t *testing.T, got *core.Sample) {
	t.Helper()

	assertParsedGPUIdentity(t, got)
	assertParsedGPUUtilization(t, got)
	assertParsedGPUPCIe(t, got)
	assertParsedGPUClocks(t, got)
}

func assertParsedGPUIdentity(t *testing.T, got *core.Sample) {
	t.Helper()

	if len(got.GPUs) != 1 || got.GPUs[0].Name != "NVIDIA GeForce RTX 3060 | Quiet" {
		t.Fatalf("gpus = %#v", got.GPUs)
	}
	if got.GPUs[0].UUID != testGPUUUID {
		t.Fatalf("gpu uuid = %#v", got.GPUs[0])
	}
}

func assertParsedGPUUtilization(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.GPUs[0].EncoderUtil != 12 || got.GPUs[0].DecoderUtil != 8 || got.GPUs[0].GraphicsClock != 1740 || got.GPUs[0].VideoClock != 1620 {
		t.Fatalf("gpu extra clocks/util = %#v", got.GPUs[0])
	}
}

func assertParsedGPUPCIe(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.GPUs[0].PCIeGenCurrent != 3 || got.GPUs[0].PCIeGenMax != 4 || got.GPUs[0].PCIeWidthCurrent != 8 || got.GPUs[0].PCIeWidthMax != 16 || got.GPUs[0].ThrottleReasons != "power cap" {
		t.Fatalf("gpu pcie/throttle = %#v", got.GPUs[0])
	}
}

func assertParsedGPUClocks(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.GPUs[0].MaxSMClock != 2100 || got.GPUs[0].MaxMemClock != 7501 {
		t.Fatalf("gpu max clocks = %#v", got.GPUs[0])
	}
}

func TestParserRejectsWrongProtocolVersion(t *testing.T) {
	t.Parallel()

	var p parser.Parser
	if got, ok := p.HandleLine(`{"version":2}`); ok || got != nil {
		t.Fatalf("expected version mismatch to be rejected: %#v %v", got, ok)
	}
}

func TestParserReportsLastRejectedSampleLine(t *testing.T) {
	t.Parallel()

	var p parser.Parser
	if got, ok := p.HandleLine(`{not-json`); ok || got != nil {
		t.Fatalf("expected invalid JSON to be rejected: %#v %v", got, ok)
	}
	err := p.LastError()
	if err == nil {
		t.Fatal("expected parser rejection error")
	}
	if !strings.Contains(err.Error(), "parse sample JSON") {
		t.Fatalf("parser error = %q", err.Error())
	}

	if got, ok := p.HandleLine(`{"version":1}`); !ok || got == nil {
		t.Fatalf("expected valid sample to clear parser error: %#v %v", got, ok)
	}
	if err := p.LastError(); err != nil {
		t.Fatalf("parser error after valid sample = %v", err)
	}
}

func TestParserBuildsDiskArrayFromSamplerJSON(t *testing.T) {
	t.Parallel()

	var p parser.Parser
	line := `{"version":1,"disk":{"root_source":"/dev/sda1","root_used_kib":42000000,"root_total_kib":100000000,"root_used_percent":42,"device":"sda","read_bps":1024,"write_bps":2048,"read_merged_per_sec":1,"write_merged_per_sec":2,"util_percent":3,"await_ms":4.5,"queue_depth":0.25,"inflight":1},"disks":[{"device":"sda","read_bps":1024,"write_bps":2048,"read_merged_per_sec":1,"write_merged_per_sec":2,"util_percent":3,"await_ms":4.5,"queue_depth":0.25,"inflight":1},{"device":"nvme0n1","read_bps":4096,"write_bps":8192,"read_merged_per_sec":3,"write_merged_per_sec":4,"util_percent":5,"await_ms":1.25,"queue_depth":0.75,"inflight":2}]}`

	got, ok := p.HandleLine(line)
	if !ok || got == nil {
		t.Fatalf("expected completed multi-disk sample from Parser")
	}
	if got.DiskDevice != "sda" || got.DiskReadBps != 1024 {
		t.Fatalf("legacy root disk fields = %#v", got)
	}
	if len(got.Disks) != 2 {
		t.Fatalf("disks = %#v", got.Disks)
	}
	if got.Disks[1].Device != "nvme0n1" || got.Disks[1].ReadBps != 4096 || got.Disks[1].QueueDepth != 0.75 {
		t.Fatalf("second disk = %#v", got.Disks[1])
	}
}

func TestParserBuildsSampleWithEscapedControlCharacters(t *testing.T) {
	t.Parallel()

	var p parser.Parser
	line := `{"version":1,"cpu_name":"AMD\u001b Ryzen\u0001","top_processes":[{"pid":4242,"command":"worker\u001f job","cpu_percent":12,"rss_mib":64}]}`

	got, ok := p.HandleLine(line)
	if !ok || got == nil {
		t.Fatalf("expected sample with escaped control characters to parse")
	}
	if got.CPUName != "AMD\x1b Ryzen\x01" {
		t.Fatalf("cpu name = %q", got.CPUName)
	}
	if len(got.TopProcesses) != 1 || got.TopProcesses[0].Command != "worker\x1f job" {
		t.Fatalf("top processes = %#v", got.TopProcesses)
	}
	if err := p.LastError(); err != nil {
		t.Fatalf("parser error after escaped control characters = %v", err)
	}
}

func TestParserBuildsIntelGPUFromSamplerJSON(t *testing.T) {
	t.Parallel()

	var p parser.Parser
	line := `{"version":1,"remote":"intel-host","gpus":[{"index":0,"uuid":"intel-0000:03:00.0","name":"Intel GPU 8086:56A5","util_percent":61,"mem_util_percent":38,"encoder_util_percent":-1,"decoder_util_percent":7,"mem_used_mib":3072,"mem_total_mib":8192,"temp_c":53,"power_draw_w":14.75,"power_limit_w":45.0,"fan_percent":-1,"sm_clock_mhz":1016,"sm_clock_max_mhz":1300,"mem_clock_mhz":-1,"mem_clock_max_mhz":-1,"graphics_clock_mhz":1016,"video_clock_mhz":-1,"pcie_gen_current":-1,"pcie_gen_max":-1,"pcie_width_current":-1,"pcie_width_max":-1,"throttle_reasons":"","p_state":""}]}`

	got, ok := p.HandleLine(line)
	if !ok || got == nil {
		t.Fatalf("expected completed Intel GPU sample from Parser")
	}
	if len(got.GPUs) != 1 {
		t.Fatalf("expected one Intel GPU, got %#v", got.GPUs)
	}
	gpu := got.GPUs[0]
	if gpu.UUID != "intel-0000:03:00.0" || gpu.Name != "Intel GPU 8086:56A5" || gpu.Util != 61 {
		t.Fatalf("unexpected Intel GPU identity/utilization: %#v", gpu)
	}
	if gpu.MemUsed != 3072 || gpu.MemTotal != 8192 || gpu.MemUtil != 38 {
		t.Fatalf("unexpected Intel GPU memory: %#v", gpu)
	}
	if gpu.Temp != 53 || gpu.PowerDraw != 14.75 || gpu.SMClock != 1016 || gpu.GraphicsClock != 1016 {
		t.Fatalf("unexpected Intel GPU telemetry: %#v", gpu)
	}
	if gpu.EncoderUtil != -1 || gpu.Fan != -1 || gpu.PState != "" {
		t.Fatalf("expected unavailable Intel vendor details to use sentinels, got %#v", gpu)
	}
}

func TestParserBuildsAMDGPUFromSamplerJSON(t *testing.T) {
	t.Parallel()

	var p parser.Parser
	line := `{"version":1,"remote":"amd-host","gpus":[{"index":0,"uuid":"amd-0000:0b:00.0","name":"AMD Radeon RX 7900 XTX","util_percent":73,"mem_util_percent":50,"encoder_util_percent":-1,"decoder_util_percent":12,"mem_used_mib":12288,"mem_total_mib":24576,"temp_c":62,"power_draw_w":315.5,"power_limit_w":355.0,"fan_percent":58,"sm_clock_mhz":2485,"sm_clock_max_mhz":2900,"mem_clock_mhz":1248,"mem_clock_max_mhz":1250,"graphics_clock_mhz":2485,"video_clock_mhz":-1,"pcie_gen_current":4,"pcie_gen_max":4,"pcie_width_current":16,"pcie_width_max":16,"throttle_reasons":"power cap","p_state":"auto"}]}`

	got, ok := p.HandleLine(line)
	if !ok || got == nil {
		t.Fatalf("expected completed AMD GPU sample from Parser")
	}
	if len(got.GPUs) != 1 {
		t.Fatalf("expected one AMD GPU, got %#v", got.GPUs)
	}
	assertParsedAMDGPUIdentity(t, got.GPUs[0])
	assertParsedAMDGPUUtilization(t, got.GPUs[0])
	assertParsedAMDGPUClocks(t, got.GPUs[0])
	assertParsedAMDGPUState(t, got.GPUs[0])
}

func assertParsedAMDGPUIdentity(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	if gpu.UUID != "amd-0000:0b:00.0" || gpu.Name != "AMD Radeon RX 7900 XTX" || gpu.Util != 73 {
		t.Fatalf("unexpected AMD GPU identity/utilization: %#v", gpu)
	}
}

func assertParsedAMDGPUUtilization(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	if gpu.MemUsed != 12288 || gpu.MemTotal != 24576 || gpu.MemUtil != 50 {
		t.Fatalf("unexpected AMD GPU memory: %#v", gpu)
	}
	if gpu.Temp != 62 || gpu.PowerDraw != 315.5 || gpu.PowerLimit != 355 || gpu.Fan != 58 {
		t.Fatalf("unexpected AMD GPU sensors: %#v", gpu)
	}
}

func assertParsedAMDGPUClocks(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	if gpu.SMClock != 2485 || gpu.MaxSMClock != 2900 || gpu.MemClock != 1248 || gpu.MaxMemClock != 1250 || gpu.GraphicsClock != 2485 {
		t.Fatalf("unexpected AMD GPU clocks: %#v", gpu)
	}
	if gpu.PCIeGenCurrent != 4 || gpu.PCIeGenMax != 4 || gpu.PCIeWidthCurrent != 16 || gpu.PCIeWidthMax != 16 {
		t.Fatalf("unexpected AMD GPU PCIe fields: %#v", gpu)
	}
}

func assertParsedAMDGPUState(t *testing.T, gpu core.GPUStat) {
	t.Helper()

	if gpu.ThrottleReasons != "power cap" || gpu.PState != "auto" || gpu.EncoderUtil != -1 || gpu.DecoderUtil != 12 {
		t.Fatalf("unexpected AMD GPU state/media fields: %#v", gpu)
	}
}

func TestParserBuildsPowerMetricsFromSamplerJSON(t *testing.T) {
	t.Parallel()

	var p parser.Parser
	line := `{"version":1,"remote":"power-host","power":{"external_power_online":1,"battery_percent":83,"battery_status":"Discharging","power_draw_w":12.34,"ups_present":1,"source_name":"BAT0","supplies":[{"name":"AC0","type":"Mains","online":1,"capacity_percent":-1,"status":"","power_draw_w":-1,"present":-1},{"name":"BAT0","type":"Battery","online":-1,"capacity_percent":83,"status":"Discharging","power_draw_w":12.34,"present":1},{"name":"BAT1","type":"Battery","online":-1,"capacity_percent":91,"status":"Charging","power_draw_w":-1,"present":1},{"name":"UPS0","type":"UPS","online":0,"capacity_percent":55,"status":"Full","power_draw_w":-1,"present":1}]}}`

	got, ok := p.HandleLine(line)
	if !ok || got == nil {
		t.Fatalf("expected completed power sample from Parser")
	}
	assertParsedPowerSummary(t, got)
	assertParsedPowerSupplies(t, got)
}

func assertParsedPowerSummary(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.ExternalPowerOnline != 1 || got.BatteryPercent != 83 || got.BatteryStatus != "Discharging" {
		t.Fatalf("unexpected power online/battery summary: %#v", got)
	}
	if got.PowerDrawWatts != 12.34 || got.UPSPresent != 1 || got.PowerSourceName != "BAT0" {
		t.Fatalf("unexpected power draw/source summary: %#v", got)
	}
}

func assertParsedPowerSupplies(t *testing.T, got *core.Sample) {
	t.Helper()

	if len(got.PowerSupplies) != 4 {
		t.Fatalf("power supplies = %#v", got.PowerSupplies)
	}
	assertParsedPowerBattery(t, got.PowerSupplies[1])
	assertParsedPowerChargingBattery(t, got.PowerSupplies[2])
	assertParsedPowerUPS(t, got.PowerSupplies[3])
}

func assertParsedPowerBattery(t *testing.T, battery core.PowerSupplyStat) {
	t.Helper()

	if battery.Name != "BAT0" || battery.Type != "Battery" || battery.CapacityPercent != 83 || battery.Status != "Discharging" || battery.PowerDrawWatts != 12.34 || battery.Present != 1 {
		t.Fatalf("unexpected battery supply: %#v", battery)
	}
}

func assertParsedPowerChargingBattery(t *testing.T, chargingBattery core.PowerSupplyStat) {
	t.Helper()

	if chargingBattery.Name != "BAT1" || chargingBattery.Status != "Charging" || chargingBattery.CapacityPercent != 91 {
		t.Fatalf("unexpected charging battery supply: %#v", chargingBattery)
	}
}

func assertParsedPowerUPS(t *testing.T, ups core.PowerSupplyStat) {
	t.Helper()

	if ups.Name != "UPS0" || ups.Type != "UPS" || ups.CapacityPercent != 55 || ups.Status != "Full" || ups.Present != 1 {
		t.Fatalf("unexpected UPS supply: %#v", ups)
	}
}

func TestParserPreservesPowerSentinelsForUnavailableFields(t *testing.T) {
	t.Parallel()

	var p parser.Parser
	line := `{"version":1,"remote":"partial-power-host","power":{"external_power_online":-1,"battery_percent":-1,"battery_status":"","power_draw_w":-1,"ups_present":0,"source_name":"","supplies":[{"name":"BAT1","type":"Battery","online":-1,"capacity_percent":-1,"status":"","power_draw_w":-1,"present":-1}]}}`

	got, ok := p.HandleLine(line)
	if !ok || got == nil {
		t.Fatalf("expected completed partial power sample from Parser")
	}
	if got.ExternalPowerOnline != -1 || got.BatteryPercent != -1 || got.BatteryStatus != "" || got.PowerDrawWatts != -1 || got.UPSPresent != 0 || got.PowerSourceName != "" {
		t.Fatalf("unexpected unavailable power summary: %#v", got)
	}
	if len(got.PowerSupplies) != 1 {
		t.Fatalf("power supplies = %#v", got.PowerSupplies)
	}
	supply := got.PowerSupplies[0]
	if supply.Name != "BAT1" || supply.Online != -1 || supply.CapacityPercent != -1 || supply.Status != "" || supply.PowerDrawWatts != -1 || supply.Present != -1 {
		t.Fatalf("unexpected unavailable supply sentinels: %#v", supply)
	}
}

func TestParserBuildsSamplesFromDegradedHostFixtures(t *testing.T) {
	t.Parallel()

	fixtures := []struct {
		name   string
		assert func(t *testing.T, got *core.Sample)
	}{
		{
			name:   "missing-optional-tools.json",
			assert: assertMissingOptionalToolsFixture,
		},
		{
			name:   "restricted-proc-sys.json",
			assert: assertRestrictedProcSysFixture,
		},
		{
			name:   "partial-sections-wsl.json",
			assert: assertPartialSectionsWSLFixture,
		},
	}

	for _, fixture := range fixtures {
		t.Run(strings.TrimSuffix(fixture.name, ".json"), func(t *testing.T) {
			t.Parallel()

			line := readDegradedFixture(t, fixture.name)
			var p parser.Parser
			got, ok := p.HandleLine(line)
			if !ok || got == nil {
				t.Fatalf("expected degraded fixture %s to parse into core.Sample; parser error = %v", fixture.name, p.LastError())
			}
			fixture.assert(t, got)
		})
	}
}

func assertMissingOptionalToolsFixture(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.RemoteName != "degraded-missing-tools" {
		t.Fatalf("remote name = %q", got.RemoteName)
	}
	if len(got.GPUs) != 0 || len(got.GPUProcesses) != 0 {
		t.Fatalf("expected absent GPU tools to leave GPU slices empty, got gpus=%#v gpuProcesses=%#v", got.GPUs, got.GPUProcesses)
	}
	if got.CPUTempC != -1 || got.CPUPressureSomeAvg10 != -1 || got.MemPressureFullAvg10 != -1 {
		t.Fatalf("expected unavailable optional metrics to use sentinels, got temp=%d cpuPressure=%f memPressure=%f", got.CPUTempC, got.CPUPressureSomeAvg10, got.MemPressureFullAvg10)
	}
	if got.ExternalPowerOnline != -1 || got.BatteryPercent != -1 || got.PowerDrawWatts != -1 || got.UPSPresent != 0 || len(got.PowerSupplies) != 0 {
		t.Fatalf("expected missing power supplies to use sampler sentinels, got %#v", got)
	}
}

func assertRestrictedProcSysFixture(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.RemoteName != "degraded-restricted-data" {
		t.Fatalf("remote name = %q", got.RemoteName)
	}
	if got.RAMAvailableMiB != -1 || got.RAMCacheMiB != -1 || got.SwapTotalKiB != -1 {
		t.Fatalf("expected restricted memory/proc fields to use sentinels, got ramAvailable=%d ramCache=%d swapTotal=%d", got.RAMAvailableMiB, got.RAMCacheMiB, got.SwapTotalKiB)
	}
	if got.RootSource != "" || got.RootUsedKiB != -1 || got.DiskReadBps != -1 {
		t.Fatalf("expected restricted storage fields to use empty/sentinel values, got root=%q used=%d readBps=%d", got.RootSource, got.RootUsedKiB, got.DiskReadBps)
	}
	if got.ExternalPowerOnline != -1 || got.BatteryPercent != -1 || got.PowerDrawWatts != -1 {
		t.Fatalf("expected restricted power sysfs sentinels, got %#v", got)
	}
}

func assertPartialSectionsWSLFixture(t *testing.T, got *core.Sample) {
	t.Helper()

	if got.RemoteName != "degraded-partial-wsl" {
		t.Fatalf("remote name = %q", got.RemoteName)
	}
	if len(got.Net) != 1 || got.Net[0].Iface != testIfaceEth0 || got.Net[0].SpeedMbps != -1 || got.Net[0].RXPps != -1 {
		t.Fatalf("expected partial network data with sentinels, got %#v", got.Net)
	}
	if len(got.Filesystems) != 1 || got.Filesystems[0].Mount != "/" || got.Filesystems[0].InodesUsedPercent != -1 {
		t.Fatalf("expected one partial filesystem row, got %#v", got.Filesystems)
	}
	if len(got.TopProcesses) != 0 {
		t.Fatalf("expected malformed process row to be omitted, got %#v", got.TopProcesses)
	}
	if got.CPUName != "WSL guest CPU" || got.CPUFreqMHz != -1 || got.CPUMaxFreqMHz != -1 {
		t.Fatalf("expected unavailable WSL host metrics to preserve Linux fallback sentinels, got name=%q freq=%d max=%d", got.CPUName, got.CPUFreqMHz, got.CPUMaxFreqMHz)
	}
}

func readDegradedFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("testdata", "degraded", name)
	// #nosec G304 -- fixture names are hard-coded in TestParserBuildsSamplesFromDegradedHostFixtures.
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read degraded fixture %s: %v", path, err)
	}

	return strings.TrimSpace(string(contents))
}
