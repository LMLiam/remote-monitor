package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmliam/remote-monitor/internal/config"
	core "github.com/lmliam/remote-monitor/internal/core"
)

const (
	testExampleHost = "example-host"
	testFlagConfig  = "-config"
	testFlagOutput  = "-output"
	testFlagOut     = "-out"
	testFlagProfile = "-profile"
	testFlagTheme   = "-theme"
	testProfileName = "gpu-box"
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

func TestParseConfigAcceptsWindowsXPTheme(t *testing.T) {
	t.Setenv("REMOTE_MONITOR_HOST", "")
	t.Setenv("MONITOR_THEME", "")

	cfg, err := config.ParseConfig([]string{testFlagTheme, "winxp", testExampleHost})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Theme != core.ThemeWindowsXP {
		t.Fatalf("theme = %q", cfg.Theme)
	}
}

func TestParseConfigAcceptsWindowsXPThemeFromEnvironment(t *testing.T) {
	t.Setenv("REMOTE_MONITOR_HOST", "")
	t.Setenv("MONITOR_THEME", core.ThemeWindowsXP)

	cfg, err := config.ParseConfig([]string{testExampleHost})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Theme != core.ThemeWindowsXP {
		t.Fatalf("theme = %q", cfg.Theme)
	}
}

func TestParseConfigAcceptsOutputModes(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{core.OutputModeTUI, core.OutputModeText, core.OutputModeJSONL} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.ParseConfig([]string{testFlagOutput, mode, testExampleHost})
			if err != nil {
				t.Fatalf("ParseConfig returned error: %v", err)
			}
			if cfg.OutputMode != mode {
				t.Fatalf("output mode = %q", cfg.OutputMode)
			}
		})
	}
}

func TestParseConfigRejectsUnknownOutputMode(t *testing.T) {
	t.Setenv("REMOTE_MONITOR_HOST", "")

	_, err := config.ParseConfig([]string{testFlagOutput, "csv", testExampleHost})
	assertErrorContains(t, err, `unknown output mode "csv"`)
	assertErrorContains(t, err, "tui, text, jsonl")
}

