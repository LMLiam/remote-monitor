package transport

import (
	"bufio"
	"bytes"
	"context"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/parser"
	"io"
	"os/exec"
	"strings"
	"time"
)

const (
	scannerInitialBufferKiB = 64
	scannerMaxBufferKiB     = 1024
	scannerBytesPerKiB      = 1024
	defaultRetryDelay       = 30 * time.Second
)

type activeStream struct {
	cmd    *exec.Cmd
	stdout io.Reader
	stderr *bytes.Buffer
}

// RunStream runs the SSH sampler loop until the context is cancelled.
func RunStream(ctx context.Context, cfg core.Config, sampleCh chan<- core.Sample, eventCh chan<- core.StreamEvent) {
	reconnectCount := 0
	attempts := 0
	delay := cfg.ReconnectBaseDelay
	intervalSeconds := int(cfg.Interval / time.Second)

	for {
		if ctx.Err() != nil {
			return
		}

		attempts++
		sendEvent(ctx, eventCh, newStreamEvent(core.StatusConnecting, core.DetailOpeningSSHSession, 0, attempts, false, time.Time{}))

		stream, err := openActiveStream(ctx, cfg, intervalSeconds)
		if err != nil {
			var keepRunning bool
			reconnectCount, delay, keepRunning = failStreamAttempt(ctx, eventCh, err.Error(), reconnectCount, attempts, delay)
			if !keepRunning {
				return
			}

			continue
		}

		sendEvent(ctx, eventCh, newStreamEvent(core.StatusConnecting, "waiting for first Sample", 0, attempts, true, time.Time{}))

		var prs parser.Parser
		scanner := bufio.NewScanner(stream.stdout)
		scanner.Buffer(make([]byte, 0, scannerInitialBufferKiB*scannerBytesPerKiB), scannerMaxBufferKiB*scannerBytesPerKiB)
		streamHadSample := false

		for scanner.Scan() {
			smp, ok := prs.HandleLine(scanner.Text())
			if !ok {
				continue
			}
			streamHadSample = true
			attempts = 0
			delay = cfg.ReconnectBaseDelay
			smp.ReceivedAt = time.Now()
			sendSample(ctx, sampleCh, *smp)
			sendEvent(ctx, eventCh, newStreamEvent(core.StatusLive, core.DetailStreamHealthy, reconnectCount, 0, true, time.Time{}))
		}

		scanErr := scanner.Err()
		waitErr := stream.cmd.Wait()
		if ctx.Err() != nil {
			return
		}

		reconnectCount++
		nextRetry := time.Now().Add(delay)
		detail := streamDisconnectDetail(scanErr, waitErr, stream.stderr.String(), streamHadSample)
		sendEvent(ctx, eventCh, newStreamEvent(core.StatusDisconnected, detail, reconnectCount, attempts+1, false, nextRetry))
		if !sleepContext(ctx, delay) {
			return
		}
		delay = backoff(delay)
	}
}

func openActiveStream(ctx context.Context, cfg core.Config, intervalSeconds int) (activeStream, error) {
	args := SSHArgs(cfg, intervalSeconds)
	// #nosec G204 -- the executable is fixed; SSHArgs builds the argument list from validated Config.
	cmd := exec.CommandContext(ctx, "ssh", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return activeStream{cmd: nil, stdout: nil, stderr: nil}, err
	}

	var stderr bytes.Buffer
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return activeStream{cmd: nil, stdout: nil, stderr: nil}, err
	}
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return activeStream{cmd: nil, stdout: nil, stderr: nil}, err
	}
	if _, err := ioWriteString(stdin, remoteSampler); err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		return activeStream{cmd: nil, stdout: nil, stderr: nil}, err
	}
	_ = stdin.Close()

	return activeStream{
		cmd:    cmd,
		stdout: stdout,
		stderr: &stderr,
	}, nil
}

func streamDisconnectDetail(scanErr, waitErr error, stderrText string, streamHadSample bool) string {
	detail := strings.TrimSpace(stderrText)
	switch {
	case scanErr != nil:
		return scanErr.Error()
	case detail != "":
		return detail
	case waitErr != nil:
		return waitErr.Error()
	case streamHadSample:
		return core.DetailSSHStreamEnded
	default:
		return "no Sample received"
	}
}

func newStreamEvent(state, detail string, reconnectCount, attempts int, streamAlive bool, nextRetry time.Time) core.StreamEvent {
	return core.StreamEvent{
		State:          state,
		Detail:         detail,
		ReconnectCount: reconnectCount,
		Attempts:       attempts,
		StreamAlive:    streamAlive,
		NextRetry:      nextRetry,
		At:             time.Time{},
	}
}

func failStreamAttempt(
	ctx context.Context,
	eventCh chan<- core.StreamEvent,
	detail string,
	reconnectCount int,
	attempts int,
	delay time.Duration,
) (int, time.Duration, bool) {
	nextRetry := time.Now().Add(delay)
	reconnectCount++
	sendEvent(ctx, eventCh, newStreamEvent(core.StatusDisconnected, detail, reconnectCount, attempts, false, nextRetry))
	if !sleepContext(ctx, delay) {
		return reconnectCount, delay, false
	}

	return reconnectCount, backoff(delay), true
}

func sendSample(ctx context.Context, ch chan<- core.Sample, smp core.Sample) {
	select {
	case <-ctx.Done():
	case ch <- smp:
	}
}

func sendEvent(ctx context.Context, ch chan<- core.StreamEvent, ev core.StreamEvent) {
	if ev.At.IsZero() {
		ev.At = time.Now()
	}
	select {
	case <-ctx.Done():
	case ch <- ev:
	}
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func backoff(d time.Duration) time.Duration {
	if d >= 30*time.Second {
		return defaultRetryDelay
	}
	d *= 2
	if d > 30*time.Second {
		return defaultRetryDelay
	}

	return d
}

func ioWriteString(w interface {
	Write(p []byte) (n int, err error)
}, s string) (int, error) {
	return w.Write([]byte(s))
}
