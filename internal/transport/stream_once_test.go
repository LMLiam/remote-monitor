package transport_test

import (
	"context"
	"strings"
	"testing"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/transport"
)

func TestRunStreamOnceReturnsAfterOpenFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sampleCh := make(chan core.Sample, 1)
	eventCh := make(chan core.StreamEvent, 4)
	done := make(chan struct{})

	go func() {
		defer close(done)
		transport.RunStream(ctx, testConfig(func(cfg *core.Config) {
			cfg.Host = "missing-ssh-host"
			cfg.Once = true
			cfg.Interval = time.Second
			cfg.ReconnectBaseDelay = time.Second
		}), sampleCh, eventCh)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("RunStream did not return after one-shot open failure")
	}

	var events []core.StreamEvent
	for {
		select {
		case ev := <-eventCh:
			events = append(events, ev)
		default:
			if len(events) < 2 {
				t.Fatalf("events = %#v, want connecting and disconnected", events)
			}
			last := events[len(events)-1]
			if last.State != core.StatusDisconnected {
				t.Fatalf("last event = %#v, want disconnected", last)
			}
			if !strings.Contains(last.Detail, "ssh") {
				t.Fatalf("last event detail = %q", last.Detail)
			}
			select {
			case smp := <-sampleCh:
				t.Fatalf("unexpected sample after open failure: %#v", smp)
			default:
			}

			return
		}
	}
}
