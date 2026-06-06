# Development Guide

This page covers the practical contributor workflow. For deeper package
boundaries, data flow, ownership, and testing strategy, read
[docs/architecture.md](https://github.com/lmliam/remote-monitor/blob/main/docs/architecture.md).

## Setup

Use Go 1.26, the project's current latest stable Go target. Tracking the latest
stable release keeps local builds on current toolchain improvements and security
fixes; the Charm v2 dependency stack currently imposes a Go 1.25 floor.

Install dependencies and local Git hooks:

```sh
go mod download
make setup
```

`make setup` installs the commit-msg hook that validates conventional commit
subjects locally.

## Local Checks

Run the full local gate before pushing:

```sh
make check
```

`make check` runs Go formatting, shell formatting, ShellCheck, workflow helper
script tests, `go vet`, integration-tagged tests, `golangci-lint`, and a build.
The ShellCheck target uses Docker to run CI's pinned image. The
integration-tagged test run includes the SSH end-to-end test, which self-skips
when Docker is unavailable.

Targeted checks are also available:

```sh
make fmt
make shfmt
make shellcheck
make scripts
make vet
make test
make lint
make build
```

## Sampler Modules

The remote sampler is assembled from Bash source modules in
`internal/transport/sampler/`. The assembled script is
`internal/transport/sampler.sh`, which is embedded into the transport package
and streamed to the remote host over SSH.

Module boundaries:

- `config.sh`: runtime defaults and refresh interval helpers.
- `json.sh`: shared string escaping and numeric normalization.
- `cpu.sh`: CPU snapshots, clocks, model name, temperature, and core JSON.
- `processes.sh`: top process sampling, filtering, sorting, count limiting, and JSON.
- `memory.sh`: RAM and swap counters.
- `pressure.sh`: Linux pressure stall information.
- `wsl.sh`: WSL detection and optional Windows host metrics.
- `filesystems.sh`: filesystem usage cache and JSON.
- `disk.sh`: root and mounted block-device counters and JSON.
- `network.sh`: interface discovery, network counters, and TCP counters.
- `power.sh`: Linux power-supply metrics.
- `gpu_common.sh`: shared GPU JSON array helpers and final vendor output combiner.
- `gpu_nvidia.sh`: NVIDIA GPU metrics and GPU process metrics.
- `gpu_intel.sh`: Intel GPU metrics from `xpu-smi`, `intel_gpu_top`, and `/sys/class/drm` fallbacks.
- `gpu_amd.sh`: AMD GPU metrics from `amd-smi metric --json`, `rocm-smi --json`, and `/sys/class/drm` fallbacks.
- `main.sh`: sampler initialization, timing loop, and final JSON assembly.

## Code Generation

After editing any sampler module, regenerate the embedded sampler script:

```sh
make generate
```

`make generate` runs `go generate ./internal/transport`, which assembles the
sampler from `internal/transport/sampler/manifest.txt`. The direct command is
also available:

```sh
bash internal/transport/sampler/assemble.sh
```

Commit both the module changes and the regenerated
`internal/transport/sampler.sh`.

## More Contributor References

- [Sampler module README](https://github.com/lmliam/remote-monitor/blob/main/internal/transport/sampler/README.md)
  for assembly and collector test details.
- [Architecture guide](https://github.com/lmliam/remote-monitor/blob/main/docs/architecture.md)
  for package layout, data flow, and testing strategy.
- [Release guide](https://github.com/lmliam/remote-monitor/blob/main/docs/releasing.md)
  for release preparation and publishing steps.
