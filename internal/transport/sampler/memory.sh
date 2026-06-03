read_swap_stats() {
  awk '
    /^SwapFree:/ { swap_free=$2 }
    /^SwapTotal:/ { swap_total=$2 }
    END {
      printf "%s|%s\n", swap_free+0, swap_total+0
    }
  ' /proc/meminfo
}

read_swap_io_sample() {
  awk '
    /^pswpin / { in_pages=$2 }
    /^pswpout / { out_pages=$2 }
    END {
      printf "%s|%s\n", in_pages+0, out_pages+0
    }
  ' /proc/vmstat 2>/dev/null
}

read_ram_stats() {
  awk '
    /^MemTotal:/ { total_kib=$2 }
    /^MemAvailable:/ { avail_kib=$2 }
    /^MemFree:/ { free_kib=$2 }
    /^Buffers:/ { buffers_kib=$2 }
    /^Cached:/ { cached_kib=$2 }
    /^SReclaimable:/ { reclaimable_kib=$2 }
    /^Shmem:/ { shmem_kib=$2 }
    END {
      if (avail_kib == 0) {
        avail_kib = free_kib + buffers_kib + cached_kib + reclaimable_kib - shmem_kib
      }
      if (avail_kib < 0) {
        avail_kib = 0
      }
      total_mib = int(total_kib / 1024)
      used_mib = int((total_kib - avail_kib) / 1024)
      cache_mib = int((cached_kib + buffers_kib + reclaimable_kib) / 1024)
      if (used_mib < 0) {
        used_mib = 0
      }
      if (cache_mib < 0) {
        cache_mib = 0
      }
      printf "%s|%s|%s|%s|%s|%s|%s|%s\n", used_mib, total_mib, int(avail_kib / 1024), int(free_kib / 1024), int(cached_kib / 1024), int(buffers_kib / 1024), int(reclaimable_kib / 1024), int(shmem_kib / 1024)
    }
  ' /proc/meminfo
}
