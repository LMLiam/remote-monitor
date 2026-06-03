read_top_process_snapshot() {
  if ! command -v ps >/dev/null 2>&1; then
    return
  fi

  local sort_spec="-pcpu,-rss"
  if [ "${process_sort:-cpu}" = "mem" ]; then
    sort_spec="-rss,-pcpu"
  fi

  LC_ALL=C ps -eo pid=,pcpu=,rss=,comm=,args= --sort="${sort_spec}" 2>/dev/null | awk -v self="$$" -v limit="${process_count:-4}" -v filter="${process_filter:-}" '
    BEGIN {
      count = 0
      filter = tolower(filter)
    }
    {
      pid = $1
      cpu = $2 + 0
      rss = $3 + 0
      cmd = $4
      args = ""
      for (i = 5; i <= NF; i++) {
        args = args (args == "" ? "" : " ") $i
      }

      if (pid == self || cmd == "") {
        next
      }
      if (cmd ~ /^(ps|awk|sort|head|tail|sleep|sampler\.sh)$/) {
        next
      }
      if (filter != "" && index(tolower(cmd " " args), filter) == 0) {
        next
      }

      cpu_int = int(cpu + 0.5)
      if (cpu_int < 0) {
        cpu_int = 0
      }
      rss_mib = int(rss / 1024)
      if (rss_mib < 0) {
        rss_mib = 0
      }

      printf "%s|%s|%s|%s\n", pid, cpu_int, rss_mib, cmd
      count++
      if (count >= limit) {
        exit
      }
    }
  '
}

build_top_process_json() {
  local pid cpu rss cmd
  local comma=''

  printf '['
  while IFS='|' read -r pid cpu rss cmd; do
    [ -n "${pid}" ] || continue
    printf '%s{"pid":%s,"command":"%s","cpu_percent":%s,"rss_mib":%s}' \
      "${comma}" \
      "${pid}" \
      "$(json_escape "${cmd}")" \
      "${cpu}" \
      "${rss}"
    comma=','
  done < <(read_top_process_snapshot)
  printf ']'
}
