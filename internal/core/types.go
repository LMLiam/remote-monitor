package core

import "time"

// Theme constants and runtime states shared across the monitor pipeline.
const (
	ThemeAurora    = "aurora"
	ThemeBasic     = "basic"
	ThemeWindowsXP = "windows-xp"

	OutputModeAuto  = ""
	OutputModeTUI   = "tui"
	OutputModeText  = "text"
	OutputModeJSONL = "jsonl"

	ProcessSortCPU    = "cpu"
	ProcessSortMemory = "mem"

	StatusConnecting   = "connecting"
	StatusDisconnected = "disconnected"
	StatusLive         = "live"
	StatusStale        = "stale"

	DetailOpeningSSHSession = "opening ssh session"
	DetailStreamHealthy     = "stream healthy"
	DetailSSHStreamEnded    = "ssh stream ended"

	DefaultProcessCount = 4
)

// Config contains CLI, SSH, sampling, and rendering settings.
type Config struct {
	Host               string
	Interval           time.Duration
	ProcessSort        string
	ProcessFilter      string
	ProcessCount       int
	NetIncludePatterns []string
	NetExcludePatterns []string
	NetAggregate       bool
	HistoryLimit       int
	StaleAfter         time.Duration
	ReconnectBaseDelay time.Duration
	RenderFPS          int
	Compact            bool
	NoBanner           bool
	ShowVersion        bool
	Once               bool
	OutputMode         string
	OutputPath         string
	Theme              string
	DisableTrueColor   bool
	SSHConnectTimeout  time.Duration
	SSHAliveInterval   time.Duration
	SSHAliveCountMax   int
	SSHControlPersist  time.Duration
	SSHControlPath     string
}

// NetStat contains one sampled network interface snapshot.
type NetStat struct {
	Iface      string `json:"iface"`
	RXBps      int64  `json:"rx_bps"`
	TXBps      int64  `json:"tx_bps"`
	RXPps      int64  `json:"rx_pps"`
	TXPps      int64  `json:"tx_pps"`
	SpeedMbps  int    `json:"speed_mbps"`
	RXDrops    int64  `json:"rx_drops"`
	RXErrors   int64  `json:"rx_errors"`
	RXOverruns int64  `json:"rx_overruns"`
	TXDrops    int64  `json:"tx_drops"`
	TXErrors   int64  `json:"tx_errors"`
	TXOverruns int64  `json:"tx_overruns"`
}

// CPUCore contains one sampled CPU core utilization.
type CPUCore struct {
	Index   int `json:"index"`
	Percent int `json:"percent"`
}

// ProcessStat contains one sampled host process row.
type ProcessStat struct {
	PID        int    `json:"pid"`
	Command    string `json:"command"`
	CPUPercent int    `json:"cpu_percent"`
	RSSMiB     int64  `json:"rss_mib"`
}

// GPUProcessStat contains one sampled GPU process row.
type GPUProcessStat struct {
	GPUUUID    string `json:"gpu_uuid"`
	PID        int    `json:"pid"`
	Command    string `json:"command"`
	UsedMemMiB int64  `json:"used_mem_mib"`
}

// FilesystemStat contains one sampled filesystem usage row.
type FilesystemStat struct {
	Source            string `json:"source"`
	Mount             string `json:"mount"`
	UsedKiB           int64  `json:"used_kib"`
	TotalKiB          int64  `json:"total_kib"`
	UsedPercent       int    `json:"used_percent"`
	InodesUsedPercent int    `json:"inodes_used_percent"`
}

// DiskStat contains one sampled block device I/O snapshot.
type DiskStat struct {
	Device            string  `json:"device"`
	ReadBps           int64   `json:"read_bps"`
	WriteBps          int64   `json:"write_bps"`
	ReadMergedPerSec  int64   `json:"read_merged_per_sec"`
	WriteMergedPerSec int64   `json:"write_merged_per_sec"`
	Util              int     `json:"util_percent"`
	AwaitMS           float64 `json:"await_ms"`
	QueueDepth        float64 `json:"queue_depth"`
	Inflight          int     `json:"inflight"`
}

