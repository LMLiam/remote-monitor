remote_name="$(hostname)"
remote_cpu_cores="$(nproc 2>/dev/null || getconf _NPROCESSORS_ONLN 2>/dev/null || printf '0')"
remote_cpu_name="$(read_cpu_model_name)"
root_source="$(df -P / | awk 'NR==2 {print $1}')"
root_device=''
page_size_bytes="$(getconf PAGESIZE 2>/dev/null || printf '4096')"
nvidia_smi_path="$(command -v nvidia-smi 2>/dev/null || true)"
intel_gpu_top_path="$(command -v intel_gpu_top 2>/dev/null || true)"
xpu_smi_path="$(command -v xpu-smi 2>/dev/null || true)"
amd_smi_path="$(command -v amd-smi 2>/dev/null || true)"
rocm_smi_path="$(command -v rocm-smi 2>/dev/null || true)"
intel_drm_class_path="${REMOTE_MONITOR_DRM_CLASS_DIR:-/sys/class/drm}"
amd_drm_class_path="${REMOTE_MONITOR_DRM_CLASS_DIR:-/sys/class/drm}"
filesystem_refresh_samples="$(refresh_samples_for_seconds "${filesystem_refresh_seconds}")"
sample_index=0
root_usage_cache=''
filesystems_json_cache=''

case "${remote_cpu_cores}" in
  ''|*[!0-9]*)
    remote_cpu_cores='0'
    ;;
esac

