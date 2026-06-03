trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "${value}"
}

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  value="${value//$'\f'/\\f}"
  value="${value//$'\b'/\\b}"
  printf '%s' "${value}"
}

normalize_int() {
  local value
  value="$(trim "${1:-}")"
  case "${value}" in
    ''|N/A|n/a)
      printf '%s' '-1'
      return
      ;;
  esac

  if [[ "${value}" =~ ^-?[0-9]+$ ]]; then
    printf '%s' "${value}"
    return
  fi

  printf '%s' '-1'
}

normalize_float() {
  local value
  value="$(trim "${1:-}")"
  case "${value}" in
    ''|N/A|n/a)
      printf '%s' '-1'
      return
      ;;
  esac

  if [[ "${value}" =~ ^-?[0-9]+([.][0-9]+)?$ ]]; then
    printf '%s' "${value}"
    return
  fi

  printf '%s' '-1'
}
