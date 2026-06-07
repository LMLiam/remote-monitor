# Network Filtering

By default, network collection and display preserve the sampler's current
interface behavior. Use `-net-include` and `-net-exclude` to focus network
metrics on specific interfaces.

```sh
remote-monitor gpu-box -net-include eth0,wlan0
remote-monitor gpu-box -net-exclude lo,docker*,br-*
remote-monitor gpu-box -net-include en*,eth* -net-aggregate
remote-monitor gpu-box -net-exclude lo,docker*,br-* -net-aggregate
```

## Pattern Syntax

Patterns are comma-separated interface names or simple glob patterns. They use
Go `path.Match` syntax, so `*` matches any string, `?` matches one character,
and character classes such as `[0-9]` are supported.

Include patterns are applied first. If an include list is present, only matching
interfaces are eligible. Exclude patterns are then applied to that eligible set.
With no include or exclude flags, all sampled network interfaces remain visible.

Examples:

| Pattern | Meaning |
| --- | --- |
| `eth0` | Match only `eth0`. |
| `eth*` | Match interfaces such as `eth0` and `eth1`. |
| `en*` | Match common predictable Ethernet names such as `enp3s0`. |
| `wl*` | Match common wireless interface names. |
| `docker*` | Match Docker bridge interfaces. |
| `br-*` | Match Linux bridge interfaces with a `br-` prefix. |

Empty pattern segments and malformed glob patterns fail before monitoring
starts. Patterns that are valid but match no sampled interfaces are allowed; the
network list is then empty.

## Aggregate Mode

`-net-aggregate` replaces per-interface rows with a single `aggregate` interface
row. Receive/transmit rates, packet rates, link speeds, drops, errors, and
overruns are summed across the selected interfaces.

TUI, non-interactive text, and JSONL output all use the same selected interface
set, so filters behave consistently across output modes.
