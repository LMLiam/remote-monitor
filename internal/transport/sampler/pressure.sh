read_pressure_avg10() {
  local path="$1"
  local some='-1'
  local full='-1'
  local line

  if [ ! -r "${path}" ]; then
    printf '%s|%s\n' "${some}" "${full}"
    return
  fi

  while read -r line; do
    case "${line}" in
      some*)
        some="$(printf '%s\n' "${line}" | awk '{
          for (i = 1; i <= NF; i++) {
            if ($i ~ /^avg10=/) {
              sub(/^avg10=/, "", $i)
              print $i
              exit
            }
          }
        }')"
        ;;
      full*)
        full="$(printf '%s\n' "${line}" | awk '{
          for (i = 1; i <= NF; i++) {
            if ($i ~ /^avg10=/) {
              sub(/^avg10=/, "", $i)
              print $i
              exit
            }
          }
        }')"
        ;;
    esac
  done <"${path}"

  printf '%s|%s\n' "$(normalize_float "${some}")" "$(normalize_float "${full}")"
}
