package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
)

// ErrEmptyHost reports a missing SSH host in parsed configuration.
var ErrEmptyHost = errors.New("host cannot be empty")

// ErrUnknownOutputMode reports an unsupported -output value.
var ErrUnknownOutputMode = errors.New("unknown output mode")

// ErrUnknownProcessSort reports an unsupported -process-sort value.
var ErrUnknownProcessSort = errors.New("unknown process sort mode")

// ErrInvalidProcessCount reports an unsupported -process-count value.
var ErrInvalidProcessCount = errors.New("invalid process count")

// ErrInvalidNetworkPattern reports an unsupported network interface pattern.
var ErrInvalidNetworkPattern = errors.New("invalid network interface pattern")

// ErrInvalidThreshold reports an unsupported alert/severity threshold value.
var ErrInvalidThreshold = errors.New("invalid threshold")

const (
	defaultHistoryLimit       = 240
	defaultReconnectDelaySecs = 2
	defaultRenderFPS          = 12
	defaultSSHTimeoutSecs     = 5
	defaultSSHAliveCount      = 2
	defaultSSHControlSecs     = 30
	minHistoryLimit           = 30
	minStaleAfterSecs         = 3
	maxRenderFPS              = 60
	minThresholdTempC         = 0
	maxThresholdTempC         = 150
	minThresholdPercent       = 0
	maxThresholdPercent       = 100
)

type thresholdFlagValues struct {
	cpuCriticalPercent          *int
	cpuWarnTemp                 *int
	cpuCriticalTemp             *int
	ramWarnAvailablePercent     *int
	ramCriticalAvailablePercent *int
	gpuWarnTemp                 *int
	gpuCriticalTemp             *int
	vramWarnPercent             *int
	vramCriticalPercent         *int
	diskWarnPercent             *int
	diskCriticalPercent         *int
}

