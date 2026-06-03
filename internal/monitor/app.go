package monitor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmliam/remote-monitor/internal/config"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	jsonl "github.com/lmliam/remote-monitor/internal/output"
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/transport"
	"github.com/lmliam/remote-monitor/internal/version"
)

const streamChannelBuffer = 32

var (
	errOutRequiresJSONL    = errors.New("-out requires -output jsonl")
	errOnceTUIUnsupported  = errors.New("--once does not support -output tui")
	errOnceNoSample        = errors.New("stream ended before first sample")
	errOnceStreamFailed    = errors.New("one-shot sample failed before first sample")
	errOnceContextCanceled = errors.New("one-shot sample canceled before first sample")
)

type streamRunner func(context.Context, core.Config, chan<- core.Sample, chan<- core.StreamEvent)

type runDependencies struct {
	stdout      io.Writer
	stdoutIsTTY func() bool
	runStream   streamRunner
}

// RunCLI parses process configuration and starts the monitor application.
func RunCLI() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return Run(ctx, os.Args[1:], os.Stdout)
}

// Run parses args and starts the monitor, or prints version metadata and exits.
func Run(ctx context.Context, args []string, stdout io.Writer) error {
	cfg, err := config.ParseConfig(args)
	if err != nil {
		return err
	}
	if cfg.ShowVersion {
		_, err = fmt.Fprintln(stdout, version.Current().String())

		return err
	}

	return run(ctx, cfg, defaultRunDependencies(stdout))
}

func defaultRunDependencies(stdout io.Writer) runDependencies {
	return runDependencies{
		stdout:      stdout,
		stdoutIsTTY: render.StdoutIsTTY,
		runStream:   transport.RunStream,
	}
}

func run(ctx context.Context, cfg core.Config, deps runDependencies) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	deps = normalizeRunDependencies(deps)
	outputMode := resolveOutputMode(cfg, deps.stdoutIsTTY())
	if cfg.Once && outputMode == core.OutputModeTUI {
		return errOnceTUIUnsupported
	}
	outputWriter, closeOutput, err := openOutputWriter(cfg, outputMode, deps.stdout)
	if err != nil {
		return err
	}

	state := initialAppState(cfg)

	sampleCh := make(chan core.Sample, streamChannelBuffer)
	eventCh := make(chan core.StreamEvent, streamChannelBuffer)
	go deps.runStream(ctx, cfg, sampleCh, eventCh)

	var runErr error
	if cfg.Once {
		runErr = runOnce(ctx, state, sampleCh, eventCh, outputMode, outputWriter)
	} else {
		switch outputMode {
		case core.OutputModeTUI:
			runErr = runTUI(ctx, state, sampleCh, eventCh)
		case core.OutputModeJSONL:
			runErr = runJSONL(ctx, state, sampleCh, eventCh, outputWriter)
		default:
			runErr = runText(ctx, cfg, state, sampleCh, eventCh, outputWriter)
		}
	}

	if closeErr := closeOutput(); runErr == nil && closeErr != nil {
		return closeErr
	}

	return runErr
}

func normalizeRunDependencies(deps runDependencies) runDependencies {
	if deps.stdout == nil {
		deps.stdout = io.Discard
	}
	if deps.stdoutIsTTY == nil {
		deps.stdoutIsTTY = render.StdoutIsTTY
	}
	if deps.runStream == nil {
		deps.runStream = transport.RunStream
	}

	return deps
}

func resolveOutputMode(cfg core.Config, stdoutIsTTY bool) string {
	if cfg.OutputMode != core.OutputModeAuto {
		return cfg.OutputMode
	}
	if cfg.Once {
		return core.OutputModeText
	}
	if stdoutIsTTY {
		return core.OutputModeTUI
	}

	return core.OutputModeText
}

func openOutputWriter(cfg core.Config, outputMode string, stdout io.Writer) (io.Writer, func() error, error) {
	if cfg.OutputPath != "" && outputMode != core.OutputModeJSONL {
		return nil, nil, errOutRequiresJSONL
	}
	if cfg.OutputPath == "" {
		return stdout, func() error { return nil }, nil
	}

	file, err := os.Create(cfg.OutputPath)
	if err != nil {
		return nil, nil, err
	}

	return file, file.Close, nil
}

