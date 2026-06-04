package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"strings"
	"time"
)

const wireProtocolVersion = 1

var errUnsupportedSampleVersion = errors.New("unsupported sample version")

type wireSwap struct {
	FreeKiB  int64 `json:"free_kib"`
	TotalKiB int64 `json:"total_kib"`
	InBps    int64 `json:"in_bps"`
	OutBps   int64 `json:"out_bps"`
}

type wireDisk struct {
	RootSource        string  `json:"root_source"`
	RootUsedKiB       int64   `json:"root_used_kib"`
	RootTotalKiB      int64   `json:"root_total_kib"`
	RootUsedPercent   int     `json:"root_used_percent"`
	Device            string  `json:"device"`
	ReadBps           int64   `json:"read_bps"`
	WriteBps          int64   `json:"write_bps"`
	ReadMergedPerSec  int64   `json:"read_merged_per_sec"`
	WriteMergedPerSec int64   `json:"write_merged_per_sec"`
	UtilPercent       int     `json:"util_percent"`
	AwaitMS           float64 `json:"await_ms"`
	QueueDepth        float64 `json:"queue_depth"`
	Inflight          int     `json:"inflight"`
}

type wireSample struct {
	Version              int                   `json:"version"`
	RemoteEpoch          int64                 `json:"epoch"`
	RemoteTime           string                `json:"timestamp"`
	RemoteName           string                `json:"remote"`
	Uptime               int64                 `json:"uptime_seconds"`
	Load1                float64               `json:"load1"`
	Load5                float64               `json:"load5"`
	Load15               float64               `json:"load15"`
	CPUCores             int                   `json:"cpu_cores"`
	CPUName              string                `json:"cpu_name"`
	CPUPercent           int                   `json:"cpu_percent"`
	CPUUserPercent       int                   `json:"cpu_user_percent"`
	CPUSystemPercent     int                   `json:"cpu_system_percent"`
	CPUIOWaitPercent     int                   `json:"cpu_iowait_percent"`
	CPUStealPercent      int                   `json:"cpu_steal_percent"`
	RAMUsedMiB           int64                 `json:"ram_used_mib"`
	RAMTotalMiB          int64                 `json:"ram_total_mib"`
	RAMAvailableMiB      int64                 `json:"ram_available_mib"`
	RAMFreeMiB           int64                 `json:"ram_free_mib"`
	RAMCacheMiB          int64                 `json:"ram_cache_mib"`
	RAMBuffersMiB        int64                 `json:"ram_buffers_mib"`
	RAMReclaimableMiB    int64                 `json:"ram_reclaimable_mib"`
	RAMSharedMiB         int64                 `json:"ram_shared_mib"`
	CPUFreqMHz           int                   `json:"cpu_freq_mhz"`
	CPUMaxFreqMHz        int                   `json:"cpu_max_freq_mhz"`
	CPUTempC             int                   `json:"cpu_temp_c"`
	CPUPressureSome      float64               `json:"cpu_pressure_some_avg10"`
	CPUPressureFull      float64               `json:"cpu_pressure_full_avg10"`
	MemPressureSome      float64               `json:"mem_pressure_some_avg10"`
	MemPressureFull      float64               `json:"mem_pressure_full_avg10"`
	Swap                 wireSwap              `json:"swap"`
	Disk                 wireDisk              `json:"disk"`
	Net                  []core.NetStat        `json:"net"`
	Filesystems          []core.FilesystemStat `json:"filesystems"`
	TCPRetransSegsPerSec int64                 `json:"tcp_retrans_segs_per_sec"`
	TCPResetsPerSec      int64                 `json:"tcp_resets_per_sec"`
	CPUCoresUsed         []core.CPUCore        `json:"cpu_core_usage"`
	TopProcesses         []core.ProcessStat    `json:"top_processes"`
	GPUProcesses         []core.GPUProcessStat `json:"gpu_processes"`
	GPUs                 []core.GPUStat        `json:"gpus"`
}

// Parser converts sampler JSON lines into monitor samples.
type Parser struct {
	lastErr error
}

