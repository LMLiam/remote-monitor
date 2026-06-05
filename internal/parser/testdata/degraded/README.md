These fixtures are sampler-shaped JSON samples for degraded remote hosts.

- `missing-optional-tools.json`: optional GPU, temperature, pressure, and power collectors are unavailable while core CPU, memory, disk, network, filesystem, and process data remain valid.
- `restricted-proc-sys.json`: `/proc` or `/sys` data is present but unreadable, so parser-visible fields use the sampler's sentinel values.
- `partial-sections-wsl.json`: WSL host metrics are unavailable and partial network, filesystem, and process sections remain parseable.
