package parser_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/parser"

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
