package config_test

import (
	"errors"
	"testing"

	"github.com/lmliam/remote-monitor/internal/config"
	core "github.com/lmliam/remote-monitor/internal/core"
)

const (
	testExampleHost = "example-host"
	testFlagTheme   = "-theme"
)

func TestParseConfigRequiresHost(t *testing.T) {
	t.Setenv("REMOTE_MONITOR_HOST", "")

	_, err := config.ParseConfig(nil)
	if !errors.Is(err, config.ErrEmptyHost) {
		t.Fatalf("error = %v", err)
	}
}

func TestParseConfigAllowsVersionWithoutHost(t *testing.T) {
	t.Setenv("REMOTE_MONITOR_HOST", "")

	cfg, err := config.ParseConfig([]string{"-version"})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if !cfg.ShowVersion {
		t.Fatalf("show version = %t", cfg.ShowVersion)
	}
}

func TestParseConfigClampsAndPositionalHost(t *testing.T) {
	t.Setenv("REMOTE_MONITOR_HOST", "")
	t.Setenv("MONITOR_INTERVAL", "")
	t.Setenv("MONITOR_HISTORY_LIMIT", "")
	t.Setenv("MONITOR_STALE_AFTER", "")
	t.Setenv("MONITOR_RECONNECT_DELAY", "")
	t.Setenv("MONITOR_FPS", "")
	t.Setenv("MONITOR_COMPACT", "")
	t.Setenv("MONITOR_NO_BANNER", "")
	t.Setenv("MONITOR_THEME", "")
	t.Setenv("MONITOR_NO_TRUECOLOR", "")
	t.Setenv("MONITOR_SSH_CONNECT_TIMEOUT", "")
	t.Setenv("MONITOR_SSH_ALIVE_INTERVAL", "")
	t.Setenv("MONITOR_SSH_ALIVE_COUNT", "")
	t.Setenv("MONITOR_SSH_CONTROL_PERSIST", "")

	cfg, err := config.ParseConfig([]string{
		"-interval", "0",
		"-history", "1",
		"-stale-after", "0",
		"-reconnect-delay", "0",
		"-fps", "0",
		testFlagTheme, "BASIC",
		"-compact",
		"-no-banner",
		"-no-truecolor",
		"-ssh-connect-timeout", "0",
		"-ssh-server-alive", "0",
		"-ssh-server-alive-count", "0",
		"-ssh-control-persist", "-1",
		testExampleHost,
	})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Host != testExampleHost {
		t.Fatalf("host = %q", cfg.Host)
	}
	if got := int(cfg.Interval.Seconds()); got != 1 {
		t.Fatalf("interval = %d", got)
	}
	if cfg.HistoryLimit != 30 {
		t.Fatalf("history limit = %d", cfg.HistoryLimit)
	}
	if got := int(cfg.StaleAfter.Seconds()); got != 3 {
		t.Fatalf("stale after = %d", got)
	}
	if got := int(cfg.ReconnectBaseDelay.Seconds()); got != 1 {
		t.Fatalf("reconnect delay = %d", got)
	}
	if cfg.RenderFPS != 1 {
		t.Fatalf("render FPS = %d", cfg.RenderFPS)
	}
	if !cfg.Compact || !cfg.NoBanner || !cfg.DisableTrueColor {
		t.Fatalf("expected compact/banner/truecolor flags to be set: %#v", cfg)
	}
	if cfg.Theme != core.ThemeBasic {
		t.Fatalf("theme = %q", cfg.Theme)
	}
	if got := int(cfg.SSHConnectTimeout.Seconds()); got != 1 {
		t.Fatalf("ssh connect timeout = %d", got)
	}
	if got := int(cfg.SSHAliveInterval.Seconds()); got != 1 {
		t.Fatalf("ssh alive interval = %d", got)
	}
	if cfg.SSHAliveCountMax != 1 {
		t.Fatalf("ssh alive count = %d", cfg.SSHAliveCountMax)
	}
	if got := int(cfg.SSHControlPersist.Seconds()); got != 0 {
		t.Fatalf("ssh control persist = %d", got)
	}
}

func TestParseConfigAcceptsBasicTheme(t *testing.T) {
	t.Setenv("REMOTE_MONITOR_HOST", "")
	t.Setenv("MONITOR_THEME", "")

	cfg, err := config.ParseConfig([]string{testFlagTheme, "BASIC", testExampleHost})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Theme != core.ThemeBasic {
		t.Fatalf("theme = %q", cfg.Theme)
	}
}

func TestParseConfigFallsBackToAuroraForUnknownThemes(t *testing.T) {
	t.Setenv("REMOTE_MONITOR_HOST", "")
	t.Setenv("MONITOR_THEME", "")

	cfg, err := config.ParseConfig([]string{testFlagTheme, "unknown-theme", testExampleHost})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Theme != core.ThemeAurora {
		t.Fatalf("theme = %q", cfg.Theme)
	}
}
