package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	core "github.com/lmliam/remote-monitor/internal/core"
)

var (
	errProfileHostEmpty        = errors.New("profile host cannot be empty")
	errProfileNotFound         = errors.New("profile not found")
	errProfileThemeUnsupported = errors.New("profile theme unsupported")
	errProfileValueTooHigh     = errors.New("profile value above maximum")
	errProfileValueTooLow      = errors.New("profile value below minimum")
	errUnknownConfigKey        = errors.New("unknown config key")
)

type configFile struct {
	Profiles map[string]profileConfig `toml:"profiles"`
}

type profileConfig struct {
	Host              *string `toml:"host"`
	Interval          *int    `toml:"interval"`
	History           *int    `toml:"history"`
	StaleAfter        *int    `toml:"stale_after"`
	ReconnectDelay    *int    `toml:"reconnect_delay"`
	FPS               *int    `toml:"fps"`
	Theme             *string `toml:"theme"`
	Compact           *bool   `toml:"compact"`
	NoBanner          *bool   `toml:"no_banner"`
	NoTrueColor       *bool   `toml:"no_truecolor"`
	SSHConnectTimeout *int    `toml:"ssh_connect_timeout"`
	SSHServerAlive    *int    `toml:"ssh_server_alive"`
	SSHServerCount    *int    `toml:"ssh_server_alive_count"`
	SSHControlPersist *int    `toml:"ssh_control_persist"`
}

func loadProfile(path, name string) (profileConfig, error) {
	resolvedPath := expandConfigPath(strings.TrimSpace(path))
	var parsed configFile
	metadata, err := toml.DecodeFile(resolvedPath, &parsed)
	if err != nil {
		return profileConfig{}, fmt.Errorf("load config %s: %w", resolvedPath, err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return profileConfig{}, fmt.Errorf("%w %q in %s", errUnknownConfigKey, dottedKey(undecoded[0]), resolvedPath)
	}

	profile, ok := parsed.Profiles[name]
	if !ok {
		return profileConfig{}, fmt.Errorf("%w: profile %q not found in %s", errProfileNotFound, name, resolvedPath)
	}

	return profile, nil
}

func applyProfile(values *configValues, profile profileConfig, name string) error {
	if profile.Host != nil {
		if strings.TrimSpace(*profile.Host) == "" {
			return fmt.Errorf("%w: profile %s host cannot be empty", errProfileHostEmpty, name)
		}
		values.host = *profile.Host
	}
	if err := applyProfileInt(profile.Interval, 1, 0, name, "interval", &values.interval); err != nil {
		return err
	}
	if err := applyProfileInt(profile.History, minHistoryLimit, 0, name, "history", &values.history); err != nil {
		return err
	}
	if err := applyProfileInt(profile.StaleAfter, minStaleAfterSecs, 0, name, "stale_after", &values.staleAfter); err != nil {
		return err
	}
	if err := applyProfileInt(profile.ReconnectDelay, 1, 0, name, "reconnect_delay", &values.reconnectDelay); err != nil {
		return err
	}
	if err := applyProfileInt(profile.FPS, 1, maxRenderFPS, name, "fps", &values.fps); err != nil {
		return err
	}
	if profile.Theme != nil {
		theme, err := profileTheme(*profile.Theme, name)
		if err != nil {
			return err
		}
		values.theme = theme
	}
	if profile.Compact != nil {
		values.compact = *profile.Compact
	}
	if profile.NoBanner != nil {
		values.noBanner = *profile.NoBanner
	}
	if profile.NoTrueColor != nil {
		values.noTrueColor = *profile.NoTrueColor
	}
	if err := applyProfileInt(profile.SSHConnectTimeout, 1, 0, name, "ssh_connect_timeout", &values.sshConnectTimeout); err != nil {
		return err
	}
	if err := applyProfileInt(profile.SSHServerAlive, 1, 0, name, "ssh_server_alive", &values.sshAliveInterval); err != nil {
		return err
	}
	if err := applyProfileInt(profile.SSHServerCount, 1, 0, name, "ssh_server_alive_count", &values.sshAliveCount); err != nil {
		return err
	}
	if err := applyProfileInt(profile.SSHControlPersist, 0, 0, name, "ssh_control_persist", &values.sshControlPersist); err != nil {
		return err
	}

	return nil
}

func applyProfileInt(value *int, minValue, maxValue int, profileName, fieldName string, target *int) error {
	if value == nil {
		return nil
	}
	if *value < minValue {
		return fmt.Errorf("%w: profile %s %s must be at least %d", errProfileValueTooLow, profileName, fieldName, minValue)
	}
	if maxValue > 0 && *value > maxValue {
		return fmt.Errorf("%w: profile %s %s must be no more than %d", errProfileValueTooHigh, profileName, fieldName, maxValue)
	}
	*target = *value

	return nil
}

func profileTheme(value, profileName string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "default", core.ThemeAurora, core.ThemeBasic, core.ThemeWindowsXP, "xp", "winxp":
		return core.CanonicalThemeName(value), nil
	default:
		return "", fmt.Errorf("%w: profile %s theme %q is not supported", errProfileThemeUnsupported, profileName, value)
	}
}

func defaultConfigPath() string {
	if configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); configHome != "" {
		return filepath.Join(configHome, "remote-monitor", "config.toml")
	}
	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		return filepath.Join(home, ".config", "remote-monitor", "config.toml")
	}

	return filepath.Join(".config", "remote-monitor", "config.toml")
}

func expandConfigPath(path string) string {
	if path == "" {
		return defaultConfigPath()
	}
	if path == "~" {
		if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
			return filepath.Join(home, path[2:])
		}
	}

	return path
}

func dottedKey(key []string) string {
	return strings.Join(key, ".")
}