// GPUStat contains one sampled GPU device snapshot.
type GPUStat struct {
	Index            int     `json:"index"`
	UUID             string  `json:"uuid"`
	Name             string  `json:"name"`
	Util             int     `json:"util_percent"`
	MemUtil          int     `json:"mem_util_percent"`
	EncoderUtil      int     `json:"encoder_util_percent"`
	DecoderUtil      int     `json:"decoder_util_percent"`
	MemUsed          int64   `json:"mem_used_mib"`
	MemTotal         int64   `json:"mem_total_mib"`
	Temp             int     `json:"temp_c"`
	PowerDraw        float64 `json:"power_draw_w"`
	PowerLimit       float64 `json:"power_limit_w"`
	Fan              int     `json:"fan_percent"`
	SMClock          int     `json:"sm_clock_mhz"`
	MaxSMClock       int     `json:"sm_clock_max_mhz"`
	MemClock         int     `json:"mem_clock_mhz"`
	MaxMemClock      int     `json:"mem_clock_max_mhz"`
	GraphicsClock    int     `json:"graphics_clock_mhz"`
	VideoClock       int     `json:"video_clock_mhz"`
	PCIeGenCurrent   int     `json:"pcie_gen_current"`
	PCIeGenMax       int     `json:"pcie_gen_max"`
	PCIeWidthCurrent int     `json:"pcie_width_current"`
	PCIeWidthMax     int     `json:"pcie_width_max"`
	ThrottleReasons  string  `json:"throttle_reasons"`
	PState           string  `json:"p_state"`
}

// PowerSupplyStat contains one sampled Linux power-supply sysfs device.
type PowerSupplyStat struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Online          int     `json:"online"`
	CapacityPercent int     `json:"capacity_percent"`
	Status          string  `json:"status"`
	PowerDrawWatts  float64 `json:"power_draw_w"`
	Present         int     `json:"present"`
}

