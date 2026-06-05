package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/output"
)

func TestWriterEmitsNormalizedSampleJSONL(t *testing.T) {
	t.Parallel()

	smp := populatedSample()
	var buf bytes.Buffer

	if err := output.NewWriter(&buf).WriteSample(smp); err != nil {
		t.Fatalf("WriteSample returned error: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatalf("JSONL output missing trailing newline: %q", buf.String())
	}

	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &got); err != nil {
		t.Fatalf("JSONL output is not valid JSON: %v", err)
	}

	assertStringField(t, got, "schema", output.JSONLSchema)
	assertNumberField(t, got, "remote_epoch", float64(smp.RemoteEpoch))
	assertStringField(t, got, "remote_timestamp", smp.RemoteTimestamp)
	assertStringField(t, got, "remote_name", smp.RemoteName)
	assertNumberField(t, got, "ram_cache_mib", float64(smp.RAMCacheMiB))
	assertNumberField(t, got, "disk_read_merged_per_sec", float64(smp.DiskReadMergedPerSec))
	assertNumberField(t, got, "tcp_retrans_segs_per_sec", float64(smp.TCPRetransSegsPerSec))
	assertStringField(t, got, "received_at", smp.ReceivedAt.UTC().Format(time.RFC3339Nano))
	assertNoField(t, got, "RemoteName")

	net := assertArrayField(t, got, "net")
	assertStringField(t, firstObject(t, net), "iface", "eth0")
	assertNumberField(t, firstObject(t, net), "rx_drops", float64(2))

	filesystems := assertArrayField(t, got, "filesystems")
	assertStringField(t, firstObject(t, filesystems), "mount", "/mnt/data")

	cores := assertArrayField(t, got, "cpu_core_usage")
	assertNumberField(t, firstObject(t, cores), "percent", float64(77))

	processes := assertArrayField(t, got, "top_processes")
	assertStringField(t, firstObject(t, processes), "command", "python")

	gpuProcesses := assertArrayField(t, got, "gpu_processes")
	assertNumberField(t, firstObject(t, gpuProcesses), "used_mem_mib", float64(3072))

	gpus := assertArrayField(t, got, "gpus")
	assertNumberField(t, firstObject(t, gpus), "power_draw_w", 103.12)

	assertNumberField(t, got, "external_power_online", float64(1))
	assertNumberField(t, got, "battery_percent", float64(83))
	assertStringField(t, got, "battery_status", "Discharging")
	assertNumberField(t, got, "power_draw_w", 12.34)
	assertNumberField(t, got, "ups_present", float64(1))
	assertStringField(t, got, "power_source_name", "BAT0")
	powerSupplies := assertArrayField(t, got, "power_supplies")
	assertStringField(t, firstObject(t, powerSupplies), "name", "BAT0")
	assertNumberField(t, firstObject(t, powerSupplies), "capacity_percent", float64(83))
}

