// Package output contains machine-readable output encoders.
package output

import (
	"encoding/json"
	"io"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
)

// JSONLSchema identifies the first normalized local sample export schema.
const JSONLSchema = "remote-monitor.normalized_sample.v1"

// Writer emits one normalized sample JSON object per line.
type Writer struct {
	encoder *json.Encoder
}

type sample struct {
	Schema                string                `json:"schema"`
	RemoteEpoch           int64                 `json:"remote_epoch"`
	RemoteTimestamp       string                `json:"remote_timestamp"`
	RemoteName            string                `json:"remote_name"`
	UptimeSeconds         int64                 `json:"uptime_seconds"`
	Load1                 float64               `json:"load1"`
	Load5                 float64               `json:"load5"`
	Load15                float64               `json:"load15"`
	CPUCores              int                   `json:"cpu_cores"`
	CPUName               string                `json:"cpu_name"`
	CPUPercent            int                   `json:"cpu_percent"`
	CPUUserPercent        int                   `json:"cpu_user_percent"`
	CPUSystemPercent      int                   `json:"cpu_system_percent"`
	CPUIOWaitPercent      int                   `json:"cpu_iowait_percent"`
	CPUStealPercent       int                   `json:"cpu_steal_percent"`
	RAMUsedMiB            int64                 `json:"ram_used_mib"`
	RAMTotalMiB           int64                 `json:"ram_total_mib"`
	RAMAvailableMiB       int64                 `json:"ram_available_mib"`
	RAMFreeMiB            int64                 `json:"ram_free_mib"`
	RAMCacheMiB           int64                 `json:"ram_cache_mib"`
	RAMBuffersMiB         int64                 `json:"ram_buffers_mib"`
	RAMReclaimableMiB     int64                 `json:"ram_reclaimable_mib"`
	RAMSharedMiB          int64                 `json:"ram_shared_mib"`
	CPUFreqMHz            int                   `json:"cpu_freq_mhz"`
	CPUMaxFreqMHz         int                   `json:"cpu_max_freq_mhz"`
	CPUTempC              int                   `json:"cpu_temp_c"`
	CPUPressureSomeAvg10  float64               `json:"cpu_pressure_some_avg10"`
	CPUPressureFullAvg10  float64               `json:"cpu_pressure_full_avg10"`
	MemPressureSomeAvg10  float64               `json:"mem_pressure_some_avg10"`
	MemPressureFullAvg10  float64               `json:"mem_pressure_full_avg10"`
	SwapFreeKiB           int64                 `json:"swap_free_kib"`
	SwapTotalKiB          int64                 `json:"swap_total_kib"`
	SwapInBps             int64                 `json:"swap_in_bps"`
	SwapOutBps            int64                 `json:"swap_out_bps"`
	RootSource            string                `json:"root_source"`
	RootUsedKiB           int64                 `json:"root_used_kib"`
	RootTotalKiB          int64                 `json:"root_total_kib"`
	RootUsedPercent       int                   `json:"root_used_percent"`
	DiskDevice            string                `json:"disk_device"`
	DiskReadBps           int64                 `json:"disk_read_bps"`
	DiskWriteBps          int64                 `json:"disk_write_bps"`
	DiskReadMergedPerSec  int64                 `json:"disk_read_merged_per_sec"`
	DiskWriteMergedPerSec int64                 `json:"disk_write_merged_per_sec"`
	DiskUtil              int                   `json:"disk_util_percent"`
	DiskAwaitMS           float64               `json:"disk_await_ms"`
	DiskQueueDepth        float64               `json:"disk_queue_depth"`
	DiskInflight          int                   `json:"disk_inflight"`
	TCPRetransSegsPerSec  int64                 `json:"tcp_retrans_segs_per_sec"`
	TCPResetsPerSec       int64                 `json:"tcp_resets_per_sec"`
	Net                   []core.NetStat        `json:"net"`
	Filesystems           []core.FilesystemStat `json:"filesystems"`
	CPUCoresUsage         []core.CPUCore        `json:"cpu_core_usage"`
	TopProcesses          []core.ProcessStat    `json:"top_processes"`
	GPUProcesses          []core.GPUProcessStat `json:"gpu_processes"`
	GPUs                  []core.GPUStat        `json:"gpus"`
	ReceivedAt            string                `json:"received_at,omitempty"`
}

