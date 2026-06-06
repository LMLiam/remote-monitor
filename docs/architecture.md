# Architecture

## Overview

Remote Monitor collects system metrics from remote Linux hosts over SSH, renders
those metrics in a Bubble Tea terminal UI, and can also stream normalized JSONL
for scripts and other tools. The remote host does not run a daemon: the local
process sends a Bash sampler over SSH, reads one JSON object per interval, parses
that object into the shared core model, and then renders or encodes the result
locally.

The codebase follows a layered architecture. Each layer owns one kind of
translation: transport owns SSH process execution, the sampler owns remote data
collection, the parser owns the sampler wire format, the core package owns the
local sample model, monitor owns application wiring and output selection, output
owns machine-readable encoding, and render owns terminal presentation. For the
package layout list, see [AGENTS.md](../AGENTS.md#layout).

## Data Flow

A live sample moves through the system in this order:

1. The transport layer launches an SSH subprocess.
2. The embedded Bash sampler is piped to the SSH process on stdin.
3. The remote `bash -s` process emits one JSON line per sampling interval.
4. The parser reads each line and converts valid JSON into `core.Sample`.
5. The monitor layer applies samples and stream events to `core.AppState`.
6. The selected output loop renders text/TUI output or writes JSONL.

```text
SSH → sampler.sh → stdout → parser → Sample → AppState
                                             ├→ render → output (TUI/text)
                                             └→ output (JSONL)
```

This flow is deliberately one-way. Remote collection code should not know about
terminal layout, render code should not know how SSH is launched, and parser code
should translate the wire shape without inventing presentation behavior.

## Transport and SSH Execution

`internal/transport/` owns the long-running SSH sampler stream. Its public entry
point is `RunStream()`, which receives a `core.Config` and two channels:
`sampleCh` for parsed `core.Sample` values and `eventCh` for connection lifecycle
updates.

`RunStream()` opens each SSH attempt through `openActiveStream()`. That helper
builds arguments with `SSHArgs()` and launches the fixed `ssh` executable with
`exec.CommandContext`, so cancellation of the parent context tears down the
subprocess. The SSH argument list enables connection reuse with
`ControlMaster=auto`, `ControlPersist`, and a `ControlPath` from
`ResolveSSHControlPath()`. If the user does not configure a control path, the
transport derives a per-process socket path under `/tmp` from the target host.

The sampler body is embedded in `internal/transport/sampler.go` with
`//go:embed sampler.sh`. After the SSH process starts, the transport writes that
embedded script to stdin and closes the pipe. The remote command is `bash -s`
with sampler arguments after `--`, including the interval, process sort, process
filter, and process count.

The stream loop scans remote stdout line by line. Each valid sample is timestamped
with local `ReceivedAt`, sent on `sampleCh`, and paired with a live lifecycle
event on `eventCh`. Parse failures are remembered by the parser but skipped so a
single malformed line does not stop a healthy stream. When the SSH process exits,
the stream sends a disconnected event with the best available detail from scanner
errors, SSH stderr, parse errors, or process exit state.

Reconnect behavior uses exponential backoff based on `Config.ReconnectBaseDelay`
and caps at 30 seconds. One-shot mode (`--once`) returns after the first sample
or after a failed attempt that never produced a sample; continuous mode sleeps
for the current delay and retries until the context is canceled.

## Remote Sampler Responsibilities

`internal/transport/sampler/` contains the source modules for the remote Bash
sampler. Remote Monitor still sends a single script to the host, but that script
is assembled from smaller modules so each collector can be reviewed and tested
independently.

`sampler/assemble.sh` reads `sampler/manifest.txt` and rewrites
`internal/transport/sampler.sh`, which is the file embedded by the Go transport.
Key modules include:

- `config.sh` for defaults and interval helpers.
- `json.sh` for JSON escaping and numeric normalization.
- `cpu.sh`, `memory.sh`, `network.sh`, `disk.sh`, `filesystems.sh`, and
  `pressure.sh` for core host metrics.
- `processes.sh` for top process rows.
- `gpu_common.sh`, `gpu_nvidia.sh`, `gpu_intel.sh`, and `gpu_amd.sh` for GPU
  device and GPU process metrics.
- `power.sh` for Linux power-supply and UPS metrics.
- `wsl.sh` for optional WSL host-side metrics.
- `main.sh` for initialization, the timing loop, and final JSON assembly.

The sampler emits one JSON object per line with `version: 1`. Numeric fields that
cannot be read should use the shared `normalize_int` or `normalize_float`
helpers. Those helpers return `-1` for empty, `N/A`, malformed, or unavailable
values. String fields use an empty string when unavailable. This sentinel
contract lets renderers distinguish "unknown" from real zero values such as 0%
utilization or 0 bytes per second.

For assembly details, module boundaries, and sampler-specific checks, see
[internal/transport/sampler/README.md](../internal/transport/sampler/README.md).

## Parser Responsibilities

`internal/parser/` owns the sampler JSON wire format. `Parser.HandleLine()`
accepts one sampler output line and returns `(*core.Sample, true)` only when the
line is non-empty, valid JSON, and uses the supported wire protocol version.
Empty lines are ignored without changing parser error state.

The wire format intentionally differs from `core.Sample` in a few places.
`wireSample` mirrors the remote JSON object, while nested objects such as
`wireSwap`, `wireDisk`, and `wirePower` preserve sampler grouping. The parser
flattens those nested values into `core.Sample` fields such as `SwapFreeKiB`,
`DiskReadBps`, `ExternalPowerOnline`, and `PowerSupplies`.

The parser validates `version` against the current wire protocol version, which
is `1`. Any other version is rejected and recorded as the latest parse failure.
`Parser.LastError()` returns the most recent rejection error from a non-empty
line so the transport can report useful disconnect detail if a stream ends
before producing a valid sample.

Parser code should stay focused on decoding and flattening. It should not apply
network filters, compute histories, or choose presentation defaults; those belong
to monitor, metrics, and render.

## Core Sample Model

`internal/core/types.go` contains the shared model used across the local process.
`Sample` represents one complete normalized snapshot from the remote host. It has
roughly 60 scalar and slice fields for CPU, memory, pressure, swap, disk,
network, processes, GPU, power, remote timestamps, and local receipt time.

Repeated or nested metric families use dedicated stat types:

- `NetStat` for per-interface network counters.
- `CPUCore` for per-core CPU utilization.
- `ProcessStat` and `GPUProcessStat` for host and GPU process rows.
- `FilesystemStat` and `DiskStat` for filesystem and block-device views.
- `GPUStat` for device-level GPU metrics.
- `PowerSupplyStat` for Linux power-supply devices.

Unavailable metrics use sentinel values rather than zero. Numeric fields use
`-1` when a command, file, permission, or device capability is unavailable.
String fields use `""` for missing text. This is especially important for
optional sections such as GPU, power, CPU temperature, network speed, and disk
latency because zero can be a valid measurement.

`EmptySample()` constructs the initial sample with important optional power
fields set to `-1`. The parser also initializes optional power fields to `-1`
when the sampler omits the `power` object entirely.

Other supporting core types include:

- `Config`, which carries CLI, SSH, sampling, history, rendering, network
  filtering, output, and theme settings.
- `AppState`, which combines the current sample, runtime connection state,
  reconnect metadata, scroll state, network ceilings, and rolling history.
- `StreamEvent`, which carries transport lifecycle state such as connecting,
  live, disconnected, reconnect attempts, and next retry time.

## Output Modes

Output mode constants live in `internal/core/types.go`:

- `OutputModeAuto` (`""`)
- `OutputModeTUI` (`"tui"`)
- `OutputModeText` (`"text"`)
- `OutputModeJSONL` (`"jsonl"`)

`internal/monitor/app.go` resolves and runs the selected mode. Explicit
`-output` values win. In auto mode, `--once` forces text output, an interactive
TTY uses the TUI, and non-TTY stdout uses text output. `--once -output tui` is
rejected because a one-shot command cannot run the interactive Bubble Tea loop.
`-out` is valid only with JSONL output.

The main run loops are:

- `runTUI()` for the interactive Bubble Tea program.
- `runText()` for repeated non-interactive text snapshots.
- `runJSONL()` for streaming normalized JSON objects.
- `runOnce()` for one-shot text or JSONL output after the first sample.

Text and TUI modes eventually use `internal/render/`. JSONL mode uses
`internal/output/`. The JSONL writer wraps `core.Sample` with a `schema` field
using `remote-monitor.normalized_sample.v1` and includes `received_at` when the
sample has a local receipt timestamp. It also normalizes nil repeated fields to
empty JSON arrays.

## TUI Renderer Responsibilities

`internal/render/` contains stateless rendering functions. It should not perform
I/O, own timers, read SSH streams, mutate monitor state, or parse sampler JSON.
Given a `core.AppState` plus dimensions, it returns strings.

Key rendering entry points are:

- `FullFrame()` for the complete dashboard frame.
- `ViewportFrame()` for a clipped view of a full frame with scroll metadata.
- `NonInteractive()` for compact line-oriented output.

The renderer builds a responsive multi-column layout with `BuildLayoutSections`
and `OptimizeSectionAssignments`. Section builders return table strings for CPU,
GPU, system, memory, storage, network, power, and process panels, and the layout
optimizer balances their rendered heights across feasible columns.

`internal/monitor/tui.go` owns the interactive Bubble Tea integration. `TUIModel`
implements Bubble Tea's `tea.Model` interface with `Init`, `Update`, and `View`,
holds the viewport, forwards sampler and stream events into monitor state, and
refreshes rendered content through `render.FullFrame()`.

Rolling history visualization lives in `internal/render/render_history.go` and
the sparkline helpers in `sparkline.go`. The monitor layer updates history
buffers when samples are applied; render only chooses labels, colors, scaling,
and layout for displaying those histories.

## Test Strategy

Use the Makefile targets for contributor workflows:

- `make test` runs all Go tests with the `integration` build tag.
- `make check` runs the full local quality gate, including Go formatting, shell
  formatting, ShellCheck, workflow helper script tests, `go vet`, integration
  tests, `golangci-lint`, and build.

Most layers have package-level unit tests under `internal/*/`:

- `internal/transport/*_test.go` covers SSH argument construction, stream
  behavior, sampler assembly verification, embedded script syntax, and sampler
  module helpers.
- `internal/parser/parser_test.go` covers wire JSON parsing, version rejection,
  degraded fixtures, GPU and power payloads, escaped control characters, and
  `LastError()`.
- `internal/core/theme_test.go` covers theme constants and validation; most
  core sample contracts are exercised through parser, monitor, output, and
  render tests that consume `core.Sample` and `core.AppState`.
- `internal/output/jsonl_test.go` covers schema, `received_at`, JSON escaping,
  and repeated-field encoding.
- `internal/monitor/*_test.go` covers output mode resolution, one-shot behavior,
  JSONL/text loops, app state application, render timing, viewport behavior, and
  history buffers. Monitor render fixture tests also cover frame, viewport,
  layout, title, row, and history behavior at the application boundary.
- `internal/render/*_test.go` covers formatting, severity, table widths,
  sparklines, and other stateless presentation helpers that live directly in the
  render package.

The integration E2E test in `tests/e2e/ssh_e2e_test.go` builds and runs a Docker
container with an SSH server, starts the local binary against that server, and
verifies the SSH-to-sampler-to-output path. It self-skips when Docker is not
available.

For focused changes, run the nearest package tests first, then `make test` or
`make check` before opening a PR. Sampler changes should also run
`go generate ./internal/transport` or `bash internal/transport/sampler/assemble.sh`
and include the regenerated `internal/transport/sampler.sh`.

## Common Change Examples

**Adding a new metric field**

Start where the data exists. Add collection logic to the relevant sampler module
or create a focused new module if the metric has a separate source. Emit the
field in `main.sh` JSON, using `normalize_int`, `normalize_float`, or
`json_escape` as appropriate. Add the field to the parser wire struct, flatten it
into `core.Sample`, and add parser tests for available and unavailable values.
Then add metrics/rendering logic where the value should appear and cover that
presentation with package tests.

**Adding a new output mode**

Add the mode constant in `internal/core/types.go`, teach config parsing to accept
the value if needed, add a resolution or dispatch case in `resolveOutputMode` and
`run()` in `internal/monitor/app.go`, and implement a focused run loop or encoder
in the appropriate package. Add tests beside the changed monitor/output code for
TTY behavior, `--once` behavior, `-out` compatibility, and cancellation.

**Renderer-only presentation changes**

Keep presentation-only work inside `internal/render/` when it does not need new
state, new fields, or new I/O. Prefer changing row builders, formatters, layout
helpers, severity thresholds, or history rendering there. Add or update render
tests that compare visible text, layout dimensions, or ANSI-stripped output as
appropriate.

**Adding a new sampler module**

Create the module under `internal/transport/sampler/`, add it to
`internal/transport/sampler/manifest.txt` so `sampler/assemble.sh` includes it,
and wire its initialization or collection call from `main.sh`. Regenerate the
embedded sampler, commit both the module and `internal/transport/sampler.sh`, and
run the sampler-focused transport tests plus the shell checks from the sampler
README.
