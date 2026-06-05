read_disk_sample() {
  local device="$1"

  if [ -z "${device}" ]; then
    printf '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n' '-1' '-1' '-1' '-1' '-1' '-1' '-1' '-1' '-1' '-1' '-1'
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
  ' /proc/diskstats
}

build_disk_json() {
  local root_source root_used_kib root_total_kib root_used_pct
  local disk_sectors_read disk_sectors_written disk_io_ms
  local disk_reads_completed disk_reads_merged disk_read_ms disk_writes_completed disk_writes_merged disk_write_ms disk_in_flight disk_weighted_ms
  local read_bps='-1'
  local write_bps='-1'
  local read_merged_per_sec='-1'
  local write_merged_per_sec='-1'
  local disk_util='-1'
  local disk_await='-1'
  local disk_queue='-1'
  local elapsed_ms

  elapsed_ms="$(elapsed_ms_or_default)"

  IFS='|' read -r root_source root_used_kib combined < <(cached_root_usage)
  root_total_kib="${combined%%|*}"
  root_used_pct="${combined##*|}"
  IFS='|' read -r disk_sectors_read disk_sectors_written disk_io_ms disk_reads_completed disk_reads_merged disk_read_ms disk_writes_completed disk_writes_merged disk_write_ms disk_in_flight disk_weighted_ms < <(read_disk_sample "${root_device}")

  if [ "${disk_sectors_read}" -ge 0 ] && [ "${prev_disk_sectors_read}" -ge 0 ]; then
    read_bps=$((((disk_sectors_read - prev_disk_sectors_read) * 512 * 1000) / elapsed_ms))
    if [ "${read_bps}" -lt 0 ]; then
      read_bps=0
    fi
  fi

  if [ "${disk_sectors_written}" -ge 0 ] && [ "${prev_disk_sectors_written}" -ge 0 ]; then
    write_bps=$((((disk_sectors_written - prev_disk_sectors_written) * 512 * 1000) / elapsed_ms))
    if [ "${write_bps}" -lt 0 ]; then
      write_bps=0
    fi
  fi

  if [ "${disk_io_ms}" -ge 0 ] && [ "${prev_disk_io_ms}" -ge 0 ]; then
    disk_util=$((((disk_io_ms - prev_disk_io_ms) * 100) / elapsed_ms))
    if [ "${disk_util}" -lt 0 ]; then
      disk_util=0
    elif [ "${disk_util}" -gt 100 ]; then
      disk_util=100
    fi
  fi

  if [ "${disk_reads_merged}" -ge 0 ] && [ "${prev_disk_reads_merged}" -ge 0 ]; then
    read_merged_per_sec=$((((disk_reads_merged - prev_disk_reads_merged) * 1000) / elapsed_ms))
    if [ "${read_merged_per_sec}" -lt 0 ]; then
      read_merged_per_sec=0
    fi
  fi

  if [ "${disk_writes_merged}" -ge 0 ] && [ "${prev_disk_writes_merged}" -ge 0 ]; then
    write_merged_per_sec=$((((disk_writes_merged - prev_disk_writes_merged) * 1000) / elapsed_ms))
    if [ "${write_merged_per_sec}" -lt 0 ]; then
      write_merged_per_sec=0
    fi
  fi

  if [ "${disk_reads_completed}" -ge 0 ] && [ "${prev_disk_reads_completed}" -ge 0 ] &&
    [ "${disk_writes_completed}" -ge 0 ] && [ "${prev_disk_writes_completed}" -ge 0 ] &&
    [ "${disk_read_ms}" -ge 0 ] && [ "${prev_disk_read_ms}" -ge 0 ] &&
    [ "${disk_write_ms}" -ge 0 ] && [ "${prev_disk_write_ms}" -ge 0 ]; then
    disk_ops_delta=$(((disk_reads_completed - prev_disk_reads_completed) + (disk_writes_completed - prev_disk_writes_completed)))
    disk_service_ms_delta=$(((disk_read_ms - prev_disk_read_ms) + (disk_write_ms - prev_disk_write_ms)))
    if [ "${disk_ops_delta}" -gt 0 ]; then
      disk_await="$(awk -v service_ms="${disk_service_ms_delta}" -v ops="${disk_ops_delta}" 'BEGIN { printf "%.2f", service_ms / ops }')"
    else
      disk_await='0.00'
    fi
  fi

  if [ "${disk_weighted_ms}" -ge 0 ] && [ "${prev_disk_weighted_ms}" -ge 0 ]; then
    disk_queue="$(awk -v weighted_ms="$((disk_weighted_ms - prev_disk_weighted_ms))" -v elapsed_ms="${elapsed_ms}" 'BEGIN {
      if (elapsed_ms <= 0 || weighted_ms < 0) {
        printf "%.2f", 0
      } else {
        printf "%.2f", weighted_ms / elapsed_ms
      }
    }')"
  fi

  printf '{"root_source":"%s","root_used_kib":%s,"root_total_kib":%s,"root_used_percent":%s,"device":"%s","read_bps":%s,"write_bps":%s,"read_merged_per_sec":%s,"write_merged_per_sec":%s,"util_percent":%s,"await_ms":%s,"queue_depth":%s,"inflight":%s}' \
    "$(json_escape "${root_source}")" \
    "${root_used_kib}" \
    "${root_total_kib}" \
    "${root_used_pct}" \
    "$(json_escape "${root_device}")" \
    "${read_bps}" \
    "${write_bps}" \
    "${read_merged_per_sec}" \
    "${write_merged_per_sec}" \
    "${disk_util}" \
    "${disk_await}" \
    "${disk_queue}" \
    "$(normalize_int "${disk_in_flight}")"

  prev_disk_sectors_read="${disk_sectors_read}"
  prev_disk_sectors_written="${disk_sectors_written}"
  prev_disk_io_ms="${disk_io_ms}"
  prev_disk_reads_completed="${disk_reads_completed}"
  prev_disk_reads_merged="${disk_reads_merged}"
  prev_disk_read_ms="${disk_read_ms}"
  prev_disk_writes_completed="${disk_writes_completed}"
  prev_disk_writes_merged="${disk_writes_merged}"
  prev_disk_write_ms="${disk_write_ms}"
  prev_disk_weighted_ms="${disk_weighted_ms}"
}