case "${root_source}" in
  /dev/*)
    root_device="${root_source#/dev/}"
    ;;
esac

cpu_labels=()
cpu_idle=()
cpu_total=()
cpu_user=()
cpu_system=()
cpu_iowait=()
cpu_steal=()
prev_cpu_labels=()
prev_cpu_idle=()
prev_cpu_total=()
prev_cpu_user=()
prev_cpu_system=()
prev_cpu_iowait=()
prev_cpu_steal=()
tracked_net_ifaces=()
prev_net_rx=()
prev_net_tx=()
prev_net_rx_packets=()
prev_net_tx_packets=()
prev_net_rx_drops=()
prev_net_rx_errors=()
prev_net_rx_overruns=()
prev_net_tx_drops=()
prev_net_tx_errors=()
prev_net_tx_overruns=()
prev_disk_sectors_read='-1'
prev_disk_sectors_written='-1'
prev_disk_io_ms='-1'
prev_disk_reads_completed='-1'
prev_disk_reads_merged='-1'
prev_disk_read_ms='-1'
prev_disk_writes_completed='-1'
prev_disk_writes_merged='-1'
prev_disk_write_ms='-1'
prev_disk_in_flight='-1'
prev_disk_weighted_ms='-1'
prev_swap_in_pages='-1'
prev_swap_out_pages='-1'
prev_tcp_retrans='-1'
prev_tcp_resets='-1'

read_cpu_snapshot
prev_cpu_labels=("${cpu_labels[@]}")
prev_cpu_idle=("${cpu_idle[@]}")
prev_cpu_total=("${cpu_total[@]}")
prev_cpu_user=("${cpu_user[@]}")
prev_cpu_system=("${cpu_system[@]}")
prev_cpu_iowait=("${cpu_iowait[@]}")
prev_cpu_steal=("${cpu_steal[@]}")
discover_net_ifaces

for ((i = 0; i < ${#tracked_net_ifaces[@]}; i++)); do
  IFS='|' read -r prev_net_rx[i] prev_net_tx[i] prev_net_rx_packets[i] prev_net_tx_packets[i] prev_net_rx_drops[i] prev_net_rx_errors[i] prev_net_rx_overruns[i] prev_net_tx_drops[i] prev_net_tx_errors[i] prev_net_tx_overruns[i] < <(read_net_sample "${tracked_net_ifaces[i]}")
done

IFS='|' read -r prev_disk_sectors_read prev_disk_sectors_written prev_disk_io_ms prev_disk_reads_completed prev_disk_reads_merged prev_disk_read_ms prev_disk_writes_completed prev_disk_writes_merged prev_disk_write_ms prev_disk_in_flight prev_disk_weighted_ms < <(read_disk_sample "${root_device}")
IFS='|' read -r prev_swap_in_pages prev_swap_out_pages < <(read_swap_io_sample)
IFS='|' read -r prev_tcp_retrans prev_tcp_resets < <(read_tcp_counter_sample)
prev_sample_ns="$(now_ns)"
sample_elapsed_ms=$((interval * 1000))
next_tick_ns="${prev_sample_ns}"

while true; do
  current_ns="$(now_ns)"
  if [ "${next_tick_ns}" -gt "${current_ns}" ]; then
    sleep_ns=$((next_tick_ns - current_ns))
    sleep "$(awk -v ns="${sleep_ns}" 'BEGIN { printf "%.3f\n", ns / 1000000000 }')"
  fi
  next_tick_ns=$((next_tick_ns + interval_ns))
  sample_now_ns="$(now_ns)"
  sample_elapsed_ms=$(((sample_now_ns - prev_sample_ns) / 1000000))
  if [ "${sample_elapsed_ms}" -lt 1 ]; then
    sample_elapsed_ms=$((interval * 1000))
  fi
  sample_index=$((sample_index + 1))

  read_cpu_snapshot
  cpu_pct=0
  cpu_user_pct=-1
  cpu_system_pct=-1
  cpu_iowait_pct=-1
  cpu_steal_pct=-1
  swap_in_bps='-1'
  swap_out_bps='-1'
  tcp_retrans_per_sec='-1'
  tcp_resets_per_sec='-1'

  if [ "${#cpu_labels[@]}" -gt 0 ] && [ "${#prev_cpu_total[@]}" -gt 0 ] && [ "${cpu_labels[0]}" = "${prev_cpu_labels[0]}" ]; then
    diff_idle=$((cpu_idle[0] - prev_cpu_idle[0]))
    diff_total=$((cpu_total[0] - prev_cpu_total[0]))
    if [ "${diff_total}" -gt 0 ]; then
      cpu_pct=$(((100 * (diff_total - diff_idle)) / diff_total))
      cpu_user_pct=$((((cpu_user[0] - prev_cpu_user[0]) * 100) / diff_total))
      cpu_system_pct=$((((cpu_system[0] - prev_cpu_system[0]) * 100) / diff_total))
      cpu_iowait_pct=$((((cpu_iowait[0] - prev_cpu_iowait[0]) * 100) / diff_total))
      cpu_steal_pct=$((((cpu_steal[0] - prev_cpu_steal[0]) * 100) / diff_total))
    fi
  fi

  IFS='|' read -r ram_used ram_total ram_available ram_free ram_cache ram_buffers ram_reclaimable ram_shared < <(read_ram_stats)
  IFS='|' read -r cpu_freq_mhz cpu_max_freq_mhz < <(read_cpu_freq_stats)
  cpu_temp_c="$(read_cpu_temp_c)"
  IFS='|' read -r cpu_pressure_some cpu_pressure_full < <(read_pressure_avg10 /proc/pressure/cpu)
  IFS='|' read -r mem_pressure_some mem_pressure_full < <(read_pressure_avg10 /proc/pressure/memory)
  wsl_host_metrics_json="$(read_wsl_windows_host_metrics_json)"
  apply_wsl_host_metrics
  IFS='|' read -r swap_free_kib swap_total_kib < <(read_swap_stats)
  IFS='|' read -r swap_in_pages swap_out_pages < <(read_swap_io_sample)
  IFS='|' read -r tcp_retrans_counter tcp_resets_counter < <(read_tcp_counter_sample)
  read -r load1 load5 load15 _ < /proc/loadavg
  uptime_s="$(awk '{printf "%d\n", $1}' /proc/uptime)"
  epoch_now="$(date +%s)"
  stamp_now="$(date '+%F %T')"

  if [ "${swap_in_pages}" -ge 0 ] && [ "${prev_swap_in_pages}" -ge 0 ]; then
    swap_in_bps=$((((swap_in_pages - prev_swap_in_pages) * page_size_bytes * 1000) / sample_elapsed_ms))
    if [ "${swap_in_bps}" -lt 0 ]; then
      swap_in_bps=0
    fi
  fi

  if [ "${swap_out_pages}" -ge 0 ] && [ "${prev_swap_out_pages}" -ge 0 ]; then
    swap_out_bps=$((((swap_out_pages - prev_swap_out_pages) * page_size_bytes * 1000) / sample_elapsed_ms))
    if [ "${swap_out_bps}" -lt 0 ]; then
      swap_out_bps=0
    fi
  fi

  if [ "${tcp_retrans_counter}" -ge 0 ] && [ "${prev_tcp_retrans}" -ge 0 ]; then
    tcp_retrans_per_sec=$((((tcp_retrans_counter - prev_tcp_retrans) * 1000) / sample_elapsed_ms))
    if [ "${tcp_retrans_per_sec}" -lt 0 ]; then
      tcp_retrans_per_sec=0
    fi
  fi

  if [ "${tcp_resets_counter}" -ge 0 ] && [ "${prev_tcp_resets}" -ge 0 ]; then
    tcp_resets_per_sec=$((((tcp_resets_counter - prev_tcp_resets) * 1000) / sample_elapsed_ms))
    if [ "${tcp_resets_per_sec}" -lt 0 ]; then
      tcp_resets_per_sec=0
    fi
  fi

  disk_json="$(build_disk_json)"
  net_json="$(build_net_json)"
  filesystems_json="$(build_filesystems_json)"
  cpu_core_json="$(build_cpu_core_json)"
  top_process_json="$(build_top_process_json)"
  gpu_process_json="$(build_gpu_process_json)"
  gpu_json="$(build_gpu_json)"

  printf '{"version":1,"epoch":%s,"timestamp":"%s","remote":"%s","uptime_seconds":%s,"load1":%s,"load5":%s,"load15":%s,"cpu_cores":%s,"cpu_name":"%s","cpu_percent":%s,"cpu_user_percent":%s,"cpu_system_percent":%s,"cpu_iowait_percent":%s,"cpu_steal_percent":%s,"ram_used_mib":%s,"ram_total_mib":%s,"ram_available_mib":%s,"ram_free_mib":%s,"ram_cache_mib":%s,"ram_buffers_mib":%s,"ram_reclaimable_mib":%s,"ram_shared_mib":%s,"cpu_freq_mhz":%s,"cpu_max_freq_mhz":%s,"cpu_temp_c":%s,"cpu_pressure_some_avg10":%s,"cpu_pressure_full_avg10":%s,"mem_pressure_some_avg10":%s,"mem_pressure_full_avg10":%s,"swap":{"free_kib":%s,"total_kib":%s,"in_bps":%s,"out_bps":%s},"disk":%s,"net":%s,"filesystems":%s,"tcp_retrans_segs_per_sec":%s,"tcp_resets_per_sec":%s,"cpu_core_usage":%s,"top_processes":%s,"gpu_processes":%s,"gpus":%s}\n' \
    "${epoch_now}" \
    "$(json_escape "${stamp_now}")" \
    "$(json_escape "${remote_name}")" \
    "${uptime_s}" \
    "${load1}" \
    "${load5}" \
    "${load15}" \
    "${remote_cpu_cores}" \
    "$(json_escape "${remote_cpu_name}")" \
    "${cpu_pct}" \
    "${cpu_user_pct}" \
    "${cpu_system_pct}" \
    "${cpu_iowait_pct}" \
    "${cpu_steal_pct}" \
    "${ram_used}" \
    "${ram_total}" \
    "${ram_available}" \
    "${ram_free}" \
    "${ram_cache}" \
    "${ram_buffers}" \
    "${ram_reclaimable}" \
    "${ram_shared}" \
    "${cpu_freq_mhz}" \
    "${cpu_max_freq_mhz}" \
    "${cpu_temp_c}" \
    "${cpu_pressure_some}" \
    "${cpu_pressure_full}" \
    "${mem_pressure_some}" \
    "${mem_pressure_full}" \
    "${swap_free_kib}" \
    "${swap_total_kib}" \
    "${swap_in_bps}" \
    "${swap_out_bps}" \
    "${disk_json}" \
    "${net_json}" \
    "${filesystems_json}" \
    "${tcp_retrans_per_sec}" \
    "${tcp_resets_per_sec}" \
    "${cpu_core_json}" \
    "${top_process_json}" \
    "${gpu_process_json}" \
    "${gpu_json}"

  prev_cpu_labels=("${cpu_labels[@]}")
  prev_cpu_idle=("${cpu_idle[@]}")
  prev_cpu_total=("${cpu_total[@]}")
  prev_cpu_user=("${cpu_user[@]}")
  prev_cpu_system=("${cpu_system[@]}")
  prev_cpu_iowait=("${cpu_iowait[@]}")
  prev_cpu_steal=("${cpu_steal[@]}")
  prev_swap_in_pages="${swap_in_pages}"
  prev_swap_out_pages="${swap_out_pages}"
  prev_tcp_retrans="${tcp_retrans_counter}"
  prev_tcp_resets="${tcp_resets_counter}"
  prev_sample_ns="${sample_now_ns}"

  current_ns="$(now_ns)"
  if [ "${next_tick_ns}" -lt "${current_ns}" ]; then
    next_tick_ns="${current_ns}"
  fi
done