// NewWriter creates a JSONL writer around the provided destination.
func NewWriter(w io.Writer) *Writer {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	return &Writer{encoder: encoder}
}

// WriteSample writes a single JSON object followed by a newline.
func (w *Writer) WriteSample(smp core.Sample) error {
	return w.encoder.Encode(fromCoreSample(smp))
}

func fromCoreSample(smp core.Sample) sample {
	return sample{
		Schema:                JSONLSchema,
		RemoteEpoch:           smp.RemoteEpoch,
		RemoteTimestamp:       smp.RemoteTimestamp,
		RemoteName:            smp.RemoteName,
		UptimeSeconds:         smp.UptimeSeconds,
		Load1:                 smp.Load1,
		Load5:                 smp.Load5,
		Load15:                smp.Load15,
		CPUCores:              smp.CPUCores,
		CPUName:               smp.CPUName,
		CPUPercent:            smp.CPUPercent,
		CPUUserPercent:        smp.CPUUserPercent,
		CPUSystemPercent:      smp.CPUSystemPercent,
		CPUIOWaitPercent:      smp.CPUIOWaitPercent,
		CPUStealPercent:       smp.CPUStealPercent,
		RAMUsedMiB:            smp.RAMUsedMiB,
		RAMTotalMiB:           smp.RAMTotalMiB,
		RAMAvailableMiB:       smp.RAMAvailableMiB,
		RAMFreeMiB:            smp.RAMFreeMiB,
		RAMCacheMiB:           smp.RAMCacheMiB,
		RAMBuffersMiB:         smp.RAMBuffersMiB,
		RAMReclaimableMiB:     smp.RAMReclaimableMiB,
		RAMSharedMiB:          smp.RAMSharedMiB,
		CPUFreqMHz:            smp.CPUFreqMHz,
		CPUMaxFreqMHz:         smp.CPUMaxFreqMHz,
		CPUTempC:              smp.CPUTempC,
		CPUPressureSomeAvg10:  smp.CPUPressureSomeAvg10,
		CPUPressureFullAvg10:  smp.CPUPressureFullAvg10,
		MemPressureSomeAvg10:  smp.MemPressureSomeAvg10,
		MemPressureFullAvg10:  smp.MemPressureFullAvg10,
		SwapFreeKiB:           smp.SwapFreeKiB,
		SwapTotalKiB:          smp.SwapTotalKiB,
		SwapInBps:             smp.SwapInBps,
		SwapOutBps:            smp.SwapOutBps,
		RootSource:            smp.RootSource,
		RootUsedKiB:           smp.RootUsedKiB,
		RootTotalKiB:          smp.RootTotalKiB,
		RootUsedPercent:       smp.RootUsedPercent,
		DiskDevice:            smp.DiskDevice,
		DiskReadBps:           smp.DiskReadBps,
		DiskWriteBps:          smp.DiskWriteBps,
		DiskReadMergedPerSec:  smp.DiskReadMergedPerSec,
		DiskWriteMergedPerSec: smp.DiskWriteMergedPerSec,
		DiskUtil:              smp.DiskUtil,
		DiskAwaitMS:           smp.DiskAwaitMS,
		DiskQueueDepth:        smp.DiskQueueDepth,
		DiskInflight:          smp.DiskInflight,
		TCPRetransSegsPerSec:  smp.TCPRetransSegsPerSec,
		TCPResetsPerSec:       smp.TCPResetsPerSec,
		Net:                   nonNilSlice(smp.Net),
		Filesystems:           nonNilSlice(smp.Filesystems),
		CPUCoresUsage:         nonNilSlice(smp.CPUCoresUsage),
		TopProcesses:          nonNilSlice(smp.TopProcesses),
		GPUProcesses:          nonNilSlice(smp.GPUProcesses),
		GPUs:                  nonNilSlice(smp.GPUs),
		ReceivedAt:            formatTime(smp.ReceivedAt),
	}
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339Nano)
}
