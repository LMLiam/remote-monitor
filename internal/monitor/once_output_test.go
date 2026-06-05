//nolint:testpackage // These tests exercise unexported run-loop dependencies to avoid opening SSH.
package monitor

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
)

func TestResolveOutputModeUsesTextForOnceAuto(t *testing.T) {
	t.Parallel()

	if got := resolveOutputMode(outputTestConfig(func(cfg *core.Config) {
		cfg.Once = true
		cfg.OutputMode = core.OutputModeAuto
	}), true); got != core.OutputModeText {
		t.Fatalf("once auto output mode = %q", got)
	}
}

func TestRunOnceWritesSingleTextSnapshot(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	firstSample := outputTestSample()
	secondSample := outputTestSample()
	secondSample.CPUPercent = 42
	secondSample.ReceivedAt = firstSample.ReceivedAt.Add(time.Second)

	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.Once = true
		cfg.OutputMode = core.OutputModeText
	}), runDependencies{
		stdout:      &out,
		stdoutIsTTY: func() bool { return true },
		runStream: func(_ context.Context, _ core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
			defer close(sampleCh)
			defer close(eventCh)
			sampleCh <- firstSample
			time.Sleep(20 * time.Millisecond)
			sampleCh <- secondSample
		},
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "CPU 99%") {
		t.Fatalf("text snapshot missing first sample CPU value: %q", got)
	}
	if strings.Contains(got, "CPU 42%") {
		t.Fatalf("text snapshot included second sample: %q", got)
	}
	if count := strings.Count(got, "\n"); count != 1 {
		t.Fatalf("text snapshot rendered %d lines in %q", count, got)
	}
}

func TestRunOnceWritesSingleJSONLToStdout(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	firstSample := outputTestSample()
	secondSample := outputTestSample()
	secondSample.CPUPercent = 42
	secondSample.ReceivedAt = firstSample.ReceivedAt.Add(time.Second)

	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.Once = true
		cfg.OutputMode = core.OutputModeJSONL
	}), runDependencies{
		stdout:      &out,
		stdoutIsTTY: func() bool { return false },
		runStream: func(_ context.Context, _ core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
			defer close(sampleCh)
			defer close(eventCh)
			sampleCh <- firstSample
			sampleCh <- secondSample
		},
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	lines := nonEmptyLines(out.String())
	if len(lines) != 1 {
		t.Fatalf("jsonl lines = %#v", lines)
	}
	assertJSONLineHasSample(t, lines[0], firstSample)
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("JSONL stdout does not end with newline: %q", out.String())
	}
}

func TestRunOnceWritesSingleJSONLWithSelectedNetwork(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	firstSample := outputTestSample()
	firstSample.Net = outputSelectionNetStats()

	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.Once = true
		cfg.OutputMode = core.OutputModeJSONL
		cfg.NetIncludePatterns = []string{"wlan*"}
	}), runDependencies{
		stdout:      &out,
		stdoutIsTTY: func() bool { return false },
		runStream: func(_ context.Context, _ core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
			defer close(sampleCh)
			defer close(eventCh)
			sampleCh <- firstSample
		},
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	lines := nonEmptyLines(out.String())
	if len(lines) != 1 {
		t.Fatalf("jsonl lines = %#v", lines)
	}
	assertJSONLineHasOnlyNetIfaces(t, lines[0], []string{outputIfaceWlan0})
}

func TestRunOnceWritesSingleJSONLToFile(t *testing.T) {
	t.Parallel()

	outputPath := filepath.Join(t.TempDir(), "snapshot.jsonl")
	if err := os.WriteFile(outputPath, []byte("stale contents\n"), 0o600); err != nil {
		t.Fatalf("prewrite jsonl output: %v", err)
	}
	var out bytes.Buffer
	firstSample := outputTestSample()
	secondSample := outputTestSample()
	secondSample.CPUPercent = 42
	secondSample.ReceivedAt = firstSample.ReceivedAt.Add(time.Second)

	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.Once = true
		cfg.OutputMode = core.OutputModeJSONL
		cfg.OutputPath = outputPath
	}), runDependencies{
		stdout:      &out,
		stdoutIsTTY: func() bool { return false },
		runStream: func(_ context.Context, _ core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
			defer close(sampleCh)
			defer close(eventCh)
			sampleCh <- firstSample
			sampleCh <- secondSample
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
	assertJSONLineHasSample(t, lines[0], firstSample)
	if strings.Contains(string(content), "stale contents") {
		t.Fatalf("jsonl output file was not truncated: %q", string(content))
	}
}

func TestRunOnceWritesTextWithSelectedNetworkSummary(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	firstSample := outputTestSample()
	firstSample.Net = outputSelectionNetStats()

	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.Once = true
		cfg.OutputMode = core.OutputModeText
		cfg.NetIncludePatterns = []string{outputIfaceEth0, outputIfaceWlan0}
		cfg.NetAggregate = true
	}), runDependencies{
		stdout:      &out,
		stdoutIsTTY: func() bool { return true },
		runStream: func(_ context.Context, _ core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
			defer close(sampleCh)
			defer close(eventCh)
			sampleCh <- firstSample
		},
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "NET RX 400 B/s") || !strings.Contains(got, "TX 50 B/s") {
		t.Fatalf("text snapshot missing selected network summary: %q", got)
	}
	if strings.Contains(got, "1200 B/s") {
		t.Fatalf("text snapshot used unfiltered network total: %q", got)
	}
}

func TestRunOnceReturnsErrorBeforeFirstSample(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.Once = true
		cfg.OutputMode = core.OutputModeText
	}), runDependencies{
		stdout:      &out,
		stdoutIsTTY: func() bool { return false },
		runStream: func(_ context.Context, _ core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
			defer close(sampleCh)
			defer close(eventCh)
			eventCh <- outputTestStreamEvent(func(ev *core.StreamEvent) {
				ev.State = core.StatusDisconnected
				ev.Detail = "ssh exited before sample"
				ev.StreamAlive = false
			})
		},
	})
	if err == nil {
		t.Fatal("expected one-shot stream failure")
	}
	if !strings.Contains(err.Error(), "ssh exited before sample") {
		t.Fatalf("error = %q", err.Error())
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want empty on failure before sample", out.String())
	}
}

func TestRunOnceRejectsTUIBeforeStartingStream(t *testing.T) {
	t.Parallel()

	streamStarted := false
	err := run(context.Background(), outputTestConfig(func(cfg *core.Config) {
		cfg.Once = true
		cfg.OutputMode = core.OutputModeTUI
	}), runDependencies{
		stdout:      &bytes.Buffer{},
		stdoutIsTTY: func() bool { return true },
		runStream: func(context.Context, core.Config, chan<- core.Sample, chan<- core.StreamEvent) {
			streamStarted = true
		},
	})
	if err == nil {
		t.Fatal("expected unsupported one-shot TUI error")
	}
	if !strings.Contains(err.Error(), "--once does not support -output tui") {
		t.Fatalf("error = %q", err.Error())
	}
	if streamStarted {
		t.Fatal("stream started for unsupported one-shot TUI mode")
	}
}
