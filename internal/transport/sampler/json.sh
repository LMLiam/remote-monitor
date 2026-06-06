trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "${value}"
}

json_control_chars=($'\001' $'\002' $'\003' $'\004' $'\005' $'\006' $'\007' $'\013' $'\016' $'\017' $'\020' $'\021' $'\022' $'\023' $'\024' $'\025' $'\026' $'\027' $'\030' $'\031' $'\032' $'\033' $'\034' $'\035' $'\036' $'\037')
json_control_escapes=('\u0001' '\u0002' '\u0003' '\u0004' '\u0005' '\u0006' '\u0007' '\u000b' '\u000e' '\u000f' '\u0010' '\u0011' '\u0012' '\u0013' '\u0014' '\u0015' '\u0016' '\u0017' '\u0018' '\u0019' '\u001a' '\u001b' '\u001c' '\u001d' '\u001e' '\u001f')

json_escape() {
  local value="$1"
  local i
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  value="${value//$'\f'/\\f}"
  value="${value//$'\b'/\\b}"
  for i in "${!json_control_chars[@]}"; do
    value="${value//${json_control_chars[$i]}/${json_control_escapes[$i]}}"
  done
  printf '%s' "${value}"
}

normalize_int() {
  local value="${1:-}"
  # Keep normalize_* trimming inline; these helpers run for every numeric sampler
  # field, and calling trim via command substitution adds one subshell per field.
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  case "${value}" in
    '' | N/A | n/a)
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
  local value="${1:-}"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  case "${value}" in
    '' | N/A | n/a)
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
