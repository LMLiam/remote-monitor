read_power_supply_text() {
  local file="$1"
  local value=''
  if [ -f "${file}" ] && [ -r "${file}" ]; then
    IFS= read -r value < "${file}" || true
  fi

  trim "${value}"
}

read_power_supply_int() {
  normalize_int "$(read_power_supply_text "$1")"
}

read_power_supply_watts() {
  local dir="$1"
  local microwatts microamps microvolts

  microwatts="$(read_power_supply_int "${dir}/power_now")"
  if [ "${microwatts}" -ge 0 ]; then
    awk -v value="${microwatts}" 'BEGIN { printf "%.2f", value / 1000000 }'
    return
  fi

  microamps="$(read_power_supply_int "${dir}/current_now")"
  microvolts="$(read_power_supply_int "${dir}/voltage_now")"
  if [ "${microamps}" -ge 0 ] && [ "${microvolts}" -ge 0 ]; then
    awk -v amps="${microamps}" -v volts="${microvolts}" 'BEGIN { printf "%.2f", (amps * volts) / 1000000000000 }'
    return
  fi

  printf '%s' '-1'
}

is_external_power_type() {
  local type_lc
  type_lc="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"
  case "${type_lc}" in
    mains|ac|usb|usb_*|usb-*|usb|usb_c|usb-c|usb_pd|usb-pd|wireless|ups)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

is_battery_power_type() {
  local type_lc
  type_lc="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"
  [ "${type_lc}" = "battery" ]
}

is_ups_power_type() {
  local type_lc
  type_lc="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"
  [ "${type_lc}" = "ups" ]
}

build_power_json() {
  local root="${power_supply_class_path:-/sys/class/power_supply}"
  local supplies_json=''
  local supply_count=0
  local external_online='-1'
  local battery_percent='-1'
  local battery_status=''
  local power_draw_w='-1'
  local ups_present='0'
  local source_name=''
  local first_battery_name=''

  if [ ! -d "${root}" ]; then
    printf '{"external_power_online":-1,"battery_percent":-1,"battery_status":"","power_draw_w":-1,"ups_present":-1,"source_name":"","supplies":[]}'
    return
  fi

  for supply_dir in "${root}"/*; do
    [ -d "${supply_dir}" ] || continue

    local name type online capacity status present draw supply_json
    name="$(basename "${supply_dir}")"
    type="$(read_power_supply_text "${supply_dir}/type")"
    online="$(read_power_supply_int "${supply_dir}/online")"
    capacity="$(read_power_supply_int "${supply_dir}/capacity")"
    status="$(read_power_supply_text "${supply_dir}/status")"
    present="$(read_power_supply_int "${supply_dir}/present")"
    draw="$(read_power_supply_watts "${supply_dir}")"

    supply_json="$(printf '{"name":"%s","type":"%s","online":%s,"capacity_percent":%s,"status":"%s","power_draw_w":%s,"present":%s}' \
      "$(json_escape "${name}")" \
      "$(json_escape "${type}")" \
      "${online}" \
      "${capacity}" \
      "$(json_escape "${status}")" \
      "${draw}" \
      "${present}")"
    if [ "${supply_count}" -eq 0 ]; then
      supplies_json="${supply_json}"
    else
      supplies_json="${supplies_json},${supply_json}"
    fi
    supply_count=$((supply_count + 1))

    if is_external_power_type "${type}" && [ "${online}" -ge 0 ]; then
      if [ "${online}" -eq 1 ]; then
        external_online='1'
        if [ -z "${source_name}" ]; then
          source_name="${name}"
        fi
      elif [ "${external_online}" -lt 0 ]; then
        external_online='0'
      fi
    fi

    if is_battery_power_type "${type}"; then
      if [ -z "${first_battery_name}" ]; then
        first_battery_name="${name}"
      fi
      if [ "${capacity}" -ge 0 ] && { [ "${battery_percent}" -lt 0 ] || [ "${capacity}" -lt "${battery_percent}" ]; }; then
        battery_percent="${capacity}"
        battery_status="${status}"
        source_name="${name}"
      elif [ "${battery_percent}" -lt 0 ] && [ -z "${battery_status}" ] && [ -n "${status}" ]; then
        battery_status="${status}"
        source_name="${name}"
      fi
      if [ "${power_draw_w}" = "-1" ] && [ "${draw}" != "-1" ]; then
        power_draw_w="${draw}"
      fi
    fi

    if is_ups_power_type "${type}"; then
      ups_present='1'
      if [ -z "${source_name}" ]; then
        source_name="${name}"
      fi
      if [ "${power_draw_w}" = "-1" ] && [ "${draw}" != "-1" ]; then
        power_draw_w="${draw}"
      fi
    fi
  done

  if [ "${supply_count}" -eq 0 ]; then
    ups_present='-1'
  fi
  if [ -z "${source_name}" ] && [ -n "${first_battery_name}" ]; then
    source_name="${first_battery_name}"
  fi

  printf '{"external_power_online":%s,"battery_percent":%s,"battery_status":"%s","power_draw_w":%s,"ups_present":%s,"source_name":"%s","supplies":[%s]}' \
    "${external_online}" \
    "${battery_percent}" \
    "$(json_escape "${battery_status}")" \
    "${power_draw_w}" \
    "${ups_present}" \
    "$(json_escape "${source_name}")" \
    "${supplies_json}"
}
