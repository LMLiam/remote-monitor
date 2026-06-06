unknown_disk_sample='-1|-1|-1|-1|-1|-1|-1|-1|-1|-1|-1'
diskstats_path="${REMOTE_MONITOR_DISKSTATS_FILE:-/proc/diskstats}"
block_sys_path="${REMOTE_MONITOR_BLOCK_SYS_DIR:-/sys/class/block}"
declare -A prev_disk_sample=()

print_unknown_disk_sample() {
  printf '%s\n' "${unknown_disk_sample}"
}

disk_device_in_stats() {
  local device="$1"

  [ -n "${device}" ] || return 1
  [ -r "${diskstats_path}" ] || return 1

  awk -v dev="${device}" '$3 == dev { found=1; exit } END { exit found ? 0 : 1 }' "${diskstats_path}" 2>/dev/null
}

read_disk_sample() {
  local device="$1"

  if [ -z "${device}" ] || [ ! -r "${diskstats_path}" ]; then
    print_unknown_disk_sample
    return
  fi

  awk -v dev="${device}" '
    $3 == dev {
      printf "%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n", $6, $10, $13, $4, $5, $7, $8, $9, $11, $12, $14
      found=1
    }
    END {
      if (!found) {
        printf "%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n", -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1
      }
    }
  ' "${diskstats_path}"
}

