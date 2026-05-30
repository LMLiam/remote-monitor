package monitor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmliam/remote-monitor/internal/config"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/metrics"
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/transport"
	"github.com/lmliam/remote-monitor/internal/version"
)

const streamChannelBuffer = 32

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

	return run(ctx, cfg)
}

func run(ctx context.Context, cfg core.Config) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	state := initialAppState(cfg)

	sampleCh := make(chan core.Sample, streamChannelBuffer)
	eventCh := make(chan core.StreamEvent, streamChannelBuffer)
	go transport.RunStream(ctx, cfg, sampleCh, eventCh)

	isTTY := render.StdoutIsTTY()
	if isTTY {
		return runTUI(ctx, state, sampleCh, eventCh)
	}

	renderTicker := time.NewTicker(RenderInterval(cfg, isTTY))
	defer renderTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-eventCh:
			ApplyEvent(&state, ev)
			appliedSample := DrainPending(&state, sampleCh, eventCh)
			if !isTTY && appliedSample {
				fmt.Println(render.NonInteractive(state))
			}
		case smp := <-sampleCh:
			ApplySample(&state, smp)
			_ = DrainPending(&state, sampleCh, eventCh)
			if !isTTY {
				fmt.Println(render.NonInteractive(state))
			}
		case <-renderTicker.C:
			_ = DrainPending(&state, sampleCh, eventCh)
			fmt.Println(render.NonInteractive(state))
		}
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
	for {
		select {
		case smp := <-sampleCh:
			ApplySample(state, smp)
			appliedSample = true

			continue
		default:
		}

		select {
		case ev := <-eventCh:
			ApplyEvent(state, ev)

			continue
		default:
		}

		return appliedSample
	}
}
