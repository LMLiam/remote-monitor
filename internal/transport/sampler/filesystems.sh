read_root_usage() {
  df -kP / | awk 'NR==2 { gsub("%", "", $5); printf "%s|%s|%s\n", $1, $3, $2 "|" $5 }'
}

refresh_root_usage_cache() {
  root_usage_cache="$(read_root_usage)"
}

cached_root_usage() {
  if [ -z "${root_usage_cache}" ] || [ $((sample_index % filesystem_refresh_samples)) -eq 0 ]; then
    refresh_root_usage_cache
  fi

  printf '%s\n' "${root_usage_cache}"
}

collect_filesystems_json() {
  local source mount used total used_pct inode_pct comma=''
  declare -A inode_pct_by_mount=()

  while IFS='|' read -r mount inode_pct; do
    [ -n "${mount}" ] || continue
    inode_pct_by_mount["${mount}"]="${inode_pct}"
  done < <(df -iP 2>/dev/null | awk '
    NR > 1 {
      gsub("%", "", $5)
      printf "%s|%s\n", $6, $5 + 0
    }
  ')

  printf '['
  while IFS='|' read -r source mount used total used_pct; do
    [ -n "${source}" ] || continue
    [ -n "${mount}" ] || continue
    if [ "${source}" = 'tmpfs' ] || [ "${source}" = 'devtmpfs' ]; then
      continue
    fi
    inode_pct="${inode_pct_by_mount[${mount}]:--1}"
    printf '%s{"source":"%s","mount":"%s","used_kib":%s,"total_kib":%s,"used_percent":%s,"inodes_used_percent":%s}' \
      "${comma}" \
      "$(json_escape "${source}")" \
      "$(json_escape "${mount}")" \
      "${used}" \
      "${total}" \
      "${used_pct}" \
      "${inode_pct}"
    comma=','
  done < <(df -kP 2>/dev/null | awk '
    NR > 1 {
      gsub("%", "", $5)
      printf "%s|%s|%s|%s|%s\n", $1, $6, $3 + 0, $2 + 0, $5 + 0
    }
  ')
  printf ']'
}

build_filesystems_json() {
  if [ -z "${filesystems_json_cache}" ] || [ $((sample_index % filesystem_refresh_samples)) -eq 0 ]; then
    filesystems_json_cache="$(collect_filesystems_json)"
  fi

  printf '%s' "${filesystems_json_cache}"
}
