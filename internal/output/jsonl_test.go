package output_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/output"
	"github.com/lmliam/remote-monitor/internal/parser"
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

	disks := assertArrayField(t, got, "disks")
	assertStringField(t, firstObject(t, disks), "device", "nvme0n1")
	assertNumberField(t, firstObject(t, disks), "queue_depth", 0.75)

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

	for _, field := range []string{"net", "filesystems", "disks", "cpu_core_usage", "top_processes", "gpu_processes", "gpus", "power_supplies"} {
		values := assertArrayField(t, got, field)
		if len(values) != 0 {
			t.Fatalf("%s = %#v, want empty array", field, values)
		}
	}
}

func TestWriterPreservesParsedWireSampleFields(t *testing.T) {
	t.Parallel()

	var prs parser.Parser
	parsed, ok := prs.HandleLine(fullWireSampleLine())
	if !ok || parsed == nil {
		t.Fatalf("expected parsed sample from wire JSON: %v", prs.LastError())
	}
	parsed.ReceivedAt = time.Unix(1716912346, 123).UTC()

	var buf bytes.Buffer
	if err := output.NewWriter(&buf).WriteSample(*parsed); err != nil {
		t.Fatalf("WriteSample returned error: %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &got); err != nil {
		t.Fatalf("JSONL output is not valid JSON: %v", err)
	}

	assertRawJSONField(t, got, "schema", output.JSONLSchema)
	assertRawJSONField(t, got, "received_at", parsed.ReceivedAt.UTC().Format(time.RFC3339Nano))
	assertSampleJSONFields(t, *parsed, got)
}

func fullWireSampleLine() string {
	return `{"version":1,"epoch":1716912345,"timestamp":"2026-05-28 19:35:45","remote":"DESKTOP","uptime_seconds":14340,"load1":10.55,"load5":4.82,"load15":4.09,"cpu_cores":12,"cpu_name":"AMD Ryzen 5 5600X 6-Core Processor","cpu_percent":99,"cpu_user_percent":71,"cpu_system_percent":19,"cpu_iowait_percent":6,"cpu_steal_percent":1,"ram_used_mib":2455,"ram_total_mib":15967,"ram_available_mib":13512,"ram_free_mib":12041,"ram_cache_mib":3120,"ram_buffers_mib":288,"ram_reclaimable_mib":601,"ram_shared_mib":92,"cpu_freq_mhz":3680,"cpu_max_freq_mhz":4700,"cpu_temp_c":66,"cpu_pressure_some_avg10":2.43,"cpu_pressure_full_avg10":0.14,"mem_pressure_some_avg10":1.2,"mem_pressure_full_avg10":0.04,"swap":{"free_kib":0,"total_kib":4194304,"in_bps":8192,"out_bps":4096},"disk":{"root_source":"/dev/sdd","root_used_kib":42000000,"root_total_kib":100000000,"root_used_percent":42,"device":"sdd","read_bps":1048576,"write_bps":524288,"read_merged_per_sec":12,"write_merged_per_sec":7,"util_percent":12,"await_ms":1.37,"queue_depth":0.21,"inflight":3},"net":[{"iface":"eth0","rx_bps":125000,"tx_bps":24000,"rx_pps":1024,"tx_pps":512,"speed_mbps":1000,"rx_drops":2,"rx_errors":1,"rx_overruns":3,"tx_drops":0,"tx_errors":0,"tx_overruns":1}],"filesystems":[{"source":"/dev/sdc","mount":"/mnt/data","used_kib":8000000,"total_kib":20000000,"used_percent":40,"inodes_used_percent":11}],"disks":[{"device":"nvme0n1","read_bps":4096,"write_bps":8192,"read_merged_per_sec":3,"write_merged_per_sec":4,"util_percent":5,"await_ms":1.25,"queue_depth":0.75,"inflight":2}],"tcp_retrans_segs_per_sec":9,"tcp_resets_per_sec":1,"cpu_core_usage":[{"index":0,"percent":77}],"top_processes":[{"pid":4242,"command":"python","cpu_percent":88,"rss_mib":2048}],"gpu_processes":[{"gpu_uuid":"GPU-123","pid":4242,"command":"python","used_mem_mib":3072}],"gpus":[{"index":0,"uuid":"GPU-123","name":"NVIDIA GeForce RTX 3060","util_percent":82,"mem_util_percent":34,"encoder_util_percent":12,"decoder_util_percent":8,"mem_used_mib":2003,"mem_total_mib":12288,"temp_c":55,"power_draw_w":103.12,"power_limit_w":170.0,"fan_percent":44,"sm_clock_mhz":210,"sm_clock_max_mhz":2100,"mem_clock_mhz":810,"mem_clock_max_mhz":7501,"graphics_clock_mhz":1740,"video_clock_mhz":1620,"pcie_gen_current":3,"pcie_gen_max":4,"pcie_width_current":8,"pcie_width_max":16,"throttle_reasons":"power cap","p_state":"P5"}],"power":{"external_power_online":1,"battery_percent":83,"battery_status":"Discharging","power_draw_w":12.34,"ups_present":1,"source_name":"BAT0","supplies":[{"name":"BAT0","type":"Battery","online":-1,"capacity_percent":83,"status":"Discharging","power_draw_w":12.34,"present":1}]}}`
}

func assertSampleJSONFields(t *testing.T, smp core.Sample, got map[string]json.RawMessage) {
	t.Helper()

	expectedFields := map[string]struct{}{
		"schema":      {},
		"received_at": {},
	}
	sampleType := reflect.TypeFor[core.Sample]()
	sampleValue := reflect.ValueOf(smp)
	for fieldIndex := range sampleType.NumField() {
		field := sampleType.Field(fieldIndex)
		name := sampleJSONFieldName(t, field)
		if name == "-" {
			continue
		}
		expectedFields[name] = struct{}{}
		assertRawJSONField(t, got, name, sampleValue.Field(fieldIndex).Interface())
	}
	assertExactJSONFieldSet(t, got, expectedFields)
}

func sampleJSONFieldName(t *testing.T, field reflect.StructField) string {
	t.Helper()

	tag := field.Tag.Get("json")
	if tag == "" {
		t.Fatalf("core.Sample.%s is missing a JSON tag", field.Name)
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		t.Fatalf("core.Sample.%s has an empty JSON field name", field.Name)
	}
	if name == "-" && field.Name != "ReceivedAt" {
		t.Fatalf("core.Sample.%s is unexpectedly omitted from JSONL parity", field.Name)
	}

	return name
}

func assertExactJSONFieldSet(t *testing.T, got map[string]json.RawMessage, want map[string]struct{}) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("JSONL field count = %d, want %d; fields = %#v", len(got), len(want), got)
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Fatalf("JSONL field %q was not expected", name)
		}
	}
}

func assertRawJSONField(t *testing.T, got map[string]json.RawMessage, name string, want any) {
	t.Helper()

	raw, ok := got[name]
	if !ok {
		t.Fatalf("JSONL field %q is missing", name)
	}

	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal expected %q value: %v", name, err)
	}
	var gotValue any
	if err := json.Unmarshal(raw, &gotValue); err != nil {
		t.Fatalf("decode JSONL field %q: %v", name, err)
	}
	var wantValue any
	if err := json.Unmarshal(wantJSON, &wantValue); err != nil {
		t.Fatalf("decode expected field %q: %v", name, err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("JSONL field %q = %#v, want %#v", name, gotValue, wantValue)
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
	smp.Disks = []core.DiskStat{populatedDiskStat()}
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

func populatedDiskStat() core.DiskStat {
	return core.DiskStat{
		Device:            "nvme0n1",
		ReadBps:           4096,
		WriteBps:          8192,
		ReadMergedPerSec:  3,
		WriteMergedPerSec: 4,
		Util:              5,
		AwaitMS:           1.25,
		QueueDepth:        0.75,
		Inflight:          2,
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
