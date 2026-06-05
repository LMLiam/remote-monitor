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
	RemoteEpoch           int64
	RemoteTimestamp       string
	RemoteName            string
	UptimeSeconds         int64
	Load1, Load5, Load15  float64
	CPUCores              int
	CPUName               string
	CPUPercent            int
	CPUUserPercent        int
	CPUSystemPercent      int
	CPUIOWaitPercent      int
	CPUStealPercent       int
	RAMUsedMiB            int64
	RAMTotalMiB           int64
	RAMAvailableMiB       int64
	RAMFreeMiB            int64
	RAMCacheMiB           int64
	RAMBuffersMiB         int64
	RAMReclaimableMiB     int64
	RAMSharedMiB          int64
	CPUFreqMHz            int
	CPUMaxFreqMHz         int
	CPUTempC              int
	CPUPressureSomeAvg10  float64
	CPUPressureFullAvg10  float64
	MemPressureSomeAvg10  float64
	MemPressureFullAvg10  float64
	SwapFreeKiB           int64
	SwapTotalKiB          int64
	SwapInBps             int64
	SwapOutBps            int64
	RootSource            string
	RootUsedKiB           int64
	RootTotalKiB          int64
	RootUsedPercent       int
	DiskDevice            string
	DiskReadBps           int64
	DiskWriteBps          int64
	DiskReadMergedPerSec  int64
	DiskWriteMergedPerSec int64
	DiskUtil              int
	DiskAwaitMS           float64
	DiskQueueDepth        float64
	DiskInflight          int
	TCPRetransSegsPerSec  int64
	TCPResetsPerSec       int64
	Net                   []NetStat
	Filesystems           []FilesystemStat
	CPUCoresUsage         []CPUCore
	TopProcesses          []ProcessStat
	GPUProcesses          []GPUProcessStat
	GPUs                  []GPUStat
	PowerSupplies         []PowerSupplyStat
	ExternalPowerOnline   int
	BatteryPercent        int
	BatteryStatus         string
	PowerDrawWatts        float64
	UPSPresent            int
	PowerSourceName       string
	ReceivedAt            time.Time
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
	return Sample{
		RemoteEpoch:           0,
		RemoteTimestamp:       "",
		RemoteName:            "",
		UptimeSeconds:         0,
		Load1:                 0,
		Load5:                 0,
		Load15:                0,
		CPUCores:              0,
		CPUName:               "",
		CPUPercent:            0,
		CPUUserPercent:        0,
		CPUSystemPercent:      0,
		CPUIOWaitPercent:      0,
		CPUStealPercent:       0,
		RAMUsedMiB:            0,
		RAMTotalMiB:           0,
		RAMAvailableMiB:       0,
		RAMFreeMiB:            0,
		RAMCacheMiB:           0,
		RAMBuffersMiB:         0,
		RAMReclaimableMiB:     0,
		RAMSharedMiB:          0,
		CPUFreqMHz:            0,
		CPUMaxFreqMHz:         0,
		CPUTempC:              0,
		CPUPressureSomeAvg10:  0,
		CPUPressureFullAvg10:  0,
		MemPressureSomeAvg10:  0,
		MemPressureFullAvg10:  0,
		SwapFreeKiB:           0,
		SwapTotalKiB:          0,
		SwapInBps:             0,
		SwapOutBps:            0,
		RootSource:            "",
		RootUsedKiB:           0,
		RootTotalKiB:          0,
		RootUsedPercent:       0,
		DiskDevice:            "",
		DiskReadBps:           0,
		DiskWriteBps:          0,
		DiskReadMergedPerSec:  0,
		DiskWriteMergedPerSec: 0,
		DiskUtil:              0,
		DiskAwaitMS:           0,
		DiskQueueDepth:        0,
		DiskInflight:          0,
		TCPRetransSegsPerSec:  0,
		TCPResetsPerSec:       0,
		Net:                   nil,
		Filesystems:           nil,
		CPUCoresUsage:         nil,
		TopProcesses:          nil,
		GPUProcesses:          nil,
		GPUs:                  nil,
		PowerSupplies:         nil,
		ExternalPowerOnline:   -1,
		BatteryPercent:        -1,
		BatteryStatus:         "",
		PowerDrawWatts:        -1,
		UPSPresent:            -1,
		PowerSourceName:       "",
		ReceivedAt:            time.Time{},
	}
}

// EmptyFilesystemStat returns the zero filesystem sample value.
func EmptyFilesystemStat() FilesystemStat {
	return FilesystemStat{
		Source:            "",
		Mount:             "",
		UsedKiB:           0,
		TotalKiB:          0,
		UsedPercent:       0,
		InodesUsedPercent: 0,
	}
}
