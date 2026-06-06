package transport_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/transport"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testHost = "gpu-wsl"

func testConfig(overrides ...func(*core.Config)) core.Config {
	cfg := core.Config{
		Host:               "",
		Interval:           0,
		ProcessSort:        "",
		ProcessFilter:      "",
		ProcessCount:       0,
		NetIncludePatterns: nil,
		NetExcludePatterns: nil,
		NetAggregate:       false,
		HistoryLimit:       0,
		StaleAfter:         0,
		ReconnectBaseDelay: 0,
		RenderFPS:          0,
		Compact:            false,
		NoBanner:           false,
		ShowVersion:        false,
		Once:               false,
		OutputMode:         "",
		OutputPath:         "",
		Theme:              "",
		DisableTrueColor:   false,
		SSHConnectTimeout:  0,
		SSHAliveInterval:   0,
		SSHAliveCountMax:   0,
		SSHControlPersist:  0,
		SSHControlPath:     "",
		Thresholds:         core.DefaultThresholds(),
	}
	for _, override := range overrides {
		override(&cfg)
	}

	return cfg
}

func TestSSHArgsIncludeKeepaliveTimeoutAndControlSocket(t *testing.T) {
	t.Parallel()

	cfg := testConfig(func(cfg *core.Config) {
		cfg.Host = testHost
		cfg.SSHConnectTimeout = 7 * time.Second
		cfg.SSHAliveInterval = 5 * time.Second
		cfg.SSHAliveCountMax = 3
		cfg.SSHControlPersist = 45 * time.Second
	})

	args := transport.SSHArgs(cfg, 2)
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"-T",
		"BatchMode=yes",
		"ConnectTimeout=7",
		"ServerAliveInterval=5",
		"ServerAliveCountMax=3",
		"TCPKeepAlive=yes",
		"ControlMaster=auto",
		"ControlPersist=45s",
		"ControlPath=",
		"gpu-wsl bash -s -- 2",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("ssh args missing %q: %q", want, joined)
		}
	}
}

func TestSSHArgsPassProcessOptionsToSampler(t *testing.T) {
	t.Parallel()

	cfg := testConfig(func(cfg *core.Config) {
		cfg.Host = testHost
		cfg.ProcessSort = core.ProcessSortMemory
		cfg.ProcessFilter = "python worker"
		cfg.ProcessCount = 15
	})

	args := transport.SSHArgs(cfg, 2)
	got := args[len(args)-8:]
	want := []string{
		testHost,
		"bash",
		"-s",
		"--",
		"2",
		core.ProcessSortMemory,
		"'python worker'",
		"15",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ssh sampler arg %d = %q, want %q in %#v", i, got[i], want[i], args)
		}
	}
}

func TestResolveSSHControlPathUsesStablePerProcessHostSocket(t *testing.T) {
	t.Parallel()

	cfg := testConfig(func(cfg *core.Config) { cfg.Host = testHost })
	first := transport.ResolveSSHControlPath(cfg)
	second := transport.ResolveSSHControlPath(cfg)
	if first != second {
		t.Fatalf("control path changed across calls: %q vs %q", first, second)
	}
	if !strings.HasSuffix(first, ".sock") {
		t.Fatalf("control path missing socket suffix: %q", first)
	}
	if len(first) >= 100 {
		t.Fatalf("control path too long for portable unix sockets: %q", first)
	}
}

//nolint:paralleltest,nolintlint // t.Setenv requires serial execution; paralleltest emits no diagnostic here.
func TestSSHControlPathDirs(t *testing.T) {
	cfg := testConfig(func(cfg *core.Config) { cfg.Host = testHost })
	t.Setenv("TMPDIR", "/tmp")

	t.Run("xdg", func(t *testing.T) { assertXDGControlPath(t, cfg) })
	t.Run("mode", func(t *testing.T) { assertExistingControlPathMode(t, cfg) })
	t.Run("home", func(t *testing.T) { assertHomeControlPath(t, cfg) })
	t.Run("tmp", func(t *testing.T) { assertTmpControlPath(t, cfg) })
	t.Run("bad xdg", func(t *testing.T) { assertBadXDGControlPath(t, cfg) })
	t.Run("rel", func(t *testing.T) { assertRelativeXDGControlPath(t, cfg) })
	t.Run("long home", func(t *testing.T) { assertLongHomeControlPath(t, cfg) })
	t.Run("override", assertOverrideControlPath)
}

func assertXDGControlPath(t *testing.T, cfg core.Config) {
	t.Helper()

	xdgRuntimeDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", xdgRuntimeDir)
	t.Setenv("HOME", homeDir)

	controlPath := transport.ResolveSSHControlPath(cfg)
	controlDir := filepath.Join(xdgRuntimeDir, "remote-monitor")

	assertControlPathUnder(t, controlPath, controlDir)
	assertControlDirMode(t, controlDir)
	assertPortableControlPath(t, controlPath)
}

func assertExistingControlPathMode(t *testing.T, cfg core.Config) {
	t.Helper()

	xdgRuntimeDir := t.TempDir()
	homeDir := t.TempDir()
	controlDir := filepath.Join(xdgRuntimeDir, "remote-monitor")
	//nolint:gosec // G301: this fixture intentionally starts permissive to verify chmod hardening.
	if err := os.MkdirAll(controlDir, 0o755); err != nil {
		t.Fatalf("create permissive control dir: %v", err)
	}
	//nolint:gosec // G302: this fixture intentionally starts permissive to verify chmod hardening.
	if err := os.Chmod(controlDir, 0o755); err != nil {
		t.Fatalf("chmod permissive control dir: %v", err)
	}
	assertControlDirPerm(t, controlDir, 0o755)
	t.Setenv("XDG_RUNTIME_DIR", xdgRuntimeDir)
	t.Setenv("HOME", homeDir)

	controlPath := transport.ResolveSSHControlPath(cfg)

	assertControlPathUnder(t, controlPath, controlDir)
	assertControlDirMode(t, controlDir)
	assertPortableControlPath(t, controlPath)
}