func runOnce(
	ctx context.Context,
	state core.AppState,
	sampleCh <-chan core.Sample,
	eventCh <-chan core.StreamEvent,
	outputMode string,
	stdout io.Writer,
) error {
	writer := jsonl.NewWriter(stdout)
	for sampleCh != nil || eventCh != nil {
		select {
		case smp, ok := <-sampleCh:
			if !ok {
				sampleCh = nil

				continue
			}

			return writeOnceSample(&state, smp, outputMode, stdout, writer)
		default:
		}

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return errOnceContextCanceled
			}

			return ctx.Err()
		case smp, ok := <-sampleCh:
			if !ok {
				sampleCh = nil

				continue
			}

			return writeOnceSample(&state, smp, outputMode, stdout, writer)
		case ev, ok := <-eventCh:
			if !ok {
				eventCh = nil

				continue
			}
			ApplyEvent(&state, ev)
			if ev.State == core.StatusDisconnected && !ev.StreamAlive {
				return fmt.Errorf("%w: %s", errOnceStreamFailed, onceFailureDetail(ev))
			}
		}
	}

	return errOnceNoSample
}

func writeOnceSample(
	state *core.AppState,
	smp core.Sample,
	outputMode string,
	stdout io.Writer,
	writer *jsonl.Writer,
) error {
	ApplySample(state, smp)
	if outputMode == core.OutputModeJSONL {
		return writer.WriteSample(smp)
	}

	_, err := fmt.Fprintln(stdout, render.NonInteractive(*state))

	return err
}

func onceFailureDetail(ev core.StreamEvent) string {
	if ev.Detail != "" {
		return ev.Detail
	}

	return errOnceNoSample.Error()
}

func runText(
	ctx context.Context,
	cfg core.Config,
	state core.AppState,
	sampleCh <-chan core.Sample,
	eventCh <-chan core.StreamEvent,
	stdout io.Writer,
) error {
	renderTicker := time.NewTicker(RenderInterval(cfg, false))
	defer renderTicker.Stop()

	for sampleCh != nil || eventCh != nil {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-eventCh:
			if !ok {
				eventCh = nil

				continue
			}
			ApplyEvent(&state, ev)
			appliedSample := DrainPending(&state, sampleCh, eventCh)
			if appliedSample {
				if _, err := fmt.Fprintln(stdout, render.NonInteractive(state)); err != nil {
					return err
				}
			}
		case smp, ok := <-sampleCh:
			if !ok {
				sampleCh = nil

				continue
			}
			ApplySample(&state, smp)
			_ = DrainPending(&state, sampleCh, eventCh)
			if _, err := fmt.Fprintln(stdout, render.NonInteractive(state)); err != nil {
				return err
			}
		case <-renderTicker.C:
			_ = DrainPending(&state, sampleCh, eventCh)
			if _, err := fmt.Fprintln(stdout, render.NonInteractive(state)); err != nil {
				return err
			}
		}
	}

	return nil
}

func runJSONL(
	ctx context.Context,
	state core.AppState,
	sampleCh <-chan core.Sample,
	eventCh <-chan core.StreamEvent,
	stdout io.Writer,
) error {
	writer := jsonl.NewWriter(stdout)
	for sampleCh != nil || eventCh != nil {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-eventCh:
			if !ok {
				eventCh = nil

				continue
			}
			ApplyEvent(&state, ev)
			if err := drainJSONLPending(&state, sampleCh, eventCh, writer); err != nil {
				return err
			}
		case smp, ok := <-sampleCh:
			if !ok {
				sampleCh = nil

				continue
			}
			ApplySample(&state, smp)
			if err := writer.WriteSample(smp); err != nil {
				return err
			}
		}
	}

	return nil
}

func drainJSONLPending(
	state *core.AppState,
	sampleCh <-chan core.Sample,
	eventCh <-chan core.StreamEvent,
	writer *jsonl.Writer,
) error {
	for {
		select {
		case smp, ok := <-sampleCh:
			if !ok {
				sampleCh = nil

				continue
			}
			ApplySample(state, smp)
			if err := writer.WriteSample(smp); err != nil {
				return err
			}

			continue
		default:
		}

		select {
		case ev, ok := <-eventCh:
			if !ok {
				eventCh = nil

				continue
			}
			ApplyEvent(state, ev)

			continue
		default:
		}

		return nil
	}
}

func initialAppState(cfg core.Config) core.AppState {
	return core.AppState{
		Cfg:                cfg,
		RuntimeState:       "starting",
		RuntimeDetail:      "initializing",
		LastTransport:      "",
		SampleCount:        0,
		ReconnectCount:     0,
		ReconnectAttempts:  0,
		NextRetry:          time.Time{},
		LastRx:             time.Time{},
		StreamAlive:        false,
		Current:            core.EmptySample(),
		HasSample:          false,
		ScrollOffset:       0,
		ScrollMax:          0,
		NetCeilings:        map[string]int64{},
		CPUHistory:         nil,
		CPUFreqHistory:     nil,
		CPUTempHistory:     nil,
		RAMHistory:         nil,
		RAMAvailHistory:    nil,
		DiskHistory:        nil,
		DiskLatencyHistory: nil,
		GPUHistory:         nil,
		VRAMHistory:        nil,
		TempHistory:        nil,
		PowerHistory:       nil,
		NetRXHistory:       nil,
		NetTXHistory:       nil,
		NetIssueHistory:    nil,
	}
}

