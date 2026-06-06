//nolint:testpackage // These tests exercise unexported run-loop dependencies to avoid opening SSH.
package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
)

const (
	outputIfaceDocker0 = "docker0"
	outputIfaceEth0    = "eth0"
	outputIfaceWlan0   = "wlan0"
)

func TestRunWritesJSONLToStdoutWithoutLifecycleText(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	sample := outputTestSample()
	secondSample := outputTestSample()
	secondSample.CPUPercent = 42
	secondSample.ReceivedAt = sample.ReceivedAt.Add(time.Second)
	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.OutputMode = core.OutputModeJSONL
	}), runDependencies{
		stdout:       &out,
		stdoutIsTTY:  func() bool { return false },
		preflightSSH: outputTestSSHPreflightOK,
		runStream: func(_ context.Context, _ core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
			defer close(sampleCh)
			defer close(eventCh)
			eventCh <- outputTestStreamEvent(func(ev *core.StreamEvent) {
				ev.State = core.StatusConnecting
				ev.Detail = core.DetailOpeningSSHSession
				ev.At = sample.ReceivedAt.Add(-time.Second)
			})
			sampleCh <- sample
			eventCh <- outputTestStreamEvent(func(ev *core.StreamEvent) {
				ev.State = core.StatusLive
				ev.Detail = core.DetailStreamHealthy
				ev.At = secondSample.ReceivedAt.Add(-time.Millisecond)
			})
			sampleCh <- secondSample
		},
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	lines := nonEmptyLines(out.String())
	if len(lines) != 2 {
		t.Fatalf("jsonl lines = %#v", lines)
	}
	assertJSONLineHasSample(t, lines[0], sample)
	assertJSONLineHasSample(t, lines[1], secondSample)
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("JSONL stdout does not end with newline: %q", out.String())
	}
	for _, forbidden := range []string{"opening ssh session", "stream healthy", "samples", "\x1b["} {
		if strings.Contains(out.String(), forbidden) {
			t.Fatalf("JSONL stdout contains %q: %q", forbidden, out.String())
		}
	}
}

func TestRunWritesJSONLToFileAndKeepsStdoutEmpty(t *testing.T) {
	t.Parallel()

	outputPath := filepath.Join(t.TempDir(), "samples.jsonl")
	if err := os.WriteFile(outputPath, []byte("stale contents\n"), 0o600); err != nil {
		t.Fatalf("prewrite jsonl output: %v", err)
	}
	var out bytes.Buffer
	sample := outputTestSample()

	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.OutputMode = core.OutputModeJSONL
		cfg.OutputPath = outputPath
	}), runDependencies{
		stdout:       &out,
		stdoutIsTTY:  func() bool { return false },
		preflightSSH: outputTestSSHPreflightOK,
		runStream: func(_ context.Context, _ core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
			defer close(sampleCh)
			defer close(eventCh)
			sampleCh <- sample
		},
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want empty when -out is used", out.String())
	}

	// #nosec G304 -- outputPath is created inside this test's temporary directory.
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read jsonl output: %v", err)
	}
	lines := nonEmptyLines(string(content))
	if len(lines) != 1 {
		t.Fatalf("jsonl file lines = %#v", lines)
	}
	assertJSONLineHasSample(t, lines[0], sample)
	if strings.Contains(string(content), "stale contents") {
		t.Fatalf("jsonl output file was not truncated: %q", string(content))
	}
	if !strings.HasSuffix(string(content), "\n") {
		t.Fatalf("JSONL file does not end with newline: %q", string(content))
	}
}

func TestRunWritesJSONLWithSelectedAggregateNetwork(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	sample := outputTestSample()
	sample.Net = outputSelectionNetStats()

	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.OutputMode = core.OutputModeJSONL
		cfg.NetIncludePatterns = []string{outputIfaceEth0, outputIfaceWlan0}
		cfg.NetAggregate = true
	}), runDependencies{
		stdout:       &out,
		stdoutIsTTY:  func() bool { return false },
		preflightSSH: outputTestSSHPreflightOK,
		runStream: func(_ context.Context, _ core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
			defer close(sampleCh)
			defer close(eventCh)
			sampleCh <- sample
		},
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	lines := nonEmptyLines(out.String())
	if len(lines) != 1 {
		t.Fatalf("jsonl lines = %#v", lines)
	}
	assertJSONLineHasAggregateNet(t, lines[0], 400, 50)
}