// ParseConfig builds monitor configuration from CLI args and environment defaults.
func ParseConfig(args []string) (core.Config, error) {
	envDefaults := configValuesFromEnv()
	cliValues := envDefaults

	fs := flag.NewFlagSet("remote-monitor", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	configPath := fs.String("config", defaultConfigPath(), "Path to TOML config file")
	profileName := fs.String("profile", "", "Named profile from the config file")
	host := fs.String("host", cliValues.host, "SSH host to monitor")
	interval := fs.Int("interval", cliValues.interval, "Refresh interval in seconds")
	history := fs.Int("history", cliValues.history, "History Sample limit")
	staleAfter := fs.Int("stale-after", cliValues.staleAfter, "Seconds before live data is considered stale")
	reconnectDelay := fs.Int("reconnect-delay", cliValues.reconnectDelay, "Base reconnect delay in seconds")
	fps := fs.Int("fps", cliValues.fps, "TTY redraw frames per second")
	compact := fs.Bool("compact", cliValues.compact, "Use a compact stacked layout")
	noBanner := fs.Bool("no-banner", cliValues.noBanner, "Disable the large rendered title banner")
	showVersion := fs.Bool("version", false, "Print version information and exit")
	once := fs.Bool("once", false, "Collect one sample and exit")
	outputMode := fs.String("output", core.OutputModeAuto, "Output mode (tui, text, jsonl)")
	outputPath := fs.String("out", "", "Write JSONL output to this file")
	processSort := fs.String("process-sort", cliValues.processSort, "Process sort order (cpu, mem)")
	processFilter := fs.String("process-filter", cliValues.processFilter, "Case-insensitive process filter text")
	processCount := fs.Int("process-count", cliValues.processCount, "Maximum process rows per sample")
	netInclude := fs.String("net-include", "", "Comma-separated network interface names or glob patterns to include")
	netExclude := fs.String("net-exclude", "", "Comma-separated network interface names or glob patterns to exclude")
	netAggregate := fs.Bool("net-aggregate", false, "Replace per-interface network rows with one selected-interface aggregate")
	theme := fs.String("theme", cliValues.theme, "Color theme (aurora, basic, windows-xp)")
	noTrueColor := fs.Bool("no-truecolor", cliValues.noTrueColor, "Force 256-color rendering even on truecolor terminals")
	sshConnectTimeout := fs.Int("ssh-connect-timeout", cliValues.sshConnectTimeout, "SSH connect timeout in seconds")
	sshAliveInterval := fs.Int("ssh-server-alive", cliValues.sshAliveInterval, "SSH keepalive interval in seconds")
	sshAliveCount := fs.Int("ssh-server-alive-count", cliValues.sshAliveCount, "SSH keepalive failure threshold before reconnect")
	sshControlPersist := fs.Int("ssh-control-persist", cliValues.sshControlPersist, "SSH control socket persist time in seconds")
	thresholdFlags := registerThresholdFlags(fs, cliValues.thresholds)

	if err := fs.Parse(args); err != nil {
		return core.Config{}, err
	}

	if *showVersion {
		return versionOnlyConfig(), nil
	}
	resolvedOutputMode, err := parseOutputMode(*outputMode)
	if err != nil {
		return core.Config{}, err
	}
	resolvedProcessSort, err := parseProcessSort(*processSort)
	if err != nil {
		return core.Config{}, err
	}
	if *processCount < 1 {
		return core.Config{}, fmt.Errorf("%w: process count must be at least 1", ErrInvalidProcessCount)
	}
	explicitFlags := visitedFlags(fs)
	netIncludePatterns, err := parseNetworkPatterns(*netInclude, explicitFlags["net-include"])
	if err != nil {
		return core.Config{}, err
	}
	netExcludePatterns, err := parseNetworkPatterns(*netExclude, explicitFlags["net-exclude"])
	if err != nil {
		return core.Config{}, err
	}

	cliValues = configValues{
		host:              *host,
		interval:          *interval,
		processSort:       resolvedProcessSort,
		processFilter:     strings.TrimSpace(*processFilter),
		processCount:      *processCount,
		history:           *history,
		staleAfter:        *staleAfter,
		reconnectDelay:    *reconnectDelay,
		fps:               *fps,
		compact:           *compact,
		noBanner:          *noBanner,
		theme:             *theme,
		noTrueColor:       *noTrueColor,
		sshConnectTimeout: *sshConnectTimeout,
		sshAliveInterval:  *sshAliveInterval,
		sshAliveCount:     *sshAliveCount,
		sshControlPersist: *sshControlPersist,
		thresholds:        thresholdFlags.thresholds(),
	}

	resolved := envDefaults
	selectedProfile := strings.TrimSpace(*profileName)
	if selectedProfile != "" {
		profile, err := loadProfile(*configPath, selectedProfile)
		if err != nil {
			return core.Config{}, err
		}
		if err := applyProfile(&resolved, profile, selectedProfile); err != nil {
			return core.Config{}, err
		}
	}

	applyExplicitFlags(&resolved, cliValues, explicitFlags)
	if err := validateThresholds(resolved.thresholds, ""); err != nil {
		return core.Config{}, err
	}
	if fs.NArg() > 0 {
		resolved.host = fs.Arg(0)
	}
	if strings.TrimSpace(resolved.host) == "" {
		return core.Config{}, ErrEmptyHost
	}

	resolved.clamp()
	resolved.theme = core.CanonicalThemeName(resolved.theme)

	return buildCoreConfig(resolved, netIncludePatterns, netExcludePatterns, *netAggregate, *once, resolvedOutputMode, *outputPath), nil
}

func buildCoreConfig(
	resolved configValues,
	netIncludePatterns []string,
	netExcludePatterns []string,
	netAggregate bool,
	once bool,
	outputMode string,
	outputPath string,
) core.Config {
	return core.Config{
		Host:               resolved.host,
		Interval:           time.Duration(resolved.interval) * time.Second,
		ProcessSort:        resolved.processSort,
		ProcessFilter:      resolved.processFilter,
		ProcessCount:       resolved.processCount,
		NetIncludePatterns: netIncludePatterns,
		NetExcludePatterns: netExcludePatterns,
		NetAggregate:       netAggregate,
		HistoryLimit:       resolved.history,
		StaleAfter:         time.Duration(resolved.staleAfter) * time.Second,
		ReconnectBaseDelay: time.Duration(resolved.reconnectDelay) * time.Second,
		RenderFPS:          resolved.fps,
		Compact:            resolved.compact,
		NoBanner:           resolved.noBanner,
		ShowVersion:        false,
		Once:               once,
		OutputMode:         outputMode,
		OutputPath:         outputPath,
		Theme:              resolved.theme,
		DisableTrueColor:   resolved.noTrueColor,
		SSHConnectTimeout:  time.Duration(resolved.sshConnectTimeout) * time.Second,
		SSHAliveInterval:   time.Duration(resolved.sshAliveInterval) * time.Second,
		SSHAliveCountMax:   resolved.sshAliveCount,
		SSHControlPersist:  time.Duration(resolved.sshControlPersist) * time.Second,
		SSHControlPath:     "",
		Thresholds:         resolved.thresholds,
	}
}

func parseNetworkPatterns(value string, explicit bool) ([]string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if explicit {
			return nil, fmt.Errorf("%w: empty pattern", ErrInvalidNetworkPattern)
		}

		return nil, nil
	}

	rawPatterns := strings.Split(trimmed, ",")
	patterns := make([]string, 0, len(rawPatterns))
	for _, raw := range rawPatterns {
		pattern := strings.TrimSpace(raw)
		if pattern == "" {
			return nil, fmt.Errorf("%w: empty pattern", ErrInvalidNetworkPattern)
		}
		if _, err := path.Match(pattern, ""); err != nil {
			return nil, fmt.Errorf("%w %q: %w", ErrInvalidNetworkPattern, pattern, err)
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

func parseOutputMode(mode string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(mode))
	switch trimmed {
	case core.OutputModeAuto, core.OutputModeTUI, core.OutputModeText, core.OutputModeJSONL:
		return trimmed, nil
	default:
		return "", fmt.Errorf("%w %q (expected one of: tui, text, jsonl)", ErrUnknownOutputMode, mode)
	}
}

func parseProcessSort(mode string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(mode))
	switch trimmed {
	case "", core.ProcessSortCPU:
		return core.ProcessSortCPU, nil
	case core.ProcessSortMemory:
		return core.ProcessSortMemory, nil
	default:
		return "", fmt.Errorf("%w %q (expected one of: cpu, mem)", ErrUnknownProcessSort, mode)
	}
}

type configValues struct {
	host              string
	interval          int
	processSort       string
	processFilter     string
	processCount      int
	history           int
	staleAfter        int
	reconnectDelay    int
	fps               int
	compact           bool
	noBanner          bool
	theme             string
	noTrueColor       bool
	sshConnectTimeout int
	sshAliveInterval  int
	sshAliveCount     int
	sshControlPersist int
	thresholds        core.Thresholds
}

func registerThresholdFlags(fs *flag.FlagSet, defaults core.Thresholds) thresholdFlagValues {
	return thresholdFlagValues{
		cpuCriticalPercent:          fs.Int("cpu-critical-percent", defaults.CPUCriticalPercent, "CPU utilization critical threshold percent"),
		cpuWarnTemp:                 fs.Int("cpu-warn-temp", defaults.CPUWarnTemp, "CPU temperature warning threshold in Celsius"),
		cpuCriticalTemp:             fs.Int("cpu-critical-temp", defaults.CPUCriticalTemp, "CPU temperature critical threshold in Celsius"),
		ramWarnAvailablePercent:     fs.Int("ram-warn-available-percent", defaults.RAMWarnAvailablePercent, "RAM available warning threshold percent"),
		ramCriticalAvailablePercent: fs.Int("ram-critical-available-percent", defaults.RAMCriticalAvailablePercent, "RAM available critical threshold percent"),
		gpuWarnTemp:                 fs.Int("gpu-warn-temp", defaults.GPUWarnTemp, "GPU temperature warning threshold in Celsius"),
		gpuCriticalTemp:             fs.Int("gpu-critical-temp", defaults.GPUCriticalTemp, "GPU temperature critical threshold in Celsius"),
		vramWarnPercent:             fs.Int("vram-warn-percent", defaults.VRAMWarnPercent, "VRAM utilization warning threshold percent"),
		vramCriticalPercent:         fs.Int("vram-critical-percent", defaults.VRAMCriticalPercent, "VRAM utilization critical threshold percent"),
		diskWarnPercent:             fs.Int("disk-warn-percent", defaults.DiskWarnPercent, "Disk usage warning threshold percent"),
		diskCriticalPercent:         fs.Int("disk-critical-percent", defaults.DiskCriticalPercent, "Disk usage critical threshold percent"),
	}
}

func (values thresholdFlagValues) thresholds() core.Thresholds {
	return core.Thresholds{
		CPUCriticalPercent:          *values.cpuCriticalPercent,
		CPUWarnTemp:                 *values.cpuWarnTemp,
		CPUCriticalTemp:             *values.cpuCriticalTemp,
		RAMWarnAvailablePercent:     *values.ramWarnAvailablePercent,
		RAMCriticalAvailablePercent: *values.ramCriticalAvailablePercent,
		GPUWarnTemp:                 *values.gpuWarnTemp,
		GPUCriticalTemp:             *values.gpuCriticalTemp,
		VRAMWarnPercent:             *values.vramWarnPercent,
		VRAMCriticalPercent:         *values.vramCriticalPercent,
		DiskWarnPercent:             *values.diskWarnPercent,
		DiskCriticalPercent:         *values.diskCriticalPercent,
	}
}

func configValuesFromEnv() configValues {
	intervalDefault := getenvInt("MONITOR_INTERVAL", 1)
	thresholds := core.DefaultThresholds()
	thresholds.CPUCriticalPercent = getenvInt("MONITOR_CPU_CRITICAL_PERCENT", thresholds.CPUCriticalPercent)
	thresholds.CPUWarnTemp = getenvInt("MONITOR_CPU_WARN_TEMP", thresholds.CPUWarnTemp)
	thresholds.CPUCriticalTemp = getenvInt("MONITOR_CPU_CRITICAL_TEMP", thresholds.CPUCriticalTemp)
	thresholds.RAMWarnAvailablePercent = getenvInt("MONITOR_RAM_WARN_AVAILABLE_PERCENT", thresholds.RAMWarnAvailablePercent)
	thresholds.RAMCriticalAvailablePercent = getenvInt("MONITOR_RAM_CRITICAL_AVAILABLE_PERCENT", thresholds.RAMCriticalAvailablePercent)
	thresholds.GPUWarnTemp = getenvInt("MONITOR_GPU_WARN_TEMP", thresholds.GPUWarnTemp)
	thresholds.GPUCriticalTemp = getenvInt("MONITOR_GPU_CRITICAL_TEMP", thresholds.GPUCriticalTemp)
	thresholds.VRAMWarnPercent = getenvInt("MONITOR_VRAM_WARN_PERCENT", thresholds.VRAMWarnPercent)
	thresholds.VRAMCriticalPercent = getenvInt("MONITOR_VRAM_CRITICAL_PERCENT", thresholds.VRAMCriticalPercent)
	thresholds.DiskWarnPercent = getenvInt("MONITOR_DISK_WARN_PERCENT", thresholds.DiskWarnPercent)
	thresholds.DiskCriticalPercent = getenvInt("MONITOR_DISK_CRITICAL_PERCENT", thresholds.DiskCriticalPercent)

	return configValues{
		host:              getenvDefault("REMOTE_MONITOR_HOST", ""),
		interval:          intervalDefault,
		processSort:       core.ProcessSortCPU,
		processFilter:     "",
		processCount:      core.DefaultProcessCount,
		history:           getenvInt("MONITOR_HISTORY_LIMIT", defaultHistoryLimit),
		staleAfter:        getenvInt("MONITOR_STALE_AFTER", intervalDefault*3+1),
		reconnectDelay:    getenvInt("MONITOR_RECONNECT_DELAY", defaultReconnectDelaySecs),
		fps:               getenvInt("MONITOR_FPS", defaultRenderFPS),
		compact:           getenvBool("MONITOR_COMPACT", false),
		noBanner:          getenvBool("MONITOR_NO_BANNER", false),
		theme:             getenvDefault("MONITOR_THEME", core.ThemeAurora),
		noTrueColor:       getenvBool("MONITOR_NO_TRUECOLOR", false),
		sshConnectTimeout: getenvInt("MONITOR_SSH_CONNECT_TIMEOUT", defaultSSHTimeoutSecs),
		sshAliveInterval:  getenvInt("MONITOR_SSH_ALIVE_INTERVAL", defaultSSHTimeoutSecs),
		sshAliveCount:     getenvInt("MONITOR_SSH_ALIVE_COUNT", defaultSSHAliveCount),
		sshControlPersist: getenvInt("MONITOR_SSH_CONTROL_PERSIST", defaultSSHControlSecs),
		thresholds:        thresholds,
	}
}

func (values *configValues) clamp() {
	if values.interval < 1 {
		values.interval = 1
	}
	if values.history < minHistoryLimit {
		values.history = minHistoryLimit
	}
	if values.staleAfter < minStaleAfterSecs {
		values.staleAfter = minStaleAfterSecs
	}
	if values.reconnectDelay < 1 {
		values.reconnectDelay = 1
	}
	if values.fps < 1 {
		values.fps = 1
	}
	if values.fps > maxRenderFPS {
		values.fps = maxRenderFPS
	}
	if values.sshConnectTimeout < 1 {
		values.sshConnectTimeout = 1
	}
	if values.sshAliveInterval < 1 {
		values.sshAliveInterval = 1
	}
	if values.sshAliveCount < 1 {
		values.sshAliveCount = 1
	}
	if values.sshControlPersist < 0 {
		values.sshControlPersist = 0
	}
}

func versionOnlyConfig() core.Config {
	var cfg core.Config
	cfg.ShowVersion = true

	return cfg
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := map[string]bool{}
	fs.Visit(func(flag *flag.Flag) {
		visited[flag.Name] = true
	})

	return visited
}

func applyExplicitFlags(resolved *configValues, cli configValues, explicit map[string]bool) {
	if explicit["host"] {
		resolved.host = cli.host
	}
	if explicit["interval"] {
		resolved.interval = cli.interval
	}
	if explicit["process-sort"] {
		resolved.processSort = cli.processSort
	}
	if explicit["process-filter"] {
		resolved.processFilter = cli.processFilter
	}
	if explicit["process-count"] {
		resolved.processCount = cli.processCount
	}
	if explicit["history"] {
		resolved.history = cli.history
	}
	if explicit["stale-after"] {
		resolved.staleAfter = cli.staleAfter
	}
	if explicit["reconnect-delay"] {
		resolved.reconnectDelay = cli.reconnectDelay
	}
	if explicit["fps"] {
		resolved.fps = cli.fps
	}
	if explicit["compact"] {
		resolved.compact = cli.compact
	}
	if explicit["no-banner"] {
		resolved.noBanner = cli.noBanner
	}
	if explicit["theme"] {
		resolved.theme = cli.theme
	}
	if explicit["no-truecolor"] {
		resolved.noTrueColor = cli.noTrueColor
	}
	if explicit["ssh-connect-timeout"] {
		resolved.sshConnectTimeout = cli.sshConnectTimeout
	}
	if explicit["ssh-server-alive"] {
		resolved.sshAliveInterval = cli.sshAliveInterval
	}
	if explicit["ssh-server-alive-count"] {
		resolved.sshAliveCount = cli.sshAliveCount
	}
	if explicit["ssh-control-persist"] {
		resolved.sshControlPersist = cli.sshControlPersist
	}
	applyExplicitThresholdFlags(&resolved.thresholds, cli.thresholds, explicit)
}

func applyExplicitThresholdFlags(resolved *core.Thresholds, cli core.Thresholds, explicit map[string]bool) {
	if explicit["cpu-critical-percent"] {
		resolved.CPUCriticalPercent = cli.CPUCriticalPercent
	}
	if explicit["cpu-warn-temp"] {
		resolved.CPUWarnTemp = cli.CPUWarnTemp
	}
	if explicit["cpu-critical-temp"] {
		resolved.CPUCriticalTemp = cli.CPUCriticalTemp
	}
	if explicit["ram-warn-available-percent"] {
		resolved.RAMWarnAvailablePercent = cli.RAMWarnAvailablePercent
	}
	if explicit["ram-critical-available-percent"] {
		resolved.RAMCriticalAvailablePercent = cli.RAMCriticalAvailablePercent
	}
	if explicit["gpu-warn-temp"] {
		resolved.GPUWarnTemp = cli.GPUWarnTemp
	}
	if explicit["gpu-critical-temp"] {
		resolved.GPUCriticalTemp = cli.GPUCriticalTemp
	}
	if explicit["vram-warn-percent"] {
		resolved.VRAMWarnPercent = cli.VRAMWarnPercent
	}
	if explicit["vram-critical-percent"] {
		resolved.VRAMCriticalPercent = cli.VRAMCriticalPercent
	}
	if explicit["disk-warn-percent"] {
		resolved.DiskWarnPercent = cli.DiskWarnPercent
	}
	if explicit["disk-critical-percent"] {
		resolved.DiskCriticalPercent = cli.DiskCriticalPercent
	}
}

func validateThresholds(thresholds core.Thresholds, prefix string) error {
	if err := validateThresholdPercent(thresholds.CPUCriticalPercent, thresholdField(prefix, "cpu_critical_percent")); err != nil {
		return err
	}
	if err := validateThresholdTemp(thresholds.CPUWarnTemp, thresholdField(prefix, "cpu_warn_temp")); err != nil {
		return err
	}
	if err := validateThresholdTemp(thresholds.CPUCriticalTemp, thresholdField(prefix, "cpu_critical_temp")); err != nil {
		return err
	}
	if err := validateThresholdPercent(thresholds.RAMWarnAvailablePercent, thresholdField(prefix, "ram_warn_available_percent")); err != nil {
		return err
	}
	if err := validateThresholdPercent(thresholds.RAMCriticalAvailablePercent, thresholdField(prefix, "ram_critical_available_percent")); err != nil {
		return err
	}
	if err := validateThresholdTemp(thresholds.GPUWarnTemp, thresholdField(prefix, "gpu_warn_temp")); err != nil {
		return err
	}
	if err := validateThresholdTemp(thresholds.GPUCriticalTemp, thresholdField(prefix, "gpu_critical_temp")); err != nil {
		return err
	}
	if err := validateThresholdPercent(thresholds.VRAMWarnPercent, thresholdField(prefix, "vram_warn_percent")); err != nil {
		return err
	}
	if err := validateThresholdPercent(thresholds.VRAMCriticalPercent, thresholdField(prefix, "vram_critical_percent")); err != nil {
		return err
	}
	if err := validateThresholdPercent(thresholds.DiskWarnPercent, thresholdField(prefix, "disk_warn_percent")); err != nil {
		return err
	}
	if err := validateThresholdPercent(thresholds.DiskCriticalPercent, thresholdField(prefix, "disk_critical_percent")); err != nil {
		return err
	}

	return validateThresholdPairs(thresholds, prefix)
}

func validateThresholdPercent(value int, fieldName string) error {
	switch {
	case value < minThresholdPercent:
		return fmt.Errorf("%w: %s must be at least %d", ErrInvalidThreshold, fieldName, minThresholdPercent)
	case value > maxThresholdPercent:
		return fmt.Errorf("%w: %s must be no more than %d", ErrInvalidThreshold, fieldName, maxThresholdPercent)
	default:
		return nil
	}
}

func validateThresholdTemp(value int, fieldName string) error {
	switch {
	case value < minThresholdTempC:
		return fmt.Errorf("%w: %s must be at least %d", ErrInvalidThreshold, fieldName, minThresholdTempC)
	case value > maxThresholdTempC:
		return fmt.Errorf("%w: %s must be no more than %d", ErrInvalidThreshold, fieldName, maxThresholdTempC)
	default:
		return nil
	}
}

func validateThresholdPairs(thresholds core.Thresholds, prefix string) error {
	if thresholds.CPUWarnTemp >= thresholds.CPUCriticalTemp {
		return fmt.Errorf("%w: %s must be less than %s", ErrInvalidThreshold, thresholdField(prefix, "cpu_warn_temp"), thresholdField(prefix, "cpu_critical_temp"))
	}
	if thresholds.RAMWarnAvailablePercent <= thresholds.RAMCriticalAvailablePercent {
		return fmt.Errorf("%w: %s must be greater than %s", ErrInvalidThreshold, thresholdField(prefix, "ram_warn_available_percent"), thresholdField(prefix, "ram_critical_available_percent"))
	}
	if thresholds.GPUWarnTemp >= thresholds.GPUCriticalTemp {
		return fmt.Errorf("%w: %s must be less than %s", ErrInvalidThreshold, thresholdField(prefix, "gpu_warn_temp"), thresholdField(prefix, "gpu_critical_temp"))
	}
	if thresholds.VRAMWarnPercent >= thresholds.VRAMCriticalPercent {
		return fmt.Errorf("%w: %s must be less than %s", ErrInvalidThreshold, thresholdField(prefix, "vram_warn_percent"), thresholdField(prefix, "vram_critical_percent"))
	}
	if thresholds.DiskWarnPercent >= thresholds.DiskCriticalPercent {
		return fmt.Errorf("%w: %s must be less than %s", ErrInvalidThreshold, thresholdField(prefix, "disk_warn_percent"), thresholdField(prefix, "disk_critical_percent"))
	}

	return nil
}

func thresholdField(prefix, fieldName string) string {
	if prefix == "" {
		return fieldName
	}

	return prefix + " " + fieldName
}

func getenvDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}

	return fallback
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return parsed
}

func getenvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
