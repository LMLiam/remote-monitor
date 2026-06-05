package transport_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/transport"
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
