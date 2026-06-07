# Platform Notes

Remote Monitor runs locally wherever the Go binary and SSH client run, but the
remote sampler targets Linux hosts with Bash and common core utilities. Optional
metrics depend on what the remote host exposes.

## WSL Host Metrics

When the remote sampler detects WSL, it can call `powershell.exe` or `pwsh.exe`
from inside WSL to fill host metrics that Linux paths do not expose. It checks
the WSL `PATH` first and then standard Windows PowerShell install paths, which
helps SSH sessions where Windows directories are not exported.

WSL host probing currently checks Windows CPU name, logical core count, current
and max CPU clocks, physical RAM totals and availability, and CPU temperature
when a hardware monitor exposes CPU package/core sensors through
LibreHardwareMonitor or OpenHardwareMonitor WMI. Windows ACPI thermal zones are
ignored because they are often chassis or firmware zones rather than CPU package
sensors.

Set `REMOTE_MONITOR_WSL_HOST_METRICS=0` in the remote WSL environment to disable
Windows host probing. Set `REMOTE_MONITOR_WSL_HOST_METRICS_TIMEOUT` to change
the PowerShell timeout from the default `2s`.

## Power Metrics

Power collection is best-effort and uses the Linux power supply class under
`/sys/class/power_supply`. Hosts without that directory, without power-supply
devices, or with unreadable fields still emit valid samples and omit the power
panel from text and TUI output.

Remote Monitor reports all discovered supplies in structured JSON under
`power_supplies`, including the sysfs device name, `type`, `online`,
`capacity_percent`, `status`, `power_draw_w`, and `present` when available.
Summary fields also expose `external_power_online`, `battery_percent`,
`battery_status`, `power_draw_w`, `ups_present`, and `power_source_name` for
simple alerting.

External power is online when any `Mains`, AC, USB, wireless, or UPS-style
supply reports `online=1`. Battery summary percent uses the lowest reported
battery percentage when multiple batteries are present, which keeps the concise
dashboard biased toward the pack most likely to need attention.

Estimated draw uses `power_now` in microwatts when exposed; otherwise it falls
back to `current_now * voltage_now` using the kernel power-supply microamp and
microvolt units. UPS presence is reported when any supply has `type=UPS`.

Known limitations: battery health, wear, cycle count, charge thresholds,
shutdown policy, vendor UPS daemons, macOS power data, and Windows power data
are outside the current scope. Kernel and firmware support varies, so missing
fields use the normal unavailable sentinels instead of failing sampling.

## Other Platform Considerations

- GPU collection is Linux-focused; Windows AMD and Intel GPU collection is not
  currently supported.
- Optional GPU tools must be installed on the remote host, not the local
  machine.
- Shell environments for SSH sessions can differ from interactive login shells,
  so optional tools must be discoverable in the remote command environment.
