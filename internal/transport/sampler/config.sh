set -euo pipefail

interval="${1:-1}"
case "${interval}" in
  ''|*[!0-9]*|0)
    interval=1
    ;;
esac
interval_ns=$((interval * 1000000000))
filesystem_refresh_seconds=10

refresh_samples_for_seconds() {
  local seconds="$1"
  local samples

  samples=$(((seconds + interval - 1) / interval))
  if [ "${samples}" -lt 1 ]; then
    samples=1
  fi

  printf '%s\n' "${samples}"
}