func assertHomeControlPath(t *testing.T, cfg core.Config) {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("HOME", homeDir)

	controlPath := transport.ResolveSSHControlPath(cfg)
	controlDir := filepath.Join(homeDir, ".cache", "remote-monitor")

	assertControlPathUnder(t, controlPath, controlDir)
	assertControlDirMode(t, controlDir)
	assertPortableControlPath(t, controlPath)
}

func assertTmpControlPath(t *testing.T, cfg core.Config) {
	t.Helper()

	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("HOME", "")

	controlPath := transport.ResolveSSHControlPath(cfg)

	assertControlPathInDir(t, controlPath, "/tmp")
	assertPortableControlPath(t, controlPath)
}

func assertBadXDGControlPath(t *testing.T, cfg core.Config) {
	t.Helper()

	badRuntimeDir := filepath.Join(t.TempDir(), "runtime")
	homeDir := t.TempDir()
	if err := os.WriteFile(badRuntimeDir, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write runtime dir placeholder: %v", err)
	}
	t.Setenv("XDG_RUNTIME_DIR", badRuntimeDir)
	t.Setenv("HOME", homeDir)

	controlPath := transport.ResolveSSHControlPath(cfg)
	controlDir := filepath.Join(homeDir, ".cache", "remote-monitor")

	assertControlPathUnder(t, controlPath, controlDir)
	assertControlDirMode(t, controlDir)
	assertPortableControlPath(t, controlPath)
}

func assertRelativeXDGControlPath(t *testing.T, cfg core.Config) {
	t.Helper()

	homeDir := t.TempDir()
	relativeRuntimeDir := "relative-runtime"
	t.Setenv("XDG_RUNTIME_DIR", relativeRuntimeDir)
	t.Setenv("HOME", homeDir)

	controlPath := transport.ResolveSSHControlPath(cfg)
	controlDir := filepath.Join(homeDir, ".cache", "remote-monitor")

	assertControlPathUnder(t, controlPath, controlDir)
	assertControlDirMode(t, controlDir)
	assertPortableControlPath(t, controlPath)
	assertDirNotCreated(t, filepath.Join(relativeRuntimeDir, "remote-monitor"))
}

func assertLongHomeControlPath(t *testing.T, cfg core.Config) {
	t.Helper()

	homeDir := filepath.Join(t.TempDir(), strings.Repeat("x", 80))
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("HOME", homeDir)

	controlPath := transport.ResolveSSHControlPath(cfg)

	assertControlPathInDir(t, controlPath, "/tmp")
	assertPortableControlPath(t, controlPath)
}

func assertOverrideControlPath(t *testing.T) {
	t.Helper()

	xdgRuntimeDir := filepath.Join(t.TempDir(), "runtime")
	homeDir := filepath.Join(t.TempDir(), "home")
	override := "/custom/control.sock"
	overrideCfg := testConfig(func(cfg *core.Config) {
		cfg.Host = testHost
		cfg.SSHControlPath = override
	})
	t.Setenv("XDG_RUNTIME_DIR", xdgRuntimeDir)
	t.Setenv("HOME", homeDir)

	controlPath := transport.ResolveSSHControlPath(overrideCfg)

	if controlPath != override {
		t.Fatalf("control path = %q, want explicit override %q", controlPath, override)
	}
	assertDirNotCreated(t, filepath.Join(xdgRuntimeDir, "remote-monitor"))
	assertDirNotCreated(t, filepath.Join(homeDir, ".cache", "remote-monitor"))
}

func assertControlPathUnder(t *testing.T, controlPath, controlDir string) {
	t.Helper()

	wantPrefix := filepath.Clean(controlDir) + string(os.PathSeparator)
	if !strings.HasPrefix(controlPath, wantPrefix) {
		t.Fatalf("control path = %q, want prefix %q", controlPath, wantPrefix)
	}
}

func assertControlPathInDir(t *testing.T, controlPath, controlDir string) {
	t.Helper()

	if got, want := filepath.Dir(controlPath), filepath.Clean(controlDir); got != want {
		t.Fatalf("control path dir = %q, want %q for path %q", got, want, controlPath)
	}
}

func assertControlDirMode(t *testing.T, controlDir string) {
	t.Helper()

	assertControlDirPerm(t, controlDir, 0o700)
}

func assertControlDirPerm(t *testing.T, controlDir string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(controlDir)
	if err != nil {
		t.Fatalf("stat control dir %q: %v", controlDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("control dir %q is not a directory", controlDir)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("control dir %q mode = %04o, want %04o", controlDir, got, want)
	}
}

func assertPortableControlPath(t *testing.T, controlPath string) {
	t.Helper()

	if !strings.HasSuffix(controlPath, ".sock") {
		t.Fatalf("control path missing socket suffix: %q", controlPath)
	}
	if len(controlPath) >= 100 {
		t.Fatalf("control path too long for portable unix sockets: %q", controlPath)
	}
}

func assertDirNotCreated(t *testing.T, controlDir string) {
	t.Helper()

	_, err := os.Stat(controlDir)
	if err == nil {
		t.Fatalf("control dir %q was created despite explicit override", controlDir)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("stat control dir %q: %v", controlDir, err)
	}
}