func TestResolveOutputModeKeepsTTYAndNonTTYDefaults(t *testing.T) {
	t.Parallel()

	if got := resolveOutputMode(outputTestConfig(func(cfg *core.Config) {
		cfg.OutputMode = core.OutputModeAuto
	}), true); got != core.OutputModeTUI {
		t.Fatalf("TTY default output mode = %q", got)
	}
	if got := resolveOutputMode(outputTestConfig(func(cfg *core.Config) {
		cfg.OutputMode = core.OutputModeAuto
	}), false); got != core.OutputModeText {
		t.Fatalf("non-TTY default output mode = %q", got)
	}
	if got := resolveOutputMode(outputTestConfig(func(cfg *core.Config) {
		cfg.OutputMode = core.OutputModeText
	}), true); got != core.OutputModeText {
		t.Fatalf("forced text output mode = %q", got)
	}
	if got := resolveOutputMode(outputTestConfig(func(cfg *core.Config) {
		cfg.OutputMode = core.OutputModeJSONL
	}), true); got != core.OutputModeJSONL {
		t.Fatalf("forced jsonl output mode = %q", got)
	}
}

func TestRunReturnsFileErrorBeforeStartingStream(t *testing.T) {
	t.Parallel()

	missingDirPath := filepath.Join(t.TempDir(), "missing", "samples.jsonl")
	streamStarted := false

	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.OutputMode = core.OutputModeJSONL
		cfg.OutputPath = missingDirPath
	}), runDependencies{
		stdout:       &bytes.Buffer{},
		stdoutIsTTY:  func() bool { return false },
		preflightSSH: outputTestSSHPreflightOK,
		runStream: func(context.Context, core.Config, chan<- core.Sample, chan<- core.StreamEvent) {
			streamStarted = true
		},
	})

	if err == nil {
		t.Fatal("expected file creation error")
	}
	if streamStarted {
		t.Fatal("stream started before output file error was returned")
	}
}

func TestRunJSONLRejectsOutForTextMode(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.OutputMode = core.OutputModeText
		cfg.OutputPath = filepath.Join(t.TempDir(), "samples.jsonl")
	}), runDependencies{
		stdout:       &bytes.Buffer{},
		stdoutIsTTY:  func() bool { return false },
		preflightSSH: outputTestSSHPreflightOK,
		runStream: func(context.Context, core.Config, chan<- core.Sample, chan<- core.StreamEvent) {
			t.Fatal("stream should not start for invalid output configuration")
		},
	})
	if err == nil {
		t.Fatal("expected -out validation error")
	}
	if !strings.Contains(err.Error(), "-out requires -output jsonl") {
		t.Fatalf("error = %q", err.Error())
	}
}

