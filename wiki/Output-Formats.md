# Output Formats

Remote Monitor supports interactive and script-friendly output. Use `-output` to
choose a mode explicitly, or leave it at `auto` and let the monitor pick the
best format for stdout.

## Output Modes

| Mode | Behavior |
| --- | --- |
| `auto` | TUI when stdout is a TTY, text when stdout is not a TTY. |
| `tui` | Interactive Bubble Tea dashboard. |
| `text` | Non-interactive text summary for terminals, pipes, and one-shot use. |
| `jsonl` | One normalized JSON object per parsed sample. |

`--once` forces text output in `auto` mode, even when stdout is a TTY. Explicit
`--once -output text` and `--once -output jsonl` are supported.
`--once -output tui` is not supported because one-shot mode exits after the
first valid sample.

## JSONL Export

Use `-output jsonl` to write one machine-readable JSON object for each parsed
sample. Without `-out`, JSONL is written to stdout and stdout contains only JSON
objects, one per line. With `-out samples.jsonl`, the file is created or
truncated before the SSH stream starts, and JSONL is written to that file while
stdout stays empty.

```sh
remote-monitor gpu-box -output jsonl
remote-monitor gpu-box -output jsonl -out samples.jsonl
```

JSONL exports use the normalized local schema
`remote-monitor.normalized_sample.v1`, derived from `internal/core.Sample` after
sampler parsing rather than the raw remote sampler JSON. Fields are snake_case
and include host, CPU, memory, pressure, swap, disk, TCP, filesystem, network,
process, power, GPU, and local `received_at` values.

Repeated values such as `net`, `filesystems`, `disks`, `cpu_core_usage`,
`top_processes`, `gpu_processes`, `gpus`, and `power_supplies` are JSON arrays.
Lifecycle and reconnect state are not included in the JSONL stream.

Example object shape:

```json
{
  "schema": "remote-monitor.normalized_sample.v1",
  "received_at": "2026-05-28T19:35:46.000000123Z",
  "remote_name": "gpu-box",
  "cpu_percent": 42,
  "ram_used_mib": 2455,
  "net": [],
  "filesystems": [],
  "disks": [],
  "cpu_core_usage": [],
  "top_processes": [],
  "gpu_processes": [],
  "gpus": [],
  "power_supplies": []
}
```

## Snapshot Mode

Use `--once` when scripts, CI jobs, cron jobs, or incident notes need exactly
one remote sample. Snapshot mode starts the normal SSH sampler, waits for the
first valid parsed sample, writes that sample, and exits.

```sh
remote-monitor gpu-box --once
remote-monitor gpu-box --once -output text
remote-monitor gpu-box --once -output jsonl
remote-monitor gpu-box --once -output jsonl -out snapshot.jsonl
```

`--once -output jsonl` writes exactly one JSON object followed by a newline. With
`-out snapshot.jsonl`, the file is created or truncated before the SSH stream
starts and stdout remains empty.