add_tracked_disk_device() {
  local device="$1"
  local i

  if ! disk_device_in_stats "${device}"; then
    return
  fi

  for ((i = 0; i < ${#tracked_disk_devices[@]}; i++)); do
    if [ "${tracked_disk_devices[i]}" = "${device}" ]; then
      return
    fi
  done

  tracked_disk_devices+=("${device}")
}

block_device_name_from_source() {
  local source="$1"
  local resolved=''

  resolved="$(readlink -f "${source}" 2>/dev/null || true)"
  case "${resolved}" in
    /dev/*)
      printf '%s\n' "${resolved##*/}"
      return
      ;;
  esac

  printf '%s\n' "${source##*/}"
}

parent_partition_device() {
  local device="$1"

  case "${device}" in
    dm-*)
      printf '%s\n' "${device}"
      return
      ;;
  esac

  if [[ "${device}" =~ ^(.+)p[0-9]+$ ]]; then
    printf '%s\n' "${BASH_REMATCH[1]}"
    return
  fi

  if [[ "${device}" =~ ^(.+[^0-9])[0-9]+$ ]]; then
    printf '%s\n' "${BASH_REMATCH[1]}"
    return
  fi

  printf '%s\n' "${device}"
}

emit_backing_block_devices() {
  local device="$1"
  local device_path="${block_sys_path}/${device}"
  local slave_path
  local slave_device
  local parent_device
  local emitted=0

  if [ -d "${device_path}/slaves" ]; then
    for slave_path in "${device_path}/slaves/"*; do
      [ -e "${slave_path}" ] || continue
      slave_device="${slave_path##*/}"
      emit_backing_block_devices "${slave_device}"
      emitted=1
    done
    if [ "${emitted}" -eq 1 ]; then
      return
    fi
  fi

  if [ -e "${device_path}/partition" ]; then
    parent_device="$(parent_partition_device "${device}")"
    if [ -n "${parent_device}" ] && [ "${parent_device}" != "${device}" ]; then
      printf '%s\n' "${parent_device}"
      return
    fi
  fi

  printf '%s\n' "${device}"
}

discover_disk_devices() {
  local source
  local source_device
  local backing_device

  tracked_disk_devices=()
  while read -r source; do
    case "${source}" in
      /dev/*)
        ;;
      *)
        continue
        ;;
    esac

    source_device="$(block_device_name_from_source "${source}")"
    while read -r backing_device; do
      add_tracked_disk_device "${backing_device}"
    done < <(emit_backing_block_devices "${source_device}")
  done < <(df -kP 2>/dev/null | awk 'NR > 1 { print $1 }')
}

reset_disk_baseline_for_device() {
  local device="$1"

  prev_disk_sample["${device}"]="${unknown_disk_sample}"
}

unset_disk_baseline_for_device() {
  local device="$1"

  unset "prev_disk_sample[${device}]"
}

prime_disk_baselines() {
  local device

  for device in "${tracked_disk_devices[@]}"; do
    prev_disk_sample["${device}"]="$(read_disk_sample "${device}")"
  done
}

refresh_tracked_disk_devices() {
  local device
  local old_devices=("${tracked_disk_devices[@]}")
  declare -A was_tracked=()
  declare -A still_tracked=()

  for device in "${old_devices[@]}"; do
    was_tracked["${device}"]=1
  done

  discover_disk_devices

  for device in "${tracked_disk_devices[@]}"; do
    still_tracked["${device}"]=1
    if [ -z "${was_tracked[${device}]:-}" ]; then
      reset_disk_baseline_for_device "${device}"
    fi
  done

  for device in "${!prev_disk_sample[@]}"; do
    if [ -z "${still_tracked[${device}]:-}" ]; then
      unset_disk_baseline_for_device "${device}"
    fi
  done
}

disk_refresh_sample_count() {
  local samples="${filesystem_refresh_samples:-1}"

  case "${samples}" in
    '' | *[!0-9]* | 0)
      samples=1
      ;;
  esac

  printf '%s\n' "${samples}"
}

refresh_disk_devices_if_needed() {
  local refresh_samples

  refresh_samples="$(disk_refresh_sample_count)"
  if [ "${sample_index:-0}" -gt 0 ] && [ $((sample_index % refresh_samples)) -eq 0 ]; then
    refresh_tracked_disk_devices
  fi
}

build_disk_metric_fields() {
  local device="$1"
  local current_sample="$2"
  local previous_sample="$3"
  local elapsed_ms="$4"
  local disk_sectors_read disk_sectors_written disk_io_ms
  local disk_reads_completed disk_reads_merged disk_read_ms disk_writes_completed disk_writes_merged disk_write_ms disk_in_flight disk_weighted_ms
  local prev_disk_sectors_read_value prev_disk_sectors_written_value prev_disk_io_ms_value
  local prev_disk_reads_completed_value prev_disk_reads_merged_value prev_disk_read_ms_value prev_disk_writes_completed_value
  local prev_disk_writes_merged_value prev_disk_write_ms_value _prev_disk_in_flight_value prev_disk_weighted_ms_value
  local disk_ops_delta disk_service_ms_delta
  local read_bps='-1'
  local write_bps='-1'
  local read_merged_per_sec='-1'
  local write_merged_per_sec='-1'
  local disk_util='-1'
  local disk_await='-1'
  local disk_queue='-1'

  IFS='|' read -r disk_sectors_read disk_sectors_written disk_io_ms disk_reads_completed disk_reads_merged disk_read_ms disk_writes_completed disk_writes_merged disk_write_ms disk_in_flight disk_weighted_ms <<<"${current_sample}"
  IFS='|' read -r prev_disk_sectors_read_value prev_disk_sectors_written_value prev_disk_io_ms_value prev_disk_reads_completed_value prev_disk_reads_merged_value prev_disk_read_ms_value prev_disk_writes_completed_value prev_disk_writes_merged_value prev_disk_write_ms_value _prev_disk_in_flight_value prev_disk_weighted_ms_value <<<"${previous_sample}"

  if [ "${disk_sectors_read}" -ge 0 ] && [ "${prev_disk_sectors_read_value}" -ge 0 ]; then
    read_bps=$((((disk_sectors_read - prev_disk_sectors_read_value) * 512 * 1000) / elapsed_ms))
    if [ "${read_bps}" -lt 0 ]; then
      read_bps=0
    fi
  fi

  if [ "${disk_sectors_written}" -ge 0 ] && [ "${prev_disk_sectors_written_value}" -ge 0 ]; then
    write_bps=$((((disk_sectors_written - prev_disk_sectors_written_value) * 512 * 1000) / elapsed_ms))
    if [ "${write_bps}" -lt 0 ]; then
      write_bps=0
    fi
  fi

  if [ "${disk_io_ms}" -ge 0 ] && [ "${prev_disk_io_ms_value}" -ge 0 ]; then
    disk_util=$((((disk_io_ms - prev_disk_io_ms_value) * 100) / elapsed_ms))
    if [ "${disk_util}" -lt 0 ]; then
      disk_util=0
    elif [ "${disk_util}" -gt 100 ]; then
      disk_util=100
    fi
  fi

  if [ "${disk_reads_merged}" -ge 0 ] && [ "${prev_disk_reads_merged_value}" -ge 0 ]; then
    read_merged_per_sec=$((((disk_reads_merged - prev_disk_reads_merged_value) * 1000) / elapsed_ms))
    if [ "${read_merged_per_sec}" -lt 0 ]; then
      read_merged_per_sec=0
    fi
  fi

  if [ "${disk_writes_merged}" -ge 0 ] && [ "${prev_disk_writes_merged_value}" -ge 0 ]; then
    write_merged_per_sec=$((((disk_writes_merged - prev_disk_writes_merged_value) * 1000) / elapsed_ms))
    if [ "${write_merged_per_sec}" -lt 0 ]; then
      write_merged_per_sec=0
    fi
  fi

  if [ "${disk_reads_completed}" -ge 0 ] && [ "${prev_disk_reads_completed_value}" -ge 0 ] &&
    [ "${disk_writes_completed}" -ge 0 ] && [ "${prev_disk_writes_completed_value}" -ge 0 ] &&
    [ "${disk_read_ms}" -ge 0 ] && [ "${prev_disk_read_ms_value}" -ge 0 ] &&
    [ "${disk_write_ms}" -ge 0 ] && [ "${prev_disk_write_ms_value}" -ge 0 ]; then
    disk_ops_delta=$(((disk_reads_completed - prev_disk_reads_completed_value) + (disk_writes_completed - prev_disk_writes_completed_value)))
    disk_service_ms_delta=$(((disk_read_ms - prev_disk_read_ms_value) + (disk_write_ms - prev_disk_write_ms_value)))
    if [ "${disk_ops_delta}" -gt 0 ]; then
      disk_await="$(awk -v service_ms="${disk_service_ms_delta}" -v ops="${disk_ops_delta}" 'BEGIN { printf "%.2f", service_ms / ops }')"
    else
      disk_await='0.00'
    fi
  fi

  if [ "${disk_weighted_ms}" -ge 0 ] && [ "${prev_disk_weighted_ms_value}" -ge 0 ]; then
    disk_queue="$(awk -v weighted_ms="$((disk_weighted_ms - prev_disk_weighted_ms_value))" -v elapsed_ms="${elapsed_ms}" 'BEGIN {
      if (elapsed_ms <= 0 || weighted_ms < 0) {
        printf "%.2f", 0
      } else {
        printf "%.2f", weighted_ms / elapsed_ms
      }
    }')"
  fi

  printf '"device":"%s","read_bps":%s,"write_bps":%s,"read_merged_per_sec":%s,"write_merged_per_sec":%s,"util_percent":%s,"await_ms":%s,"queue_depth":%s,"inflight":%s' \
    "$(json_escape "${device}")" \
    "${read_bps}" \
    "${write_bps}" \
    "${read_merged_per_sec}" \
    "${write_merged_per_sec}" \
    "${disk_util}" \
    "${disk_await}" \
    "${disk_queue}" \
    "$(normalize_int "${disk_in_flight}")"
}

build_disk_json() {
  local root_source root_used_kib root_total_kib root_used_pct combined
  local current_sample previous_sample metric_fields
  local elapsed_ms

  elapsed_ms="$(elapsed_ms_or_default)"

  IFS='|' read -r root_source root_used_kib combined < <(cached_root_usage)
  root_total_kib="${combined%%|*}"
  root_used_pct="${combined##*|}"
  current_sample="$(read_disk_sample "${root_device}")"
  previous_sample="${prev_disk_sectors_read}|${prev_disk_sectors_written}|${prev_disk_io_ms}|${prev_disk_reads_completed}|${prev_disk_reads_merged}|${prev_disk_read_ms}|${prev_disk_writes_completed}|${prev_disk_writes_merged}|${prev_disk_write_ms}|-1|${prev_disk_weighted_ms}"
  metric_fields="$(build_disk_metric_fields "${root_device}" "${current_sample}" "${previous_sample}" "${elapsed_ms}")"

  printf '{"root_source":"%s","root_used_kib":%s,"root_total_kib":%s,"root_used_percent":%s,%s}' \
    "$(json_escape "${root_source}")" \
    "${root_used_kib}" \
    "${root_total_kib}" \
    "${root_used_pct}" \
    "${metric_fields}"

  IFS='|' read -r prev_disk_sectors_read prev_disk_sectors_written prev_disk_io_ms prev_disk_reads_completed prev_disk_reads_merged prev_disk_read_ms prev_disk_writes_completed prev_disk_writes_merged prev_disk_write_ms _prev_disk_in_flight prev_disk_weighted_ms <<<"${current_sample}"
}

build_disks_json() {
  local device
  local current_sample
  local previous_sample
  local metric_fields
  local elapsed_ms
  local comma=''

  refresh_disk_devices_if_needed
  elapsed_ms="$(elapsed_ms_or_default)"

  printf '['
  for device in "${tracked_disk_devices[@]}"; do
    current_sample="$(read_disk_sample "${device}")"
    previous_sample="${prev_disk_sample[${device}]:-${unknown_disk_sample}}"
    metric_fields="$(build_disk_metric_fields "${device}" "${current_sample}" "${previous_sample}" "${elapsed_ms}")"

    printf '%s{%s}' "${comma}" "${metric_fields}"
    comma=','
    prev_disk_sample["${device}"]="${current_sample}"
  done
  printf ']'
}
