# GPU Metrics

GPU collection is best-effort and vendor tooling is optional. Hosts without
supported GPU tools or exposed GPU devices still emit valid samples with an
empty GPU list.

## NVIDIA

NVIDIA metrics use `nvidia-smi` when it is available on the remote host.

| Category | Metrics |
| --- | --- |
| Identity | index, UUID, GPU name |
| Utilization | core, memory, encoder, and decoder utilization |
| Memory | used memory, total memory, and memory utilization |
| Thermals and power | temperature, power draw, power limit, and fan percent |
| Clocks | SM, maximum SM, memory, maximum memory, graphics, and video clocks |
| PCIe | current and maximum PCIe generation and link width |
| Driver state | throttle reasons and performance state |
| Processes | top compute-process rows with GPU UUID, PID, command, and used memory |

Requirements and limitations:

- `nvidia-smi` must be installed and visible in the remote SSH session `PATH`.
- Older drivers may omit some query fields; unavailable values use normal
  sentinel values rather than failing sampling.
- GPU process rows depend on driver support for compute process reporting.

## AMD

AMD metrics use these sources in priority order:

| Source | Typical stack | Metrics |
| --- | --- | --- |
| `amd-smi metric --json` | AMD SMI / ROCm on Linux | device identity, utilization, memory used/total/utilization, thermals, power, fan, graphics and memory clocks, PCIe link fields, throttle text, and performance state when exposed |
| `rocm-smi --json` | legacy ROCm SMI packages | device identity, utilization, memory used/total/utilization, edge temperature, power, fan, active graphics and memory clocks, and performance state when exposed |
| `/sys/class/drm` | kernel fallback | AMD device identity plus available VRAM, temperature, power, fan, graphics/memory clocks, and performance state files |

Requirements and limitations:

- `amd-smi` is preferred because AMD documents it as the replacement for
  `rocm-smi`; `rocm-smi` remains a fallback for hosts that still package it.
- Platform output varies by ROCm version, driver stack, permissions, and ASIC.
- Missing tools, unavailable JSON fields, unsupported counters, and unreadable
  sysfs files are treated as unavailable metrics rather than sampler failures.
- Windows AMD GPU collection is not currently supported.

## Intel

Intel metrics merge these sources when they are available:

| Source | Typical stack | Metrics |
| --- | --- | --- |
| `intel_gpu_top -J -s 100 -n 1 -o -` | i915 / intel-gpu-tools | render, compute, media, overall utilization, graphics clock, and GPU power when exposed |
| `xpu-smi discovery --dump` and `xpu-smi dump` | Intel discrete GPU / Level Zero stacks | device identity, UUID, memory total/used/utilization, utilization, temperature, power, graphics/media clocks, media utilization, and throttle reason when exposed |
| `/sys/class/drm` | kernel fallback | Intel device identity plus available VRAM, temperature, power limit, and graphics clocks |

Requirements and limitations:

- Intel platforms vary in what the kernel and tools expose.
- XPU-SMI devices are de-duplicated with matching DRM sysfs devices by PCI BDF
  so mixed Intel systems can report both discrete and integrated GPUs.
- `intel_gpu_top` may require perf counter access.
- Unsupported hardware, missing permissions, absent tools, or dashed tool values
  are treated as unavailable metrics.
- Windows Intel GPU collection is not currently supported.