func outputTestConfig(overrides ...func(*core.Config)) core.Config {
	cfg := core.Config{
		Host:               "example-host",
		Interval:           time.Second,
		ProcessSort:        "",
		ProcessFilter:      "",
		ProcessCount:       0,
		NetIncludePatterns: nil,
		NetExcludePatterns: nil,
		NetAggregate:       false,
		HistoryLimit:       30,
		StaleAfter:         4 * time.Second,
		ReconnectBaseDelay: time.Second,
		RenderFPS:          12,
		Compact:            false,
		NoBanner:           false,
		ShowVersion:        false,
		Once:               false,
		OutputMode:         core.OutputModeAuto,
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

func outputTestSample() core.Sample {
	smp := core.EmptySample()
	smp.RemoteEpoch = 1716912345
	smp.RemoteTimestamp = "2026-05-28 19:35:45"
	smp.RemoteName = "gpu-box"
	smp.UptimeSeconds = 14340
	smp.Load1 = 10.55
	smp.Load5 = 4.82
	smp.Load15 = 4.09
	smp.CPUCores = 12
	smp.CPUName = "AMD Ryzen 5 5600X"
	smp.CPUPercent = 99
	smp.CPUUserPercent = 71
	smp.CPUSystemPercent = 19
	smp.CPUIOWaitPercent = 6
	smp.CPUStealPercent = 1
	smp.RAMUsedMiB = 2455
	smp.RAMTotalMiB = 15967
	smp.RAMAvailableMiB = 13512
	smp.DiskDevice = "sdd"
	smp.DiskReadBps = 1048576
	smp.DiskWriteBps = 524288
	smp.Net = []core.NetStat{outputTestNetStat()}
	smp.GPUs = []core.GPUStat{outputTestGPUStat()}
	smp.ReceivedAt = time.Unix(1716912346, 0).UTC()

	return smp
}

func outputTestStreamEvent(overrides ...func(*core.StreamEvent)) core.StreamEvent {
	ev := core.StreamEvent{
		State:          "",
		Detail:         "",
		ReconnectCount: 0,
		Attempts:       0,
		StreamAlive:    false,
		NextRetry:      time.Time{},
		At:             time.Time{},
	}
	for _, override := range overrides {
		override(&ev)
	}

	return ev
}

func outputTestNetStat() core.NetStat {
	return core.NetStat{
		Iface:      "eth0",
		RXBps:      125000,
		TXBps:      24000,
		RXPps:      0,
		TXPps:      0,
		SpeedMbps:  1000,
		RXDrops:    0,
		RXErrors:   0,
		RXOverruns: 0,
		TXDrops:    0,
		TXErrors:   0,
		TXOverruns: 0,
	}
}

func outputSelectionNetStats() []core.NetStat {
	return []core.NetStat{
		{
			Iface:      outputIfaceEth0,
			RXBps:      100,
			TXBps:      20,
			RXPps:      10,
			TXPps:      2,
			SpeedMbps:  1000,
			RXDrops:    0,
			RXErrors:   0,
			RXOverruns: 0,
			TXDrops:    0,
			TXErrors:   0,
			TXOverruns: 0,
		},
		{
			Iface:      outputIfaceWlan0,
			RXBps:      300,
			TXBps:      30,
			RXPps:      30,
			TXPps:      3,
			SpeedMbps:  100,
			RXDrops:    0,
			RXErrors:   0,
			RXOverruns: 0,
			TXDrops:    0,
			TXErrors:   0,
			TXOverruns: 0,
		},
		{
			Iface:      outputIfaceDocker0,
			RXBps:      1000,
			TXBps:      200,
			RXPps:      100,
			TXPps:      20,
			SpeedMbps:  -1,
			RXDrops:    0,
			RXErrors:   0,
			RXOverruns: 0,
			TXDrops:    0,
			TXErrors:   0,
			TXOverruns: 0,
		},
	}
}

func outputTestGPUStat() core.GPUStat {
	return core.GPUStat{
		Index:            0,
		UUID:             "GPU-123",
		Name:             "NVIDIA GeForce RTX 3060",
		Util:             82,
		MemUtil:          0,
		EncoderUtil:      0,
		DecoderUtil:      0,
		MemUsed:          2003,
		MemTotal:         12288,
		Temp:             55,
		PowerDraw:        0,
		PowerLimit:       0,
		Fan:              0,
		SMClock:          0,
		MaxSMClock:       0,
		MemClock:         0,
		MaxMemClock:      0,
		GraphicsClock:    0,
		VideoClock:       0,
		PCIeGenCurrent:   0,
		PCIeGenMax:       0,
		PCIeWidthCurrent: 0,
		PCIeWidthMax:     0,
		ThrottleReasons:  "",
		PState:           "",
	}
}

func nonEmptyLines(s string) []string {
	raw := strings.Split(s, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

func assertJSONLineHasSample(t *testing.T, line string, sample core.Sample) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("invalid JSON line %q: %v", line, err)
	}
	if got["schema"] != "remote-monitor.normalized_sample.v1" {
		t.Fatalf("schema = %#v", got["schema"])
	}
	if got["remote_name"] != sample.RemoteName {
		t.Fatalf("remote_name = %#v", got["remote_name"])
	}
	if got["cpu_percent"] != float64(sample.CPUPercent) {
		t.Fatalf("cpu_percent = %#v", got["cpu_percent"])
	}
	if got["received_at"] != sample.ReceivedAt.Format(time.RFC3339Nano) {
		t.Fatalf("received_at = %#v", got["received_at"])
	}
	if _, ok := got["RemoteName"]; ok {
		t.Fatalf("found Go field name in JSON output: %#v", got)
	}
	if _, ok := got["RuntimeState"]; ok {
		t.Fatalf("found lifecycle state in JSON output: %#v", got)
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, []byte(line)); err != nil {
		t.Fatalf("compact JSON line: %v", err)
	}
}

func assertJSONLineHasAggregateNet(t *testing.T, line string, wantRXBps, wantTXBps float64) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("invalid JSON line %q: %v", line, err)
	}
	net, ok := got["net"].([]any)
	if !ok || len(net) != 1 {
		t.Fatalf("net = %#v, want one aggregate row", got["net"])
	}
	agg, ok := net[0].(map[string]any)
	if !ok {
		t.Fatalf("aggregate net row = %#v", net[0])
	}
	if agg["iface"] != "aggregate" || agg["rx_bps"] != wantRXBps || agg["tx_bps"] != wantTXBps {
		t.Fatalf("aggregate net row = %#v", agg)
	}
}

func assertJSONLineHasOnlyNetIfaces(t *testing.T, line string, want []string) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("invalid JSON line %q: %v", line, err)
	}
	net, ok := got["net"].([]any)
	if !ok || len(net) != len(want) {
		t.Fatalf("net = %#v, want interfaces %#v", got["net"], want)
	}
	for i, wantIface := range want {
		row, ok := net[i].(map[string]any)
		if !ok {
			t.Fatalf("net row %d = %#v", i, net[i])
		}
		if row["iface"] != wantIface {
			t.Fatalf("net row %d iface = %#v, want %q", i, row["iface"], wantIface)
		}
	}
}
