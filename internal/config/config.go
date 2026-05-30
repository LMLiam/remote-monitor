package config

import (
	"errors"
	"flag"
	core "github.com/lmliam/remote-monitor/internal/core"
	"os"
	"strconv"
	"strings"
	"time"
)

// ErrEmptyHost reports a missing SSH host in parsed configuration.
var ErrEmptyHost = errors.New("host cannot be empty")

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
	hostDefault := getenvDefault("REMOTE_MONITOR_HOST", "")
	intervalDefault := getenvInt("MONITOR_INTERVAL", 1)
	historyDefault := getenvInt("MONITOR_HISTORY_LIMIT", defaultHistoryLimit)
	staleDefault := getenvInt("MONITOR_STALE_AFTER", intervalDefault*3+1)
	reconnectDefault := getenvInt("MONITOR_RECONNECT_DELAY", defaultReconnectDelaySecs)
	fpsDefault := getenvInt("MONITOR_FPS", defaultRenderFPS)
	compactDefault := getenvBool("MONITOR_COMPACT", false)
	noBannerDefault := getenvBool("MONITOR_NO_BANNER", false)
	themeDefault := getenvDefault("MONITOR_THEME", core.ThemeAurora)
	noTrueColorDefault := getenvBool("MONITOR_NO_TRUECOLOR", false)
	sshConnectTimeoutDefault := getenvInt("MONITOR_SSH_CONNECT_TIMEOUT", defaultSSHTimeoutSecs)
	sshAliveIntervalDefault := getenvInt("MONITOR_SSH_ALIVE_INTERVAL", defaultSSHTimeoutSecs)
	sshAliveCountDefault := getenvInt("MONITOR_SSH_ALIVE_COUNT", defaultSSHAliveCount)
	sshControlPersistDefault := getenvInt("MONITOR_SSH_CONTROL_PERSIST", defaultSSHControlSecs)

	fs := flag.NewFlagSet("remote-monitor", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	host := fs.String("host", hostDefault, "SSH host to monitor")
	interval := fs.Int("interval", intervalDefault, "Refresh interval in seconds")
	history := fs.Int("history", historyDefault, "History Sample limit")
	staleAfter := fs.Int("stale-after", staleDefault, "Seconds before live data is considered stale")
	reconnectDelay := fs.Int("reconnect-delay", reconnectDefault, "Base reconnect delay in seconds")
	fps := fs.Int("fps", fpsDefault, "TTY redraw frames per second")
	compact := fs.Bool("compact", compactDefault, "Use a compact stacked layout")
	noBanner := fs.Bool("no-banner", noBannerDefault, "Disable the large rendered title banner")
	theme := fs.String("theme", themeDefault, "Color theme (aurora, basic)")
	noTrueColor := fs.Bool("no-truecolor", noTrueColorDefault, "Force 256-color rendering even on truecolor terminals")
	sshConnectTimeout := fs.Int("ssh-connect-timeout", sshConnectTimeoutDefault, "SSH connect timeout in seconds")
	sshAliveInterval := fs.Int("ssh-server-alive", sshAliveIntervalDefault, "SSH keepalive interval in seconds")
	sshAliveCount := fs.Int("ssh-server-alive-count", sshAliveCountDefault, "SSH keepalive failure threshold before reconnect")
	sshControlPersist := fs.Int("ssh-control-persist", sshControlPersistDefault, "SSH control socket persist time in seconds")

	if err := fs.Parse(args); err != nil {
		return core.Config{}, err
	}

	if fs.NArg() > 0 {
		*host = fs.Arg(0)
	}
	if strings.TrimSpace(*host) == "" {
		return core.Config{}, ErrEmptyHost
	}
	if *interval < 1 {
		*interval = 1
	}
	if *history < minHistoryLimit {
		*history = minHistoryLimit
	}
	if *staleAfter < minStaleAfterSecs {
		*staleAfter = minStaleAfterSecs
	}
	if *reconnectDelay < 1 {
		*reconnectDelay = 1
	}
	if *fps < 1 {
		*fps = 1
	}
	if *fps > maxRenderFPS {
		*fps = maxRenderFPS
	}
	if *sshConnectTimeout < 1 {
		*sshConnectTimeout = 1
	}
	if *sshAliveInterval < 1 {
		*sshAliveInterval = 1
	}
	if *sshAliveCount < 1 {
		*sshAliveCount = 1
	}
	if *sshControlPersist < 0 {
		*sshControlPersist = 0
	}
	*theme = core.CanonicalThemeName(*theme)

	return core.Config{
		Host:               *host,
		Interval:           time.Duration(*interval) * time.Second,
		HistoryLimit:       *history,
		StaleAfter:         time.Duration(*staleAfter) * time.Second,
		ReconnectBaseDelay: time.Duration(*reconnectDelay) * time.Second,
		RenderFPS:          *fps,
		Compact:            *compact,
		NoBanner:           *noBanner,
		Theme:              *theme,
		DisableTrueColor:   *noTrueColor,
		SSHConnectTimeout:  time.Duration(*sshConnectTimeout) * time.Second,
		SSHAliveInterval:   time.Duration(*sshAliveInterval) * time.Second,
		SSHAliveCountMax:   *sshAliveCount,
		SSHControlPersist:  time.Duration(*sshControlPersist) * time.Second,
		SSHControlPath:     "",
	}, nil
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