// HandleLine parses one sampler output line and reports whether it produced a sample.
func (p *Parser) HandleLine(line string) (*core.Sample, bool) {
	if strings.TrimSpace(line) == "" {
		return nil, false
	}

	var wire wireSample
	if err := json.Unmarshal([]byte(line), &wire); err != nil {
		p.lastErr = fmt.Errorf("parse sample JSON: %w", err)

		return nil, false
	}
	if wire.Version != wireProtocolVersion {
		p.lastErr = fmt.Errorf("%w %d", errUnsupportedSampleVersion, wire.Version)

		return nil, false
	}
	p.lastErr = nil

	return &core.Sample{
		RemoteEpoch:           wire.RemoteEpoch,
		RemoteTimestamp:       wire.RemoteTime,
		RemoteName:            wire.RemoteName,
		UptimeSeconds:         wire.Uptime,
		Load1:                 wire.Load1,
		Load5:                 wire.Load5,
		Load15:                wire.Load15,
		CPUCores:              wire.CPUCores,
		CPUName:               wire.CPUName,
		CPUPercent:            wire.CPUPercent,
		CPUUserPercent:        wire.CPUUserPercent,
		CPUSystemPercent:      wire.CPUSystemPercent,
		CPUIOWaitPercent:      wire.CPUIOWaitPercent,
		CPUStealPercent:       wire.CPUStealPercent,
		RAMUsedMiB:            wire.RAMUsedMiB,
		RAMTotalMiB:           wire.RAMTotalMiB,
		RAMAvailableMiB:       wire.RAMAvailableMiB,
		RAMFreeMiB:            wire.RAMFreeMiB,
		RAMCacheMiB:           wire.RAMCacheMiB,
		RAMBuffersMiB:         wire.RAMBuffersMiB,
		RAMReclaimableMiB:     wire.RAMReclaimableMiB,
		RAMSharedMiB:          wire.RAMSharedMiB,
		CPUFreqMHz:            wire.CPUFreqMHz,
		CPUMaxFreqMHz:         wire.CPUMaxFreqMHz,
		CPUTempC:              wire.CPUTempC,
		CPUPressureSomeAvg10:  wire.CPUPressureSome,
		CPUPressureFullAvg10:  wire.CPUPressureFull,
		MemPressureSomeAvg10:  wire.MemPressureSome,
		MemPressureFullAvg10:  wire.MemPressureFull,
		SwapFreeKiB:           wire.Swap.FreeKiB,
		SwapTotalKiB:          wire.Swap.TotalKiB,
		SwapInBps:             wire.Swap.InBps,
		SwapOutBps:            wire.Swap.OutBps,
		RootSource:            wire.Disk.RootSource,
		RootUsedKiB:           wire.Disk.RootUsedKiB,
		RootTotalKiB:          wire.Disk.RootTotalKiB,
		RootUsedPercent:       wire.Disk.RootUsedPercent,
		DiskDevice:            wire.Disk.Device,
		DiskReadBps:           wire.Disk.ReadBps,
		DiskWriteBps:          wire.Disk.WriteBps,
		DiskReadMergedPerSec:  wire.Disk.ReadMergedPerSec,
		DiskWriteMergedPerSec: wire.Disk.WriteMergedPerSec,
		DiskUtil:              wire.Disk.UtilPercent,
		DiskAwaitMS:           wire.Disk.AwaitMS,
		DiskQueueDepth:        wire.Disk.QueueDepth,
		DiskInflight:          wire.Disk.Inflight,
		TCPRetransSegsPerSec:  wire.TCPRetransSegsPerSec,
		TCPResetsPerSec:       wire.TCPResetsPerSec,
		Net:                   wire.Net,
		Filesystems:           wire.Filesystems,
		CPUCoresUsage:         wire.CPUCoresUsed,
		TopProcesses:          wire.TopProcesses,
		GPUProcesses:          wire.GPUProcesses,
		GPUs:                  wire.GPUs,
		ReceivedAt:            time.Time{},
	}, true
}

// LastError returns the latest non-empty line rejection, if any.
func (p *Parser) LastError() error {
	return p.lastErr
}
