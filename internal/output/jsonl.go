// Package output contains machine-readable output encoders.
package output

import (
	"encoding/json"
	"io"
	"time"

	core "github.com/lmliam/remote-monitor/internal/core"
)

// JSONLSchema identifies the first normalized local sample export schema.
const JSONLSchema = "remote-monitor.normalized_sample.v1"

// Writer emits one normalized sample JSON object per line.
type Writer struct {
	encoder *json.Encoder
}

type sample struct {
	core.Sample

	Schema     string `json:"schema"`
	ReceivedAt string `json:"received_at,omitempty"`
}

// NewWriter creates a JSONL writer around the provided destination.
func NewWriter(w io.Writer) *Writer {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	return &Writer{encoder: encoder}
}

// WriteSample writes a single JSON object followed by a newline.
func (w *Writer) WriteSample(smp core.Sample) error {
	return w.encoder.Encode(fromCoreSample(smp))
}

func fromCoreSample(smp core.Sample) sample {
	receivedAt := formatTime(smp.ReceivedAt)
	smp.Net = nonNilSlice(smp.Net)
	smp.Filesystems = nonNilSlice(smp.Filesystems)
	smp.CPUCoresUsage = nonNilSlice(smp.CPUCoresUsage)
	smp.TopProcesses = nonNilSlice(smp.TopProcesses)
	smp.GPUProcesses = nonNilSlice(smp.GPUProcesses)
	smp.GPUs = nonNilSlice(smp.GPUs)
	smp.PowerSupplies = nonNilSlice(smp.PowerSupplies)

	return sample{
		Schema:     JSONLSchema,
		Sample:     smp,
		ReceivedAt: receivedAt,
	}
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339Nano)
}