func TestParseConfigStoresJSONLOutputPath(t *testing.T) {
	t.Setenv("REMOTE_MONITOR_HOST", "")

	cfg, err := config.ParseConfig([]string{testFlagOutput, core.OutputModeJSONL, testFlagOut, "samples.jsonl", testExampleHost})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.OutputMode != core.OutputModeJSONL {
		t.Fatalf("output mode = %q", cfg.OutputMode)
	}
	if cfg.OutputPath != "samples.jsonl" {
		t.Fatalf("output path = %q", cfg.OutputPath)
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

func TestParseConfigLoadsNamedProfileFromExplicitConfig(t *testing.T) {
	t.Parallel()

	configPath := writeConfigFile(t, `
[profiles.gpu-box]
host = "user@gpu-box"
interval = 2
history = 600
stale_after = 7
reconnect_delay = 4
fps = 24
theme = "windows-xp"
compact = true
no_banner = true
no_truecolor = true
ssh_connect_timeout = 6
ssh_server_alive = 8
ssh_server_alive_count = 3
ssh_control_persist = 45

[profiles.vps]
host = "ops@vps"
interval = 5
`)

	cfg, err := config.ParseConfig([]string{testFlagConfig, configPath, testFlagProfile, testProfileName})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Host != "user@gpu-box" {
		t.Fatalf("host = %q", cfg.Host)
	}
	if got := int(cfg.Interval.Seconds()); got != 2 {
		t.Fatalf("interval = %d", got)
	}
	if cfg.HistoryLimit != 600 {
		t.Fatalf("history limit = %d", cfg.HistoryLimit)
	}
	if got := int(cfg.StaleAfter.Seconds()); got != 7 {
		t.Fatalf("stale after = %d", got)
	}
	if got := int(cfg.ReconnectBaseDelay.Seconds()); got != 4 {
		t.Fatalf("reconnect delay = %d", got)
	}
	if cfg.RenderFPS != 24 {
		t.Fatalf("render FPS = %d", cfg.RenderFPS)
	}
	if cfg.Theme != core.ThemeWindowsXP {
		t.Fatalf("theme = %q", cfg.Theme)
	}
	if !cfg.Compact || !cfg.NoBanner || !cfg.DisableTrueColor {
		t.Fatalf("expected profile booleans to be set: %#v", cfg)
	}
	if got := int(cfg.SSHConnectTimeout.Seconds()); got != 6 {
		t.Fatalf("ssh connect timeout = %d", got)
	}
	if got := int(cfg.SSHAliveInterval.Seconds()); got != 8 {
		t.Fatalf("ssh alive interval = %d", got)
	}
	if cfg.SSHAliveCountMax != 3 {
		t.Fatalf("ssh alive count = %d", cfg.SSHAliveCountMax)
	}
	if got := int(cfg.SSHControlPersist.Seconds()); got != 45 {
		t.Fatalf("ssh control persist = %d", got)
	}
}

func TestParseConfigUsesDefaultConfigPath(t *testing.T) {
	clearConfigEnv(t)
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	configDir := filepath.Join(configRoot, "remote-monitor")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
[profiles.gpu-box]
host = "user@default-path"
interval = 3
`), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := config.ParseConfig([]string{testFlagProfile, testProfileName})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Host != "user@default-path" {
		t.Fatalf("host = %q", cfg.Host)
	}
	if got := int(cfg.Interval.Seconds()); got != 3 {
		t.Fatalf("interval = %d", got)
	}
}

func TestParseConfigUsesHomeConfigPathFallback(t *testing.T) {
	clearConfigEnv(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	configDir := filepath.Join(homeDir, ".config", "remote-monitor")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
[profiles.gpu-box]
host = "user@home-fallback"
interval = 5
`), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := config.ParseConfig([]string{testFlagProfile, testProfileName})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Host != "user@home-fallback" {
		t.Fatalf("host = %q", cfg.Host)
	}
	if got := int(cfg.Interval.Seconds()); got != 5 {
		t.Fatalf("interval = %d", got)
	}
}

func TestParseConfigAppliesCLIProfileEnvironmentPrecedence(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("REMOTE_MONITOR_HOST", "env-host")
	t.Setenv("MONITOR_INTERVAL", "9")
	t.Setenv("MONITOR_HISTORY_LIMIT", "333")
	t.Setenv("MONITOR_STALE_AFTER", "12")
	t.Setenv("MONITOR_COMPACT", "true")
	t.Setenv("MONITOR_NO_BANNER", "true")
	t.Setenv("MONITOR_THEME", core.ThemeBasic)
	configPath := writeConfigFile(t, `
[profiles.gpu-box]
host = "profile-host"
interval = 4
history = 600
theme = "windows-xp"
compact = true
no_banner = false
`)

	cfg, err := config.ParseConfig([]string{
		testFlagConfig, configPath,
		testFlagProfile, testProfileName,
		"-interval", "2",
		"-compact=false",
		"cli-host",
	})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Host != "cli-host" {
		t.Fatalf("host = %q", cfg.Host)
	}
	if got := int(cfg.Interval.Seconds()); got != 2 {
		t.Fatalf("interval = %d", got)
	}
	if cfg.HistoryLimit != 600 {
		t.Fatalf("history limit = %d", cfg.HistoryLimit)
	}
	if got := int(cfg.StaleAfter.Seconds()); got != 12 {
		t.Fatalf("stale after = %d", got)
	}
	if cfg.Theme != core.ThemeWindowsXP {
		t.Fatalf("theme = %q", cfg.Theme)
	}
	if cfg.Compact {
		t.Fatalf("compact = %t", cfg.Compact)
	}
	if cfg.NoBanner {
		t.Fatalf("no banner = %t", cfg.NoBanner)
	}
}

func TestParseConfigPreservesDirectHostInputsWithProfiles(t *testing.T) {
	//nolint:paralleltest // These subtests use t.Setenv through clearConfigEnv.
	t.Run("host flag overrides selected profile", func(t *testing.T) {
		clearConfigEnv(t)
		configPath := writeConfigFile(t, `
[profiles.gpu-box]
host = "profile-host"
interval = 4
`)

		cfg, err := config.ParseConfig([]string{
			testFlagConfig, configPath,
			testFlagProfile, testProfileName,
			"-host", "flag-host",
		})
		if err != nil {
			t.Fatalf("ParseConfig returned error: %v", err)
		}

		if cfg.Host != "flag-host" {
			t.Fatalf("host = %q", cfg.Host)
		}
		if got := int(cfg.Interval.Seconds()); got != 4 {
			t.Fatalf("interval = %d", got)
		}
	})

	t.Run("environment host fills unset profile host", func(t *testing.T) {
		clearConfigEnv(t)
		t.Setenv("REMOTE_MONITOR_HOST", "env-host")
		configPath := writeConfigFile(t, `
[profiles.gpu-box]
interval = 6
`)

		cfg, err := config.ParseConfig([]string{testFlagConfig, configPath, testFlagProfile, testProfileName})
		if err != nil {
			t.Fatalf("ParseConfig returned error: %v", err)
		}

		if cfg.Host != "env-host" {
			t.Fatalf("host = %q", cfg.Host)
		}
		if got := int(cfg.Interval.Seconds()); got != 6 {
			t.Fatalf("interval = %d", got)
		}
	})
}

func TestParseConfigReportsProfileErrors(t *testing.T) {
	t.Parallel()

	t.Run("missing profile", func(t *testing.T) {
		t.Parallel()

		configPath := writeConfigFile(t, `
[profiles.gpu-box]
host = "user@gpu-box"
`)

		_, err := config.ParseConfig([]string{testFlagConfig, configPath, testFlagProfile, "missing"})
		assertErrorContains(t, err, `profile "missing" not found`)
	})

	t.Run("invalid TOML", func(t *testing.T) {
		t.Parallel()

		configPath := writeRawConfigFile(t, "not valid toml = ]")

		_, err := config.ParseConfig([]string{testFlagConfig, configPath, testFlagProfile, testProfileName})
		assertErrorContains(t, err, "load config")
	})

	t.Run("unknown keys", func(t *testing.T) {
		t.Parallel()

		configPath := writeConfigFile(t, `
[profiles.gpu-box]
host = "user@gpu-box"
unexpected = true
`)

		_, err := config.ParseConfig([]string{testFlagConfig, configPath, testFlagProfile, testProfileName})
		assertErrorContains(t, err, "unknown config key")
		assertErrorContains(t, err, "profiles.gpu-box.unexpected")
	})

	t.Run("invalid values", func(t *testing.T) {
		t.Parallel()

		configPath := writeConfigFile(t, `
[profiles.gpu-box]
host = "user@gpu-box"
interval = 0
`)

		_, err := config.ParseConfig([]string{testFlagConfig, configPath, testFlagProfile, testProfileName})
		assertErrorContains(t, err, "profile gpu-box interval must be at least 1")
	})
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"REMOTE_MONITOR_HOST",
		"MONITOR_INTERVAL",
		"MONITOR_HISTORY_LIMIT",
		"MONITOR_STALE_AFTER",
		"MONITOR_RECONNECT_DELAY",
		"MONITOR_FPS",
		"MONITOR_COMPACT",
		"MONITOR_NO_BANNER",
		"MONITOR_THEME",
		"MONITOR_NO_TRUECOLOR",
		"MONITOR_SSH_CONNECT_TIMEOUT",
		"MONITOR_SSH_ALIVE_INTERVAL",
		"MONITOR_SSH_ALIVE_COUNT",
		"MONITOR_SSH_CONTROL_PERSIST",
		"XDG_CONFIG_HOME",
	} {
		t.Setenv(key, "")
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	return writeRawConfigFile(t, strings.TrimSpace(content)+"\n")
}

func writeRawConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err.Error(), want)
	}
}