// RenderInterval returns the redraw interval for TTY and non-interactive output.
func RenderInterval(cfg core.Config, isTTY bool) time.Duration {
	if isTTY {
		fps := cfg.RenderFPS
		if fps < 1 {
			fps = 12
		}

		return time.Second / time.Duration(fps)
	}

	return time.Second
}

// ApplyEvent merges a stream lifecycle event into application state.
func ApplyEvent(state *core.AppState, ev core.StreamEvent) {
	if ev.At.IsZero() {
		ev.At = time.Now()
	}
	if state.HasSample && ev.At.Before(state.LastRx) {
		return
	}
	state.RuntimeState = ev.State
	state.RuntimeDetail = ev.Detail
	state.StreamAlive = ev.StreamAlive
	state.ReconnectCount = ev.ReconnectCount
	state.ReconnectAttempts = ev.Attempts
	state.NextRetry = ev.NextRetry
	if ev.Detail != "" {
		state.LastTransport = ev.Detail
	}
}

// ApplySample merges a new sample and updates all rolling history series.
func ApplySample(state *core.AppState, smp core.Sample) {
	if smp.ReceivedAt.IsZero() {
		smp.ReceivedAt = time.Now()
	}
	if state.HasSample && smp.ReceivedAt.Before(state.LastRx) {
		return
	}
	state.Current = smp
	state.HasSample = true
	state.SampleCount++
	state.LastRx = smp.ReceivedAt
	state.RuntimeState = core.StatusLive
	state.RuntimeDetail = core.DetailStreamHealthy
	state.StreamAlive = true
	state.ReconnectAttempts = 0
	metrics.UpdateNetCeilings(state, smp)

	appendHistory(&state.CPUHistory, smp.CPUPercent, state.Cfg.HistoryLimit)
	appendHistory(&state.CPUFreqHistory, metrics.ClockPercent(smp.CPUFreqMHz, smp.CPUMaxFreqMHz), state.Cfg.HistoryLimit)
	appendHistory(&state.CPUTempHistory, smp.CPUTempC, state.Cfg.HistoryLimit)
	appendHistory(&state.RAMHistory, metrics.PercentOf(smp.RAMUsedMiB, smp.RAMTotalMiB), state.Cfg.HistoryLimit)
	appendHistory(&state.RAMAvailHistory, metrics.RAMAvailablePercent(smp), state.Cfg.HistoryLimit)
	appendHistory(&state.DiskHistory, smp.DiskUtil, state.Cfg.HistoryLimit)
	appendHistory(&state.DiskLatencyHistory, metrics.DiskLatencyHistoryPercent(smp), state.Cfg.HistoryLimit)
	appendHistory(&state.GPUHistory, metrics.OverallGPUUtil(smp), state.Cfg.HistoryLimit)
	appendHistory(&state.VRAMHistory, metrics.OverallVRAMPct(smp), state.Cfg.HistoryLimit)
	appendHistory(&state.TempHistory, metrics.OverallTempPct(smp), state.Cfg.HistoryLimit)
	appendHistory(&state.PowerHistory, metrics.OverallPowerPct(smp), state.Cfg.HistoryLimit)
	appendHistory64(&state.NetRXHistory, metrics.TotalNetRXBps(smp), state.Cfg.HistoryLimit)
	appendHistory64(&state.NetTXHistory, metrics.TotalNetTXBps(smp), state.Cfg.HistoryLimit)
	appendHistory(&state.NetIssueHistory, metrics.NetIssueHistoryPercent(smp), state.Cfg.HistoryLimit)
}

// DrainPending applies all buffered samples and events without blocking.
func DrainPending(state *core.AppState, sampleCh <-chan core.Sample, eventCh <-chan core.StreamEvent) bool {
	appliedSample := false
	for sampleCh != nil || eventCh != nil {
		select {
		case smp, ok := <-sampleCh:
			if !ok {
				sampleCh = nil

				continue
			}
			ApplySample(state, smp)
			appliedSample = true

			continue
		default:
		}

		select {
		case ev, ok := <-eventCh:
			if !ok {
				eventCh = nil

				continue
			}
			ApplyEvent(state, ev)

			continue
		default:
		}

		return appliedSample
	}

	return appliedSample
}