func TestWriterNormalizesNilSlicesToEmptyArrays(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := output.NewWriter(&buf).WriteSample(core.EmptySample()); err != nil {
		t.Fatalf("WriteSample returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &got); err != nil {
		t.Fatalf("JSONL output is not valid JSON: %v", err)
	}

	for _, field := range []string{"net", "filesystems", "cpu_core_usage", "top_processes", "gpu_processes", "gpus", "power_supplies"} {
		values := assertArrayField(t, got, field)
		if len(values) != 0 {
			t.Fatalf("%s = %#v, want empty array", field, values)
		}
	}
}

func populatedSample() core.Sample {
	smp := core.EmptySample()
	smp.RemoteEpoch = 1716912345
	smp.RemoteTimestamp = "2026-05-28 19:35:45"
	smp.RemoteName = "gpu-box"
	smp.UptimeSeconds = 14340
	smp.Load1 = 10.55
	smp.Load5 = 4.82
	smp.Load15 = 4.09
	smp.CPUCores = 12
	smp.CPUName = "AMD Ryzen 5 5600X"
	smp.CPUPercent = 99
	smp.CPUUserPercent = 71
	smp.CPUSystemPercent = 19
	smp.CPUIOWaitPercent = 6
	smp.CPUStealPercent = 1
	smp.RAMUsedMiB = 2455
	smp.RAMTotalMiB = 15967
	smp.RAMAvailableMiB = 13512
	smp.RAMFreeMiB = 12041
	smp.RAMCacheMiB = 3120
	smp.RAMBuffersMiB = 288
	smp.RAMReclaimableMiB = 601
	smp.RAMSharedMiB = 92
	smp.CPUFreqMHz = 3680
	smp.CPUMaxFreqMHz = 4700
	smp.CPUTempC = 66
	smp.CPUPressureSomeAvg10 = 2.43
	smp.CPUPressureFullAvg10 = 0.14
	smp.MemPressureSomeAvg10 = 1.20
	smp.MemPressureFullAvg10 = 0.04
	smp.SwapFreeKiB = 0
	smp.SwapTotalKiB = 4194304
	smp.SwapInBps = 8192
	smp.SwapOutBps = 4096
	smp.RootSource = "/dev/sdd"
	smp.RootUsedKiB = 42000000
	smp.RootTotalKiB = 100000000
	smp.RootUsedPercent = 42
	smp.DiskDevice = "sdd"
	smp.DiskReadBps = 1048576
	smp.DiskWriteBps = 524288
	smp.DiskReadMergedPerSec = 12
	smp.DiskWriteMergedPerSec = 7
	smp.DiskUtil = 12
	smp.DiskAwaitMS = 1.37
	smp.DiskQueueDepth = 0.21
	smp.DiskInflight = 3
	smp.TCPRetransSegsPerSec = 9
	smp.TCPResetsPerSec = 1
	smp.Net = []core.NetStat{populatedNetStat()}
	smp.Filesystems = []core.FilesystemStat{populatedFilesystemStat()}
	smp.CPUCoresUsage = []core.CPUCore{populatedCPUCore()}
	smp.TopProcesses = []core.ProcessStat{populatedProcessStat()}
	smp.GPUProcesses = []core.GPUProcessStat{populatedGPUProcessStat()}
	smp.GPUs = []core.GPUStat{populatedGPUStat()}
	smp.ExternalPowerOnline = 1
	smp.BatteryPercent = 83
	smp.BatteryStatus = "Discharging"
	smp.PowerDrawWatts = 12.34
	smp.UPSPresent = 1
	smp.PowerSourceName = "BAT0"
	smp.PowerSupplies = []core.PowerSupplyStat{populatedPowerSupplyStat()}
	smp.ReceivedAt = time.Unix(1716912346, 123).UTC()

	return smp
}

func populatedNetStat() core.NetStat {
	return core.NetStat{
		Iface:      "eth0",
		RXBps:      125000,
		TXBps:      24000,
		RXPps:      1024,
		TXPps:      512,
		SpeedMbps:  1000,
		RXDrops:    2,
		RXErrors:   1,
		RXOverruns: 3,
		TXDrops:    0,
		TXErrors:   0,
		TXOverruns: 1,
	}
}

func populatedFilesystemStat() core.FilesystemStat {
	return core.FilesystemStat{
		Source:            "/dev/sdc",
		Mount:             "/mnt/data",
		UsedKiB:           8000000,
		TotalKiB:          20000000,
		UsedPercent:       40,
		InodesUsedPercent: 11,
	}
}

func populatedCPUCore() core.CPUCore {
	return core.CPUCore{
		Index:   0,
		Percent: 77,
	}
}

func populatedProcessStat() core.ProcessStat {
	return core.ProcessStat{
		PID:        4242,
		Command:    "python",
		CPUPercent: 88,
		RSSMiB:     2048,
	}
}

func populatedGPUProcessStat() core.GPUProcessStat {
	return core.GPUProcessStat{
		GPUUUID:    "GPU-123",
		PID:        4242,
		Command:    "python",
		UsedMemMiB: 3072,
	}
}

func populatedGPUStat() core.GPUStat {
	return core.GPUStat{
		Index:            0,
		UUID:             "GPU-123",
		Name:             "NVIDIA GeForce RTX 3060",
		Util:             82,
		MemUtil:          34,
		EncoderUtil:      12,
		DecoderUtil:      8,
		MemUsed:          2003,
		MemTotal:         12288,
		Temp:             55,
		PowerDraw:        103.12,
		PowerLimit:       170,
		Fan:              44,
		SMClock:          210,
		MaxSMClock:       2100,
		MemClock:         810,
		MaxMemClock:      7501,
		GraphicsClock:    1740,
		VideoClock:       1620,
		PCIeGenCurrent:   3,
		PCIeGenMax:       4,
		PCIeWidthCurrent: 8,
		PCIeWidthMax:     16,
		ThrottleReasons:  "power cap",
		PState:           "P5",
	}
}

func populatedPowerSupplyStat() core.PowerSupplyStat {
	return core.PowerSupplyStat{
		Name:            "BAT0",
		Type:            "Battery",
		Online:          -1,
		CapacityPercent: 83,
		Status:          "Discharging",
		PowerDrawWatts:  12.34,
		Present:         1,
	}
}

func assertStringField(t *testing.T, object map[string]any, field, want string) {
	t.Helper()

	got, ok := object[field].(string)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %q", field, object[field], want)
	}
}

func assertNumberField(t *testing.T, object map[string]any, field string, want float64) {
	t.Helper()

	got, ok := object[field].(float64)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %v", field, object[field], want)
	}
}

func assertNoField(t *testing.T, object map[string]any, field string) {
	t.Helper()

	if _, ok := object[field]; ok {
		t.Fatalf("unexpected field %q in %#v", field, object)
	}
}

func assertArrayField(t *testing.T, object map[string]any, field string) []any {
	t.Helper()

	got, ok := object[field].([]any)
	if !ok {
		t.Fatalf("%s = %#v, want JSON array", field, object[field])
	}

	return got
}

func firstObject(t *testing.T, values []any) map[string]any {
	t.Helper()

	if len(values) == 0 {
		t.Fatal("array is empty, want at least one JSON object")
	}
	got, ok := values[0].(map[string]any)
	if !ok {
		t.Fatalf("array[0] = %#v, want JSON object", values[0])
	}

	return got
}
