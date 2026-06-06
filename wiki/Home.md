# Remote Monitor Wiki

Remote Monitor is a terminal UI for monitoring a remote Linux host over SSH. It
streams a small Bash sampler to the target host and renders CPU, memory, disk,
network, process, optional power-source, and optional NVIDIA, AMD, or Intel GPU
metrics locally. The remote machine does not need a daemon, agent, or checked-out
copy of this repository.

## Start Here

- [Configuration Reference](Configuration-Reference) - flags, environment
  variables, defaults, profiles, thresholds, and precedence.
- [Output Formats](Output-Formats) - TUI, text, JSONL, auto selection, and
  one-shot snapshots.
- [Network Filtering](Network-Filtering) - include/exclude patterns, glob
  examples, and aggregate network rows.

## Metrics Reference

- [GPU Metrics](GPU-Metrics) - NVIDIA, AMD, and Intel collection sources,
  requirements, and limitations.
- [Platform Notes](Platform-Notes) - WSL host metrics, power metrics, and
  platform-specific behavior.

## Contributors

- [Development Guide](Development-Guide) - local setup, checks, sampler module
  structure, code generation, and architecture links.

## Repository-Managed Sync

This wiki is managed from the main repository's `wiki/` directory. The
`Publish Wiki` workflow syncs repository content to the GitHub wiki after
changes land on `main`. Direct edits made in the GitHub wiki UI can be
overwritten by the next repository-to-wiki sync, so update wiki pages in a pull
request instead.