// Sample contains one complete sampler payload after JSON parsing.
type Sample struct {
	RemoteEpoch           int64             `json:"remote_epoch"`
	RemoteTimestamp       string            `json:"remote_timestamp"`
	RemoteName            string            `json:"remote_name"`
	UptimeSeconds         int64             `json:"uptime_seconds"`
	Load1                 float64           `json:"load1"`
	Load5                 float64           `json:"load5"`
	Load15                float64           `json:"load15"`
	CPUCores              int               `json:"cpu_cores"`
	CPUName               string            `json:"cpu_name"`
	CPUPercent            int               `json:"cpu_percent"`
	CPUUserPercent        int               `json:"cpu_user_percent"`
	CPUSystemPercent      int               `json:"cpu_system_percent"`
	CPUIOWaitPercent      int               `json:"cpu_iowait_percent"`
	CPUStealPercent       int               `json:"cpu_steal_percent"`
	RAMUsedMiB            int64             `json:"ram_used_mib"`
	RAMTotalMiB           int64             `json:"ram_total_mib"`
	RAMAvailableMiB       int64             `json:"ram_available_mib"`
	RAMFreeMiB            int64             `json:"ram_free_mib"`
	RAMCacheMiB           int64             `json:"ram_cache_mib"`
	RAMBuffersMiB         int64             `json:"ram_buffers_mib"`
	RAMReclaimableMiB     int64             `json:"ram_reclaimable_mib"`
	RAMSharedMiB          int64             `json:"ram_shared_mib"`
	CPUFreqMHz            int               `json:"cpu_freq_mhz"`
	CPUMaxFreqMHz         int               `json:"cpu_max_freq_mhz"`
	CPUTempC              int               `json:"cpu_temp_c"`
	CPUPressureSomeAvg10  float64           `json:"cpu_pressure_some_avg10"`
	CPUPressureFullAvg10  float64           `json:"cpu_pressure_full_avg10"`
	MemPressureSomeAvg10  float64           `json:"mem_pressure_some_avg10"`
	MemPressureFullAvg10  float64           `json:"mem_pressure_full_avg10"`
	SwapFreeKiB           int64             `json:"swap_free_kib"`
	SwapTotalKiB          int64             `json:"swap_total_kib"`
	SwapInBps             int64             `json:"swap_in_bps"`
	SwapOutBps            int64             `json:"swap_out_bps"`
	RootSource            string            `json:"root_source"`
	RootUsedKiB           int64             `json:"root_used_kib"`
	RootTotalKiB          int64             `json:"root_total_kib"`
	RootUsedPercent       int               `json:"root_used_percent"`
	DiskDevice            string            `json:"disk_device"`
	DiskReadBps           int64             `json:"disk_read_bps"`
	DiskWriteBps          int64             `json:"disk_write_bps"`
	DiskReadMergedPerSec  int64             `json:"disk_read_merged_per_sec"`
	DiskWriteMergedPerSec int64             `json:"disk_write_merged_per_sec"`
	DiskUtil              int               `json:"disk_util_percent"`
	DiskAwaitMS           float64           `json:"disk_await_ms"`
	DiskQueueDepth        float64           `json:"disk_queue_depth"`
	DiskInflight          int               `json:"disk_inflight"`
	TCPRetransSegsPerSec  int64             `json:"tcp_retrans_segs_per_sec"`
	TCPResetsPerSec       int64             `json:"tcp_resets_per_sec"`
	Net                   []NetStat         `json:"net"`
	Filesystems           []FilesystemStat  `json:"filesystems"`
	Disks                 []DiskStat        `json:"disks"`
	CPUCoresUsage         []CPUCore         `json:"cpu_core_usage"`
	TopProcesses          []ProcessStat     `json:"top_processes"`
	GPUProcesses          []GPUProcessStat  `json:"gpu_processes"`
	GPUs                  []GPUStat         `json:"gpus"`
	PowerSupplies         []PowerSupplyStat `json:"power_supplies"`
	ExternalPowerOnline   int               `json:"external_power_online"`
	BatteryPercent        int               `json:"battery_percent"`
	BatteryStatus         string            `json:"battery_status"`
	PowerDrawWatts        float64           `json:"power_draw_w"`
	UPSPresent            int               `json:"ups_present"`
	PowerSourceName       string            `json:"power_source_name"`
	ReceivedAt            time.Time         `json:"-"`
}

// StreamEvent describes connection lifecycle state from the sampler stream.
type StreamEvent struct {
	State          string
	Detail         string
	ReconnectCount int
	Attempts       int
	StreamAlive    bool
	NextRetry      time.Time
	At             time.Time
}

// AppState contains the current dashboard state and rolling history.
type AppState struct {
	Cfg                Config
	RuntimeState       string
	RuntimeDetail      string
	LastTransport      string
	SampleCount        int
	ReconnectCount     int
	ReconnectAttempts  int
	NextRetry          time.Time
	LastRx             time.Time
	StreamAlive        bool
	Current            Sample
	HasSample          bool
	ScrollOffset       int
	ScrollMax          int
	NetCeilings        map[string]int64
	CPUHistory         []int
	CPUFreqHistory     []int
	CPUTempHistory     []int
	RAMHistory         []int
	RAMAvailHistory    []int
	DiskHistory        []int
	DiskLatencyHistory []int
	GPUHistory         []int
	VRAMHistory        []int
	TempHistory        []int
	PowerHistory       []int
	NetRXHistory       []int64
	NetTXHistory       []int64
	NetIssueHistory    []int
}

// EmptySample returns a sample populated with sentinel zero and unknown values.
func EmptySample() Sample {
	var sample Sample
	sample.ExternalPowerOnline = -1
	sample.BatteryPercent = -1
	sample.PowerDrawWatts = -1
	sample.UPSPresent = -1

	return sample
}

// EmptyFilesystemStat returns the zero filesystem sample value.
func EmptyFilesystemStat() FilesystemStat {
	var stat FilesystemStat

	return stat
}
