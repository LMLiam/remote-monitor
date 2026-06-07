# Configuration Reference

Remote Monitor can be configured with environment variables, a TOML profile,
explicit CLI flags, and an optional positional host. Built-in defaults are used
first; then values are applied from environment variables, the selected profile,
explicit CLI flags, and finally the positional host.

## Flags, Environment, and Defaults

| Flag | Environment variable | Default |
| --- | --- | --- |
| `-profile` | none | disabled |
| `-config` | none | `$XDG_CONFIG_HOME/remote-monitor/config.toml` or `$HOME/.config/remote-monitor/config.toml` |
| `-host` | `REMOTE_MONITOR_HOST` | required |
| `-version` | none | disabled; print version information and exit |
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

Process rows are sorted by descending CPU by default. Use `-process-sort mem` to
sort by descending resident memory. `-process-filter` applies a case-insensitive
substring match against the command name and full command line exposed by `ps`.
Filtering is applied before the `-process-count` row limit.

## Profile Files

Named profiles keep host-specific defaults in TOML. By default, Remote Monitor
reads `$XDG_CONFIG_HOME/remote-monitor/config.toml`, falling back to
`$HOME/.config/remote-monitor/config.toml`. Use `-config` to choose a different
file and `-profile` to select a profile.

```toml
[profiles.gpu-box]
host = "user@gpu-box"
interval = 2
history = 600
stale_after = 7
reconnect_delay = 2
fps = 12
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

Supported profile keys are `host`, `interval`, `history`, `stale_after`,
`reconnect_delay`, `fps`, `theme`, `compact`, `no_banner`, `no_truecolor`,
`ssh_connect_timeout`, `ssh_server_alive`, `ssh_server_alive_count`,
`ssh_control_persist`, `cpu_critical_percent`, `cpu_warn_temp`,
`cpu_critical_temp`, `ram_warn_available_percent`,
`ram_critical_available_percent`, `gpu_warn_temp`, `gpu_critical_temp`,
`vram_warn_percent`, `vram_critical_percent`, `disk_warn_percent`, and
`disk_critical_percent`.

Unknown keys, missing profiles, invalid TOML, empty profile hosts, unsupported
themes, and invalid profile values fail before monitoring starts. Theme values
support `aurora`, `basic`, `windows-xp`, and the `xp` or `winxp` aliases.

## Threshold Semantics

Thresholds control dashboard coloring and alert summary text. CPU utilization
has a critical threshold. CPU temperature and GPU temperature use warning and
critical thresholds in Celsius, where the warning value must be lower than the
critical value.

RAM availability thresholds fire when available memory falls at or below the
configured percent. Because lower available RAM is worse, the warning value must
be greater than the critical value. GPU VRAM and disk usage thresholds fire as
usage rises, so their warning values must be lower than their critical values.

Percent thresholds must be between `0` and `100`. Temperature thresholds must be
between `0` and `150` C.

## Precedence

Configuration is resolved from lowest to highest precedence:

1. Built-in defaults.
2. Environment variables.
3. The selected TOML profile.
4. Explicit CLI flags.
5. Positional host argument.

Unset profile keys fall through to environment variables and built-in defaults.
The positional host argument overrides `-host` when both are provided.
