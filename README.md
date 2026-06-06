# remote-monitor

[![Build](https://github.com/lmliam/remote-monitor/actions/workflows/build.yml/badge.svg)](https://github.com/lmliam/remote-monitor/actions/workflows/build.yml)

Terminal UI for monitoring a remote Linux host over SSH.

`remote-monitor` streams a small Bash sampler to the target host and renders CPU, memory, disk, network, process, optional power-source, and optional NVIDIA, AMD, or Intel GPU metrics locally. The remote machine does not need a daemon, agent, or checked-out copy of this repository.

## Features

- Live Bubble Tea dashboard with aurora, basic, and Windows XP-inspired themes.
- Non-interactive text output when stdout is not a TTY.
- SSH reconnects with keepalive and control socket reuse.
- Rolling history for load, pressure, memory, disks, network, GPU, and temperatures.
- Linux host sampling from `/proc`, `/sys`, `df`, `ps`, `awk`, and optional GPU tooling.
- Strict CI with `gofmt`, `go vet`, `go test`, `golangci-lint`, ShellCheck, shfmt, and compile checks.

## Requirements

- Go 1.26 or newer.
- Local `ssh` client.
- SSH access to a Linux host with Bash and common core utilities.
- Optional NVIDIA GPU metrics from `nvidia-smi` on the remote host.
- Optional AMD GPU metrics from `amd-smi`, `rocm-smi`, or `/sys/class/drm` on the remote host.
- Optional Intel GPU metrics from `intel_gpu_top`, `xpu-smi`, or `/sys/class/drm` on the remote host.
- Optional battery, AC, USB, mains, and UPS metrics from `/sys/class/power_supply` on Linux remote hosts.

## Install

With Homebrew:

```sh
brew install LMLiam/tap/remote-monitor
```

With Go:

```sh
go install github.com/lmliam/remote-monitor/cmd/remote-monitor@latest
```

For local development:

```sh
git clone https://github.com/lmliam/remote-monitor.git
cd remote-monitor
go run ./cmd/remote-monitor --help
```

## Usage

Pass the SSH target as a positional argument or with `-host`.

```sh
remote-monitor user@example-host
remote-monitor -host gpu-box -interval 2
remote-monitor -theme basic -compact user@example-host
remote-monitor -theme windows-xp user@example-host
remote-monitor gpu-box -output jsonl -out samples.jsonl
remote-monitor gpu-box -process-sort mem -process-filter postgres -process-count 20
remote-monitor gpu-box --once
remote-monitor gpu-box --once -output jsonl -out snapshot.jsonl
```

You can also set `REMOTE_MONITOR_HOST` and run without a host argument.

```sh
export REMOTE_MONITOR_HOST=user@example-host
remote-monitor
```

Named profiles can keep host-specific defaults in TOML. By default, `remote-monitor` reads `$XDG_CONFIG_HOME/remote-monitor/config.toml`, falling back to `$HOME/.config/remote-monitor/config.toml`. Use `-config` to choose a different file.

```sh
remote-monitor -profile gpu-box
remote-monitor -profile gpu-box -interval 2 -theme basic
remote-monitor -config ~/.config/remote-monitor/config.toml -profile uni-server
```

```toml
[profiles.gpu-box]
host = "user@gpu-box"
interval = 2
history = 600
stale_after = 7
theme = "aurora"
compact = true
no_banner = false
no_truecolor = false
ssh_connect_timeout = 5
ssh_server_alive = 5
ssh_server_alive_count = 2
ssh_control_persist = 30
cpu_critical_percent = 95
cpu_warn_temp = 75
cpu_critical_temp = 85
ram_warn_available_percent = 15
ram_critical_available_percent = 5
gpu_warn_temp = 70
gpu_critical_temp = 80
vram_warn_percent = 85
vram_critical_percent = 95
disk_warn_percent = 90
disk_critical_percent = 95
```

Profile keys use snake case: `host`, `interval`, `history`, `stale_after`, `reconnect_delay`, `fps`, `theme`, `compact`, `no_banner`, `no_truecolor`, `ssh_connect_timeout`, `ssh_server_alive`, `ssh_server_alive_count`, `ssh_control_persist`, `cpu_critical_percent`, `cpu_warn_temp`, `cpu_critical_temp`, `ram_warn_available_percent`, `ram_critical_available_percent`, `gpu_warn_temp`, `gpu_critical_temp`, `vram_warn_percent`, `vram_critical_percent`, `disk_warn_percent`, and `disk_critical_percent`. Unknown keys, missing profiles, invalid TOML, and invalid profile values fail before monitoring starts.

Precedence is: explicit CLI flags and positional host, then the selected profile, then environment variables, then built-in defaults. Unset profile keys fall through to the environment and built-in defaults.

Useful flags:

| Flag | Environment variable | Default |
| --- | --- | --- |
| `-profile` | none | disabled |
| `-config` | none | `$XDG_CONFIG_HOME/remote-monitor/config.toml` or `$HOME/.config/remote-monitor/config.toml` |
| `-host` | `REMOTE_MONITOR_HOST` | required |
| `-once` | none | disabled; write one sample and exit |
| `-output` | none | auto (`tui` on TTY stdout, `text` on non-TTY stdout) |
| `-out` | none | disabled; supported with `-output jsonl` |
| `-process-sort` | none | `cpu` (`cpu`, `mem`) |
| `-process-filter` | none | disabled |
| `-process-count` | none | `4` |
| `-net-include` | none | disabled; comma-separated interface names or glob patterns |
| `-net-exclude` | none | disabled; comma-separated interface names or glob patterns |
| `-net-aggregate` | none | disabled; replace selected interfaces with one aggregate row |
| `-interval` | `MONITOR_INTERVAL` | `1` second |
| `-history` | `MONITOR_HISTORY_LIMIT` | `240` samples |
| `-stale-after` | `MONITOR_STALE_AFTER` | `interval * 3 + 1` seconds |
| `-reconnect-delay` | `MONITOR_RECONNECT_DELAY` | `2` seconds |
| `-fps` | `MONITOR_FPS` | `12` |
| `-compact` | `MONITOR_COMPACT` | `false` |
| `-no-banner` | `MONITOR_NO_BANNER` | `false` |
| `-theme` | `MONITOR_THEME` | `aurora` (`aurora`, `basic`, `windows-xp`; aliases: `xp`, `winxp`) |
| `-no-truecolor` | `MONITOR_NO_TRUECOLOR` | `false` |
| `-ssh-connect-timeout` | `MONITOR_SSH_CONNECT_TIMEOUT` | `5` seconds |
| `-ssh-server-alive` | `MONITOR_SSH_ALIVE_INTERVAL` | `5` seconds |
| `-ssh-server-alive-count` | `MONITOR_SSH_ALIVE_COUNT` | `2` |
| `-ssh-control-persist` | `MONITOR_SSH_CONTROL_PERSIST` | `30` seconds |
| `-cpu-critical-percent` | `MONITOR_CPU_CRITICAL_PERCENT` | `95` |
| `-cpu-warn-temp` | `MONITOR_CPU_WARN_TEMP` | `75` C |
| `-cpu-critical-temp` | `MONITOR_CPU_CRITICAL_TEMP` | `85` C |
| `-ram-warn-available-percent` | `MONITOR_RAM_WARN_AVAILABLE_PERCENT` | `15` |
| `-ram-critical-available-percent` | `MONITOR_RAM_CRITICAL_AVAILABLE_PERCENT` | `5` |
| `-gpu-warn-temp` | `MONITOR_GPU_WARN_TEMP` | `70` C |
| `-gpu-critical-temp` | `MONITOR_GPU_CRITICAL_TEMP` | `80` C |
| `-vram-warn-percent` | `MONITOR_VRAM_WARN_PERCENT` | `85` |
| `-vram-critical-percent` | `MONITOR_VRAM_CRITICAL_PERCENT` | `95` |
| `-disk-warn-percent` | `MONITOR_DISK_WARN_PERCENT` | `90` |
| `-disk-critical-percent` | `MONITOR_DISK_CRITICAL_PERCENT` | `95` |

Process rows are sorted by descending CPU by default, preserving the original
dashboard behavior. Use `-process-sort mem` to sort by descending resident
memory instead. `-process-filter` applies a case-insensitive substring match
against the process command name and full command line exposed by `ps`; the
displayed process column remains the command name. Filtering is applied before
the `-process-count` row limit.

Threshold options control when the dashboard colors values and raises alert
summary text. `cpu_critical_percent` marks high CPU utilization as critical.
`cpu_warn_temp` and `cpu_critical_temp` control CPU thermal warnings. RAM
availability thresholds fire when available memory falls at or below the given
percent, so `ram_warn_available_percent` must be greater than
`ram_critical_available_percent`. GPU temperature, VRAM utilization, and disk
usage thresholds use warn/critical pairs where the warning value must be lower
than the critical value. Percent thresholds must be 0-100; temperature
thresholds must be 0-150 C.

## Network Interface Selection

By default, network collection and display preserve the sampler's current
interface behavior. Use `-net-include` and `-net-exclude` to focus network
metrics on specific interfaces:

```sh
remote-monitor gpu-box -net-include eth0,wlan0
remote-monitor gpu-box -net-exclude lo,docker*,br-*
remote-monitor gpu-box -net-include en*,eth* -net-aggregate
remote-monitor gpu-box -net-exclude lo,docker*,br-* -net-aggregate
```

Patterns are comma-separated interface names or simple glob patterns. Include
patterns are applied first; if an include list is present, only matching
interfaces are eligible. Exclude patterns are then applied to that eligible set.
With no include or exclude flags, all sampled network interfaces remain visible.

Empty pattern segments and malformed glob patterns fail before monitoring
starts. Patterns that are valid but match no sampled interfaces are allowed; the
network list is then empty. `-net-aggregate` replaces per-interface rows with a
single `aggregate` interface row whose receive/transmit rates, packet rates,
link speeds, drops, errors, and overruns are summed across the selected
interfaces. TUI, non-interactive text, and JSONL output all use the same
selected interface set.

## JSONL Export

Use `-output jsonl` to write one machine-readable JSON object for each parsed sample. Without `-out`, JSONL is written to stdout and stdout contains only JSON objects, one per line. With `-out samples.jsonl`, the file is created or truncated before the SSH stream starts, and JSONL is written to that file while stdout stays empty.

```sh
remote-monitor gpu-box -output jsonl
remote-monitor gpu-box -output jsonl -out samples.jsonl
```

JSONL exports use the normalized local schema `remote-monitor.normalized_sample.v1`, derived from `internal/core.Sample` after sampler parsing rather than the raw remote sampler JSON. Fields are snake_case and include host, CPU, memory, pressure, swap, disk, TCP, filesystem, network, process, power, GPU, and local `received_at` values. Repeated values such as `net`, `filesystems`, `cpu_core_usage`, `top_processes`, `gpu_processes`, `gpus`, and `power_supplies` are JSON arrays. Lifecycle and reconnect state are not included in the JSONL stream.

## Snapshot Mode

Use `--once` when scripts, CI jobs, cron jobs, or incident notes need exactly one remote sample. Snapshot mode starts the normal SSH sampler, waits for the first valid parsed sample, writes that sample, and exits. Without an explicit `-output`, `--once` uses the script-friendly text summary even when stdout is a TTY.

```sh
remote-monitor gpu-box --once
remote-monitor gpu-box --once -output text
remote-monitor gpu-box --once -output jsonl
remote-monitor gpu-box --once -output jsonl -out snapshot.jsonl
```

`--once -output jsonl` writes exactly one JSON object followed by a newline. With `-out snapshot.jsonl`, the file is created or truncated before the SSH stream starts and stdout remains empty. `--once -output tui` is not supported; use continuous TUI mode without `--once`.

## WSL Host Metrics

When the remote sampler detects WSL, it can call `powershell.exe` or `pwsh.exe` from inside WSL to fill host metrics that Linux paths do not expose. It checks the WSL `PATH` first and then standard Windows PowerShell install paths, which helps SSH sessions where Windows directories are not exported. It currently probes Windows CPU name, logical core count, current and max CPU clocks, physical RAM totals and availability, and CPU temperature when a hardware monitor exposes CPU package/core sensors through LibreHardwareMonitor or OpenHardwareMonitor WMI. Windows ACPI thermal zones are ignored because they are often chassis or firmware zones rather than CPU package sensors.

Set `REMOTE_MONITOR_WSL_HOST_METRICS=0` in the remote WSL environment to disable Windows host probing. Set `REMOTE_MONITOR_WSL_HOST_METRICS_TIMEOUT` to change the PowerShell timeout from the default `2s`.

## Power Metrics

Power collection is best-effort and uses the Linux power supply class under `/sys/class/power_supply`. Hosts without that directory, without power-supply devices, or with unreadable fields still emit valid samples and omit the power panel from text and TUI output.

Remote Monitor reports all discovered supplies in structured JSON under `power_supplies`, including the sysfs device name, `type`, `online`, `capacity_percent`, `status`, `power_draw_w`, and `present` when available. Summary fields also expose `external_power_online`, `battery_percent`, `battery_status`, `power_draw_w`, `ups_present`, and `power_source_name` for simple alerting.

External power is online when any `Mains`, AC, USB, wireless, or UPS-style supply reports `online=1`. Battery summary percent uses the lowest reported battery percentage when multiple batteries are present, which keeps the concise dashboard biased toward the pack most likely to need attention. Estimated draw uses `power_now` in microwatts when exposed; otherwise it falls back to `current_now * voltage_now` using the kernel power-supply microamp and microvolt units. UPS presence is reported when any supply has `type=UPS`.

Known limitations: battery health, wear, cycle count, charge thresholds, shutdown policy, vendor UPS daemons, macOS power data, and Windows power data are outside the current scope. Kernel and firmware support varies, so missing fields use the normal unavailable sentinels instead of failing sampling.

## GPU Metrics

GPU collection is best-effort and vendor tooling is optional. Hosts without supported GPU tools or exposed GPU devices still emit valid samples with an empty GPU list.

NVIDIA metrics use `nvidia-smi` when it is available. Remote Monitor collects device identity, utilization, memory, thermals, power, fan, clock, PCIe, throttle, performance-state, and top compute-process rows supported by the installed driver.

AMD metrics use these sources in priority order:

| Source | Typical stack | Metrics |
| --- | --- | --- |
| `amd-smi metric --json` | AMD SMI / ROCm on Linux | device identity, utilization, memory used/total/utilization, thermals, power, fan, graphics and memory clocks, PCIe link fields, throttle text, and performance state when exposed |
| `rocm-smi --json` | legacy ROCm SMI packages | device identity, utilization, memory used/total/utilization, edge temperature, power, fan, active graphics and memory clocks, and performance state when exposed |
| `/sys/class/drm` | kernel fallback | AMD device identity plus available VRAM, temperature, power, fan, graphics/memory clocks, and performance state files |

AMD platforms vary by ROCm version, driver stack, permissions, and ASIC. `amd-smi` is preferred because AMD documents it as the replacement for `rocm-smi`; `rocm-smi` remains a fallback for hosts that still package it. Missing tools, unavailable JSON fields, unsupported hardware counters, and unreadable sysfs files are treated as unavailable metrics rather than sampler failures. Windows AMD GPU collection is not currently supported.

Intel metrics merge these sources when they are available:

| Source | Typical stack | Metrics |
| --- | --- | --- |
| `intel_gpu_top -J -s 100 -n 1 -o -` | i915 / intel-gpu-tools | render, compute, media, overall utilization, graphics clock, and GPU power when exposed |
| `xpu-smi discovery --dump` and `xpu-smi dump` | Intel discrete GPU / Level Zero stacks | device identity, UUID, memory total/used/utilization, utilization, temperature, power, graphics/media clocks, media utilization, and throttle reason when exposed |
| `/sys/class/drm` | kernel fallback | Intel device identity plus available VRAM, temperature, power limit, and graphics clocks |

Intel platforms vary in what the kernel and tools expose. XPU-SMI devices are de-duplicated with matching DRM sysfs devices by PCI BDF so mixed Intel systems can report both discrete and integrated GPUs. Unsupported metrics use the same sentinel values as other GPU collectors and hidden vendor-detail rows are omitted from the dashboard. `intel_gpu_top` may require perf counter access; unsupported hardware, missing permissions, absent tools, or dashed tool values are treated as unavailable metrics. Windows Intel GPU collection is not currently supported.

## Development

Install dependencies and the local Git hooks:

```sh
go mod download
make setup
```

Run the local checks before pushing:

```sh
make check
```

The GitHub Actions workflow runs the same native, shell, workflow-helper, and
integration-tagged checks. The ShellCheck target uses Docker to run CI's pinned image.
Individual targets such as `make fmt`, `make scripts`, `make test`, and `make lint`
are available for targeted validation.

Sampler module assembly and collector test guidance lives in
[internal/transport/sampler/README.md](internal/transport/sampler/README.md). Run
`make generate` after editing `internal/transport/sampler/` modules.

Architecture and package layout notes live in
[docs/architecture.md](docs/architecture.md).

Release instructions live in [docs/releasing.md](docs/releasing.md).

## Contributing

Contributions are welcome when they fit the SSH-based terminal monitoring scope. Please read [CONTRIBUTING.md](CONTRIBUTING.md), [GOVERNANCE.md](GOVERNANCE.md), and the [Code of Conduct](CODE_OF_CONDUCT.md) before opening larger issues or pull requests.

## Support

For usage questions, bug reports, feature requests, and private security reports, see [SUPPORT.md](SUPPORT.md).

## Security

Please do not open public issues for sensitive security reports. See [SECURITY.md](SECURITY.md) for supported versions, scope, and reporting guidance.

## License

`remote-monitor` is released under the [MIT License](LICENSE).
