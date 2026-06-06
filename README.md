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

- Go 1.26, the current latest stable Go target for this project. Tracking the
  latest stable release keeps builds on current toolchain improvements and
  security fixes; the Charm v2 dependency stack currently imposes a Go 1.25
  floor.
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
```

## Documentation

The wiki contains comprehensive reference material:
<https://github.com/lmliam/remote-monitor/wiki>.

- [Configuration Reference](https://github.com/lmliam/remote-monitor/wiki/Configuration-Reference) covers flags, environment variables, profiles, defaults, thresholds, and precedence.
- [Output Formats](https://github.com/lmliam/remote-monitor/wiki/Output-Formats) covers TUI, text, JSONL, and one-shot snapshots.
- [Network Filtering](https://github.com/lmliam/remote-monitor/wiki/Network-Filtering) covers include/exclude glob patterns and aggregate network rows.
- [GPU Metrics](https://github.com/lmliam/remote-monitor/wiki/GPU-Metrics) covers NVIDIA, AMD, and Intel collection sources and limitations.
- [Platform Notes](https://github.com/lmliam/remote-monitor/wiki/Platform-Notes) covers WSL host metrics, power metrics, and platform-specific behavior.
- [Development Guide](https://github.com/lmliam/remote-monitor/wiki/Development-Guide) covers local setup, checks, sampler modules, and code generation.

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
integration-tagged checks. Individual targets such as `make fmt`,
`make scripts`, `make test`, and `make lint` are available for targeted
validation.

Sampler module guidance lives in
[internal/transport/sampler/README.md](internal/transport/sampler/README.md).
Architecture and package layout notes live in
[docs/architecture.md](docs/architecture.md). Release instructions live in
[docs/releasing.md](docs/releasing.md).

## Contributing

Contributions are welcome when they fit the SSH-based terminal monitoring scope. Please read [CONTRIBUTING.md](CONTRIBUTING.md), [GOVERNANCE.md](GOVERNANCE.md), and the [Code of Conduct](CODE_OF_CONDUCT.md) before opening larger issues or pull requests.

## Support

For usage questions, bug reports, feature requests, and private security reports, see [SUPPORT.md](SUPPORT.md).

## Security

Please do not open public issues for sensitive security reports. See [SECURITY.md](SECURITY.md) for supported versions, scope, and reporting guidance.

## License

`remote-monitor` is released under the [MIT License](LICENSE).
