# remote-monitor

[![Build](https://github.com/lmliam/remote-monitor/actions/workflows/build.yml/badge.svg)](https://github.com/lmliam/remote-monitor/actions/workflows/build.yml)

Terminal UI for monitoring a remote Linux host over SSH.

`remote-monitor` streams a small Bash sampler to the target host and renders CPU, memory, disk, network, process, and optional NVIDIA GPU metrics locally. The remote machine does not need a daemon, agent, or checked-out copy of this repository.

## Features

- Live Bubble Tea dashboard with aurora and basic themes.
- Non-interactive text output when stdout is not a TTY.
- SSH reconnects with keepalive and control socket reuse.
- Rolling history for load, pressure, memory, disks, network, GPU, and temperatures.
- Linux host sampling from `/proc`, `/sys`, `df`, `ps`, `awk`, and optional `nvidia-smi`.
- Strict native Go CI with `gofmt`, `go vet`, `go test`, `golangci-lint`, and compile checks.

## Requirements

- Go 1.26 or newer.
- Local `ssh` client.
- SSH access to a Linux host with Bash and common core utilities.
- Optional NVIDIA GPU metrics when `nvidia-smi` is available on the remote host.

## Install

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
```

You can also set `REMOTE_MONITOR_HOST` and run without a host argument.

```sh
export REMOTE_MONITOR_HOST=user@example-host
remote-monitor
```

Useful flags:

| Flag | Environment variable | Default |
| --- | --- | --- |
| `-host` | `REMOTE_MONITOR_HOST` | required |
| `-interval` | `MONITOR_INTERVAL` | `1` second |
| `-history` | `MONITOR_HISTORY_LIMIT` | `240` samples |
| `-stale-after` | `MONITOR_STALE_AFTER` | `interval * 3 + 1` seconds |
| `-reconnect-delay` | `MONITOR_RECONNECT_DELAY` | `2` seconds |
| `-fps` | `MONITOR_FPS` | `12` |
| `-compact` | `MONITOR_COMPACT` | `false` |
| `-no-banner` | `MONITOR_NO_BANNER` | `false` |
| `-theme` | `MONITOR_THEME` | `aurora` |
| `-no-truecolor` | `MONITOR_NO_TRUECOLOR` | `false` |
| `-ssh-connect-timeout` | `MONITOR_SSH_CONNECT_TIMEOUT` | `5` seconds |
| `-ssh-server-alive` | `MONITOR_SSH_ALIVE_INTERVAL` | `5` seconds |
| `-ssh-server-alive-count` | `MONITOR_SSH_ALIVE_COUNT` | `2` |
| `-ssh-control-persist` | `MONITOR_SSH_CONTROL_PERSIST` | `30` seconds |

## Development

Run the local checks before pushing:

```sh
unformatted="$(gofmt -l ./cmd ./internal)"
test -z "$unformatted" || { echo "$unformatted"; exit 1; }
go vet ./...
go test ./...
golangci-lint run
go build -o remote-monitor ./cmd/remote-monitor
```

The GitHub Actions workflow runs the same native checks.
