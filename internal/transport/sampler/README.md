# Remote sampler modules

Remote Monitor still sends one Bash script to the target host. The source for
that script lives in this directory so each collector can stay small enough to
test and review.

## Assembly

`manifest.txt` defines the module order. Run this command from the repository
root after changing any sampler module:

```sh
bash internal/transport/sampler/assemble.sh
```

The command rewrites `internal/transport/sampler.sh`, which remains the file
embedded by `internal/transport/sampler.go`.

`go generate ./internal/transport` runs the same assembly step.

## Testing

Run focused sampler checks with:

```sh
go test ./internal/transport
```

Those tests verify that the embedded script matches the manifest assembly, that
the embedded script passes `bash -n`, and that collector modules can be sourced
without running the long-lived sampler loop.

Run the shell checks used by CI with:

```bash
go run mvdan.cc/sh/v3/cmd/shfmt@v3.13.1 -i 2 -ci -d .github/scripts internal/transport/sampler internal/transport/sampler.sh tests/e2e/ssh-target
shellcheck -S warning -s bash .github/scripts/*.sh tests/e2e/ssh-target/*.sh internal/transport/sampler/assemble.sh internal/transport/sampler.sh
sampler_modules=()
while IFS= read -r module || [ -n "${module}" ]; do
  case "${module}" in
    '' | '#'*)
      continue
      ;;
  esac
  sampler_modules+=("internal/transport/sampler/${module}")
done <internal/transport/sampler/manifest.txt
shellcheck -S warning -s bash -e SC2034,SC2154 "${sampler_modules[@]}"
```

The sampler module ShellCheck command disables `SC2034` and `SC2154` only for
module fragments. The modules are sourced into `sampler.sh`, so direct module
analysis cannot see cross-module state that is assigned in one file and read in
another. The assembled `sampler.sh` is checked without those disables.

Run the CI-equivalent project suite before publishing sampler changes. The
integration-tagged test run includes the SSH E2E package and requires Docker:

```sh
go vet -tags=integration ./...
go test -tags=integration ./...
```

## Module boundaries

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
- `gpu_common.sh`: shared GPU JSON array helpers and final vendor output combiner.
- `gpu_nvidia.sh`: NVIDIA GPU metrics and GPU process metrics.
- `gpu_intel.sh`: Intel GPU metrics from `xpu-smi`, `intel_gpu_top`, and `/sys/class/drm` fallbacks.
- `gpu_amd.sh`: AMD GPU metrics from `amd-smi metric --json`, `rocm-smi --json`, and `/sys/class/drm` fallbacks.
- `main.sh`: sampler initialization, timing loop, and final JSON assembly.
