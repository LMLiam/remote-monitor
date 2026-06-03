package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
)

// ErrEmptyHost reports a missing SSH host in parsed configuration.
var ErrEmptyHost = errors.New("host cannot be empty")

// ErrUnknownOutputMode reports an unsupported -output value.
var ErrUnknownOutputMode = errors.New("unknown output mode")

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
)

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
	outputMode := fs.String("output", core.OutputModeAuto, "Output mode (tui, text, jsonl)")
	outputPath := fs.String("out", "", "Write JSONL output to this file")
	theme := fs.String("theme", cliValues.theme, "Color theme (aurora, basic, windows-xp)")
	noTrueColor := fs.Bool("no-truecolor", cliValues.noTrueColor, "Force 256-color rendering even on truecolor terminals")
	sshConnectTimeout := fs.Int("ssh-connect-timeout", cliValues.sshConnectTimeout, "SSH connect timeout in seconds")
	sshAliveInterval := fs.Int("ssh-server-alive", cliValues.sshAliveInterval, "SSH keepalive interval in seconds")
	sshAliveCount := fs.Int("ssh-server-alive-count", cliValues.sshAliveCount, "SSH keepalive failure threshold before reconnect")
	sshControlPersist := fs.Int("ssh-control-persist", cliValues.sshControlPersist, "SSH control socket persist time in seconds")

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

	cliValues = configValues{
		host:              *host,
		interval:          *interval,
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

	applyExplicitFlags(&resolved, cliValues, visitedFlags(fs))
	if fs.NArg() > 0 {
		resolved.host = fs.Arg(0)
	}
	if strings.TrimSpace(resolved.host) == "" {
		return core.Config{}, ErrEmptyHost
	}

	resolved.clamp()
	resolved.theme = core.CanonicalThemeName(resolved.theme)

	return core.Config{
		Host:               resolved.host,
		Interval:           time.Duration(resolved.interval) * time.Second,
		HistoryLimit:       resolved.history,
		StaleAfter:         time.Duration(resolved.staleAfter) * time.Second,
		ReconnectBaseDelay: time.Duration(resolved.reconnectDelay) * time.Second,
		RenderFPS:          resolved.fps,
		Compact:            resolved.compact,
		NoBanner:           resolved.noBanner,
		ShowVersion:        false,
		OutputMode:         resolvedOutputMode,
		OutputPath:         *outputPath,
		Theme:              resolved.theme,
		DisableTrueColor:   resolved.noTrueColor,
		SSHConnectTimeout:  time.Duration(resolved.sshConnectTimeout) * time.Second,
		SSHAliveInterval:   time.Duration(resolved.sshAliveInterval) * time.Second,
		SSHAliveCountMax:   resolved.sshAliveCount,
		SSHControlPersist:  time.Duration(resolved.sshControlPersist) * time.Second,
		SSHControlPath:     "",
	}, nil
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

type configValues struct {
	host              string
	interval          int
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
}

func configValuesFromEnv() configValues {
	intervalDefault := getenvInt("MONITOR_INTERVAL", 1)

	return configValues{
		host:              getenvDefault("REMOTE_MONITOR_HOST", ""),
		interval:          intervalDefault,
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
