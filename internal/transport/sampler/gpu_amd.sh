amd_sysfs_paths=()
amd_sysfs_bdfs=()
amd_sysfs_names=()
amd_sysfs_uuids=()

amd_round_float_to_int() {
  local value
  value="$(normalize_float "${1:-}")"
  if [ "${value}" = "-1" ]; then
    printf '%s' '-1'
    return
  fi

  awk -v value="${value}" 'BEGIN { printf "%d", value + 0.5 }'
}

amd_percent_from_mib() {
  local used total
  used="$(normalize_int "${1:-}")"
  total="$(normalize_int "${2:-}")"
  if [ "${used}" -lt 0 ] || [ "${total}" -le 0 ]; then
    printf '%s' '-1'
    return
  fi

  awk -v used="${used}" -v total="${total}" 'BEGIN { printf "%d", ((used * 100) / total) + 0.5 }'
}

amd_bytes_to_mib() {
  local value
  value="$(normalize_int "${1:-}")"
  if [ "${value}" -lt 0 ]; then
    printf '%s' '-1'
    return
  fi

  awk -v value="${value}" 'BEGIN { printf "%d", value / 1048576 }'
}

amd_microwatts_to_watts() {
  local value
  value="$(normalize_int "${1:-}")"
  if [ "${value}" -lt 0 ]; then
    printf '%s' '-1'
    return
  fi

  awk -v value="${value}" 'BEGIN { printf "%.2f", value / 1000000 }'
}

amd_millicelsius_to_celsius() {
  local value
  value="$(normalize_int "${1:-}")"
  if [ "${value}" -lt 0 ]; then
    printf '%s' '-1'
    return
  fi

  awk -v value="${value}" 'BEGIN { printf "%d", (value / 1000) + 0.5 }'
}

amd_number_from_text() {
  local value
  value="$(trim "${1:-}")"
  if [[ "${value}" =~ (-?[0-9]+([.][0-9]+)?) ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
    return
  fi

  printf '%s' '-1'
}

amd_read_first_existing_file() {
  local path line
  for path in "$@"; do
    if [ -r "${path}" ]; then
      IFS= read -r line <"${path}" || line=''
      trim "${line}"
      return
    fi
  done

  printf ''
}

amd_read_sysfs_uevent_field() {
  local path key line value
  path="$1"
  key="$2"
  if [ ! -r "${path}" ]; then
    printf ''
    return
  fi

  while IFS='=' read -r line value; do
    if [ "${line}" = "${key}" ]; then
      trim "${value}"
      return
    fi
  done <"${path}"

  printf ''
}

amd_normalize_pci_id() {
  local value
  value="$(trim "${1:-}")"
  value="${value#0x}"
  value="${value#0X}"
  printf '%s' "${value}" | tr '[:lower:]' '[:upper:]'
}

amd_regex_escape() {
  local value
  value="${1:-}"
  value="${value//\\/\\\\}"
  value="${value//./\\.}"
  value="${value//\[/\\[}"
  value="${value//\]/\\]}"
  value="${value//\(/\\(}"
  value="${value//\)/\\)}"
  value="${value//+/\\+}"
  value="${value//\*/\\*}"
  value="${value//\?/\\?}"
  value="${value//^/\\^}"
  value="${value//\$/\\$}"
  value="${value//|/\\|}"
  printf '%s' "${value}"
}

amd_json_one_line() {
  printf '%s' "${1:-}" | tr '\n\r\t' '   '
}

amd_json_string_for_key() {
  local json key escaped pattern
  json="$(amd_json_one_line "${1:-}")"
  key="$2"
  escaped="$(amd_regex_escape "${key}")"
  pattern="\"${escaped}\"[[:space:]]*:[[:space:]]*\"([^\"]*)\""
  if [[ "${json}" =~ ${pattern} ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
    return
  fi

  printf ''
}

amd_json_number_for_key() {
  local json key escaped pattern value
  json="$(amd_json_one_line "${1:-}")"
  key="$2"
  escaped="$(amd_regex_escape "${key}")"
  pattern="\"${escaped}\"[[:space:]]*:[[:space:]]*\"?(-?[0-9]+([.][0-9]+)?)"
  if [[ "${json}" =~ ${pattern} ]]; then
    value="${BASH_REMATCH[1]}"
    normalize_float "${value}"
    return
  fi

  value="$(amd_json_string_for_key "${json}" "${key}")"
  amd_number_from_text "${value}"
}

amd_first_json_string() {
  local json key value
  json="${1:-}"
  shift
  for key in "$@"; do
    value="$(amd_json_string_for_key "${json}" "${key}")"
    if [ -n "${value}" ]; then
      printf '%s' "${value}"
      return
    fi
  done

  printf ''
}

amd_first_json_number() {
  local json key value
  json="${1:-}"
  shift
  for key in "$@"; do
    value="$(amd_json_number_for_key "${json}" "${key}")"
    if [ "${value}" != "-1" ]; then
      printf '%s' "${value}"
      return
    fi
  done

  printf '%s' '-1'
}

amd_current_clock_from_levels() {
  local value
  value="$(trim "${1:-}")"
  if [[ "${value}" =~ ([0-9]+)[[:space:]]*[mM][hH][zZ][^0-9]*[*] ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
    return
  fi
  if [[ "${value}" =~ ([0-9]+)[[:space:]]*[mM][hH][zZ] ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
    return
  fi

  printf '%s' '-1'
}

amd_json_array_objects_for_key() {
  local json key
  json="$(amd_json_one_line "${1:-}")"
  key="$2"
  awk -v key="\"${key}\"" '
    {
      start = index($0, key)
      if (start == 0) {
        next
      }
      in_string = 0
      escaped = 0
      array_started = 0
      depth = 0
      object = ""
      for (i = start + length(key); i <= length($0); i++) {
        c = substr($0, i, 1)
        if (!array_started) {
          if (c == "[") {
            array_started = 1
          }
          continue
        }
        if (in_string) {
          if (depth > 0) {
            object = object c
          }
          if (escaped) {
            escaped = 0
          } else if (c == "\\") {
            escaped = 1
          } else if (c == "\"") {
            in_string = 0
          }
          continue
        }
        if (c == "\"") {
          if (depth > 0) {
            object = object c
          }
          in_string = 1
          continue
        }
        if (c == "{") {
          depth++
          object = object c
          continue
        }
        if (depth > 0) {
          object = object c
        }
        if (c == "}") {
          depth--
          if (depth == 0) {
            print object
            object = ""
          }
          continue
        }
        if (c == "]" && depth == 0) {
          break
        }
      }
    }
  ' <<<"${json}"
}

discover_amd_drm_devices() {
  local card_path card_name vendor device_id pci_id pci_bdf display_id
  amd_sysfs_paths=()
  amd_sysfs_bdfs=()
  amd_sysfs_names=()
  amd_sysfs_uuids=()

  if [ -z "${amd_drm_class_path:-}" ]; then
    return
  fi

  for card_path in "${amd_drm_class_path}"/card*; do
    if [ ! -e "${card_path}" ]; then
      continue
    fi
    card_name="${card_path##*/}"
    case "${card_name}" in
      card*[!0-9]*)
        continue
        ;;
      card[0-9]*)
        ;;
      *)
        continue
        ;;
    esac

    vendor="$(amd_read_first_existing_file "${card_path}/device/vendor")"
    vendor="$(printf '%s' "${vendor}" | tr '[:upper:]' '[:lower:]')"
    if [ "${vendor}" != "0x1002" ]; then
      continue
    fi

    device_id="$(amd_normalize_pci_id "$(amd_read_first_existing_file "${card_path}/device/device")")"
    pci_id="$(amd_read_sysfs_uevent_field "${card_path}/device/uevent" "PCI_ID")"
    pci_bdf="$(amd_read_sysfs_uevent_field "${card_path}/device/uevent" "PCI_SLOT_NAME")"
    if [ -z "${pci_id}" ] && [ -n "${device_id}" ]; then
      pci_id="1002:${device_id}"
    fi
    display_id="${pci_id:-1002:${device_id:-unknown}}"

    amd_sysfs_paths+=("${card_path}")
    amd_sysfs_bdfs+=("${pci_bdf}")
    amd_sysfs_names+=("AMD GPU ${display_id}")
    if [ -n "${pci_bdf}" ]; then
      amd_sysfs_uuids+=("amd-${pci_bdf}")
    else
      amd_sysfs_uuids+=("amd-${card_name}")
    fi
  done
}

amd_sysfs_mem_total_mib() {
  local idx value
  idx="$1"
  value="$(amd_read_first_existing_file \
    "${amd_sysfs_paths[idx]}/device/mem_info_vram_total" \
    "${amd_sysfs_paths[idx]}/device/mem_info_vis_vram_total")"
  amd_bytes_to_mib "${value}"
}

amd_sysfs_mem_used_mib() {
  local idx value
  idx="$1"
  value="$(amd_read_first_existing_file \
    "${amd_sysfs_paths[idx]}/device/mem_info_vram_used" \
    "${amd_sysfs_paths[idx]}/device/mem_info_vis_vram_used")"
  amd_bytes_to_mib "${value}"
}

amd_sysfs_temp_c() {
  local idx temp_path value best='-1' temp
  idx="$1"
  for temp_path in "${amd_sysfs_paths[idx]}"/device/hwmon/hwmon*/temp*_input; do
    if [ ! -r "${temp_path}" ]; then
      continue
    fi
    value="$(amd_read_first_existing_file "${temp_path}")"
    temp="$(amd_millicelsius_to_celsius "${value}")"
    if [ "${temp}" -gt "${best}" ]; then
      best="${temp}"
    fi
  done

  printf '%s' "${best}"
}

amd_sysfs_power_draw_w() {
  local idx value
  idx="$1"
  value="$(amd_read_first_existing_file \
    "${amd_sysfs_paths[idx]}/device/hwmon/hwmon0/power1_average" \
    "${amd_sysfs_paths[idx]}/device/hwmon/hwmon0/power1_input" \
    "${amd_sysfs_paths[idx]}"/device/hwmon/hwmon*/power1_average \
    "${amd_sysfs_paths[idx]}"/device/hwmon/hwmon*/power1_input)"
  amd_microwatts_to_watts "${value}"
}

amd_sysfs_power_limit_w() {
  local idx value
  idx="$1"
  value="$(amd_read_first_existing_file \
    "${amd_sysfs_paths[idx]}/device/hwmon/hwmon0/power1_cap" \
    "${amd_sysfs_paths[idx]}"/device/hwmon/hwmon*/power1_cap)"
  amd_microwatts_to_watts "${value}"
}

amd_sysfs_fan_percent() {
  local idx pwm max
  idx="$1"
  pwm="$(amd_read_first_existing_file \
    "${amd_sysfs_paths[idx]}/device/hwmon/hwmon0/pwm1" \
    "${amd_sysfs_paths[idx]}"/device/hwmon/hwmon*/pwm1)"
  max="$(amd_read_first_existing_file \
    "${amd_sysfs_paths[idx]}/device/hwmon/hwmon0/pwm1_max" \
    "${amd_sysfs_paths[idx]}"/device/hwmon/hwmon*/pwm1_max)"
  pwm="$(normalize_int "${pwm}")"
  max="$(normalize_int "${max}")"
  if [ "${pwm}" -lt 0 ] || [ "${max}" -le 0 ]; then
    printf '%s' '-1'
    return
  fi

  awk -v pwm="${pwm}" -v max="${max}" 'BEGIN { printf "%d", ((pwm * 100) / max) + 0.5 }'
}

amd_sysfs_clock_from_levels() {
  local idx path
  idx="$1"
  path="$2"
  amd_current_clock_from_levels "$(amd_read_first_existing_file "${amd_sysfs_paths[idx]}/device/${path}")"
}

amd_sysfs_pstate() {
  local idx state
  idx="$1"
  state="$(amd_read_first_existing_file "${amd_sysfs_paths[idx]}/device/power_dpm_force_performance_level")"
  case "${state}" in
    auto | low | high | manual | profile_* | performance | balanced | powersave)
      printf '%s' "${state}"
      ;;
    *)
      printf ''
      ;;
  esac
}

read_amd_smi_metric_json() {
  local output=''
  if [ -z "${amd_smi_path:-}" ]; then
    printf ''
    return
  fi

  if command -v timeout >/dev/null 2>&1; then
    output="$(timeout 2s "${amd_smi_path}" metric --json 2>/dev/null || true)"
  else
    output="$("${amd_smi_path}" metric --json 2>/dev/null || true)"
  fi

  printf '%s' "${output}"
}

read_rocm_smi_json() {
  local output=''
  if [ -z "${rocm_smi_path:-}" ]; then
    printf ''
    return
  fi

  if command -v timeout >/dev/null 2>&1; then
    output="$(timeout 2s "${rocm_smi_path}" --showproductname --showuniqueid --showuse --showmemuse --showmeminfo vram --showtemp --showpower --showfan --showclocks --showperflevel --json 2>/dev/null || true)"
  else
    output="$("${rocm_smi_path}" --showproductname --showuniqueid --showuse --showmemuse --showmeminfo vram --showtemp --showpower --showfan --showclocks --showperflevel --json 2>/dev/null || true)"
  fi

  printf '%s' "${output}"
}

emit_amd_gpu_json_object() {
  local comma idx uuid name util mem_util encoder_util decoder_util mem_used mem_total temp power_draw power_limit fan sm_clock sm_clock_max mem_clock mem_clock_max graphics_clock video_clock pcie_gen_cur pcie_gen_cap pcie_width_cur pcie_width_cap throttle_reason pstate
  comma="$1"
  idx="$2"
  uuid="$3"
  name="$4"
  util="$5"
  mem_util="$6"
  encoder_util="$7"
  decoder_util="$8"
  mem_used="$9"
  mem_total="${10}"
  temp="${11}"
  power_draw="${12}"
  power_limit="${13}"
  fan="${14}"
  sm_clock="${15}"
  sm_clock_max="${16}"
  mem_clock="${17}"
  mem_clock_max="${18}"
  graphics_clock="${19}"
  video_clock="${20}"
  pcie_gen_cur="${21}"
  pcie_gen_cap="${22}"
  pcie_width_cur="${23}"
  pcie_width_cap="${24}"
  throttle_reason="${25}"
  pstate="${26}"

  printf '%s{"index":%s,"uuid":"%s","name":"%s","util_percent":%s,"mem_util_percent":%s,"encoder_util_percent":%s,"decoder_util_percent":%s,"mem_used_mib":%s,"mem_total_mib":%s,"temp_c":%s,"power_draw_w":%s,"power_limit_w":%s,"fan_percent":%s,"sm_clock_mhz":%s,"sm_clock_max_mhz":%s,"mem_clock_mhz":%s,"mem_clock_max_mhz":%s,"graphics_clock_mhz":%s,"video_clock_mhz":%s,"pcie_gen_current":%s,"pcie_gen_max":%s,"pcie_width_current":%s,"pcie_width_max":%s,"throttle_reasons":"%s","p_state":"%s"}' \
    "${comma}" \
    "${idx}" \
    "$(json_escape "${uuid}")" \
    "$(json_escape "${name}")" \
    "${util}" \
    "${mem_util}" \
    "${encoder_util}" \
    "${decoder_util}" \
    "${mem_used}" \
    "${mem_total}" \
    "${temp}" \
    "${power_draw}" \
    "${power_limit}" \
    "${fan}" \
    "${sm_clock}" \
    "${sm_clock_max}" \
    "${mem_clock}" \
    "${mem_clock_max}" \
    "${graphics_clock}" \
    "${video_clock}" \
    "${pcie_gen_cur}" \
    "${pcie_gen_cap}" \
    "${pcie_width_cur}" \
    "${pcie_width_cap}" \
    "$(json_escape "${throttle_reason}")" \
    "$(json_escape "${pstate}")"
}

build_amd_smi_gpu_json() {
  local metric_json gpu_objects gpu_json idx uuid name bdf util mem_used mem_total mem_util decoder_util temp power_draw power_limit fan sm_clock sm_clock_max mem_clock mem_clock_max pcie_gen_cur pcie_gen_cap pcie_width_cur pcie_width_cap throttle_reason pstate emitted='0' comma=''
  metric_json="$(read_amd_smi_metric_json)"
  if [ -z "${metric_json}" ]; then
    printf '[]'
    return
  fi

  gpu_objects="$(amd_json_array_objects_for_key "${metric_json}" "gpu")"
  if [ -z "${gpu_objects}" ]; then
    gpu_objects="${metric_json}"
  fi

  printf '['
  while IFS= read -r gpu_json || [ -n "${gpu_json}" ]; do
    gpu_json="$(trim "${gpu_json}")"
    if [ -z "${gpu_json}" ]; then
      continue
    fi

    idx="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "gpu_id" "gpu" "index")")"
    if [ "${idx}" -lt 0 ]; then
      idx="${emitted}"
    fi
    name="$(amd_first_json_string "${gpu_json}" "market_name" "name" "gpu_name")"
    uuid="$(amd_first_json_string "${gpu_json}" "uuid" "unique_id")"
    bdf="$(amd_first_json_string "${gpu_json}" "bdf" "pci_bdf" "pci_bus")"
    util="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "gfx_activity" "gfx_busy" "gpu_util" "util_percent")")"
    decoder_util="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "mm_activity" "decoder_util" "decoder_util_percent")")"
    mem_used="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "vram_used_mb" "vram_used_mib" "mem_used_mib")")"
    mem_total="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "vram_total_mb" "vram_total_mib" "mem_total_mib")")"
    mem_util="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "vram_usage" "mem_util_percent")")"
    if [ "${mem_util}" -lt 0 ]; then
      mem_util="$(amd_percent_from_mib "${mem_used}" "${mem_total}")"
    fi
    temp="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "edge_celsius" "hotspot_celsius" "temperature" "temp_c")")"
    power_draw="$(amd_first_json_number "${gpu_json}" "average_socket_power_w" "power_draw_w" "current_socket_power_w")"
    power_limit="$(amd_first_json_number "${gpu_json}" "power_cap_w" "power_limit_w" "socket_power_cap_w")"
    fan="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "speed_percent" "fan_percent")")"
    sm_clock="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "gfxclk_mhz" "sclk_mhz" "sm_clock_mhz")")"
    sm_clock_max="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "gfxclk_max_mhz" "sclk_max_mhz" "sm_clock_max_mhz")")"
    mem_clock="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "memclk_mhz" "mclk_mhz" "mem_clock_mhz")")"
    mem_clock_max="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "memclk_max_mhz" "mclk_max_mhz" "mem_clock_max_mhz")")"
    pcie_gen_cur="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "current_gen" "pcie_gen_current")")"
    pcie_gen_cap="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "max_gen" "pcie_gen_max")")"
    pcie_width_cur="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "current_width" "pcie_width_current")")"
    pcie_width_cap="$(amd_round_float_to_int "$(amd_first_json_number "${gpu_json}" "max_width" "pcie_width_max")")"
    throttle_reason="$(amd_first_json_string "${gpu_json}" "throttle_status" "throttle_reasons")"
    pstate="$(amd_first_json_string "${gpu_json}" "level" "perf_level" "performance_level" "p_state")"

    if [ -z "${name}" ] && [ -z "${uuid}" ] && [ -z "${bdf}" ] &&
      [ "${util}" -lt 0 ] && [ "${decoder_util}" -lt 0 ] &&
      [ "${mem_used}" -lt 0 ] && [ "${mem_total}" -lt 0 ] && [ "${mem_util}" -lt 0 ] &&
      [ "${temp}" -lt 0 ] && [ "${power_draw}" = "-1" ] && [ "${power_limit}" = "-1" ] && [ "${fan}" -lt 0 ] &&
      [ "${sm_clock}" -lt 0 ] && [ "${sm_clock_max}" -lt 0 ] && [ "${mem_clock}" -lt 0 ] && [ "${mem_clock_max}" -lt 0 ] &&
      [ "${pcie_gen_cur}" -lt 0 ] && [ "${pcie_gen_cap}" -lt 0 ] && [ "${pcie_width_cur}" -lt 0 ] && [ "${pcie_width_cap}" -lt 0 ] &&
      [ -z "${throttle_reason}" ] && [ -z "${pstate}" ]; then
      continue
    fi

    if [ -z "${name}" ]; then
      name='AMD GPU'
    fi
    if [ -z "${uuid}" ]; then
      if [ -n "${bdf}" ]; then
        uuid="amd-${bdf}"
      else
        uuid="amd-gpu-${idx}"
      fi
    fi

    emit_amd_gpu_json_object "${comma}" "${idx}" "${uuid}" "${name}" "${util}" "${mem_util}" '-1' "${decoder_util}" "${mem_used}" "${mem_total}" "${temp}" "${power_draw}" "${power_limit}" "${fan}" "${sm_clock}" "${sm_clock_max}" "${mem_clock}" "${mem_clock_max}" "${sm_clock}" '-1' "${pcie_gen_cur}" "${pcie_gen_cap}" "${pcie_width_cur}" "${pcie_width_cap}" "${throttle_reason}" "${pstate}"
    comma=','
    emitted=$((emitted + 1))
  done <<<"${gpu_objects}"
  printf ']'
}

build_rocm_smi_gpu_json() {
  local rocm_json idx uuid name util mem_util mem_used_bytes mem_total_bytes mem_used mem_total temp power_draw power_limit fan sm_clock_levels mem_clock_levels sm_clock mem_clock pstate
  rocm_json="$(read_rocm_smi_json)"
  if [ -z "${rocm_json}" ]; then
    printf '[]'
    return
  fi

  idx='0'
  name="$(amd_first_json_string "${rocm_json}" "Card series" "Product Name" "Device Name")"
  uuid="$(amd_first_json_string "${rocm_json}" "Unique ID" "GPU ID" "Serial Number")"
  util="$(amd_round_float_to_int "$(amd_first_json_number "${rocm_json}" "GPU use (%)" "GPU use")")"
  mem_util="$(amd_round_float_to_int "$(amd_first_json_number "${rocm_json}" "GPU Memory Allocated (VRAM%)" "GPU Memory Allocated")")"
  mem_used_bytes="$(amd_first_json_number "${rocm_json}" "VRAM Total Used Memory (B)" "VRAM Used Memory (B)")"
  mem_total_bytes="$(amd_first_json_number "${rocm_json}" "VRAM Total Memory (B)" "VRAM Memory Total (B)")"
  mem_used="$(amd_bytes_to_mib "${mem_used_bytes}")"
  mem_total="$(amd_bytes_to_mib "${mem_total_bytes}")"
  if [ "${mem_util}" -lt 0 ]; then
    mem_util="$(amd_percent_from_mib "${mem_used}" "${mem_total}")"
  fi
  temp="$(amd_round_float_to_int "$(amd_first_json_number "${rocm_json}" "Temperature (Sensor edge) (C)" "Temperature (Sensor junction) (C)" "Temperature (Sensor memory) (C)")")"
  power_draw="$(amd_first_json_number "${rocm_json}" "Average Graphics Package Power (W)" "Current Socket Graphics Package Power (W)")"
  power_limit="$(amd_first_json_number "${rocm_json}" "Max Graphics Package Power (W)" "Power Cap (W)")"
  fan="$(amd_round_float_to_int "$(amd_first_json_number "${rocm_json}" "Fan Level" "Fan Speed (%)")")"
  sm_clock_levels="$(amd_first_json_string "${rocm_json}" "sclk clock level" "SCLK Clock Level")"
  mem_clock_levels="$(amd_first_json_string "${rocm_json}" "mclk clock level" "MCLK Clock Level")"
  sm_clock="$(amd_current_clock_from_levels "${sm_clock_levels}")"
  mem_clock="$(amd_current_clock_from_levels "${mem_clock_levels}")"
  pstate="$(amd_first_json_string "${rocm_json}" "Performance Level" "Perf Level")"

  if [ -z "${name}" ] && [ -z "${uuid}" ] &&
    [ "${util}" -lt 0 ] && [ "${mem_util}" -lt 0 ] &&
    [ "${mem_used}" -lt 0 ] && [ "${mem_total}" -lt 0 ] &&
    [ "${temp}" -lt 0 ] && [ "${power_draw}" = "-1" ] && [ "${power_limit}" = "-1" ] && [ "${fan}" -lt 0 ] &&
    [ "${sm_clock}" -lt 0 ] && [ "${mem_clock}" -lt 0 ] &&
    [ -z "${pstate}" ]; then
    printf '[]'
    return
  fi

  if [ -z "${name}" ]; then
    name='AMD GPU'
  fi
  if [ -z "${uuid}" ]; then
    uuid='amd-rocm-smi-0'
  fi

  printf '['
  emit_amd_gpu_json_object '' "${idx}" "${uuid}" "${name}" "${util}" "${mem_util}" '-1' '-1' "${mem_used}" "${mem_total}" "${temp}" "${power_draw}" "${power_limit}" "${fan}" "${sm_clock}" '-1' "${mem_clock}" '-1' "${sm_clock}" '-1' '-1' '-1' '-1' '-1' '' "${pstate}"
  printf ']'
}

build_amd_sysfs_gpu_json() {
  local sysfs_idx idx uuid name mem_used mem_total mem_util temp power_draw power_limit fan sm_clock sm_clock_max mem_clock mem_clock_max pstate comma=''

  if [ "${#amd_sysfs_paths[@]}" -eq 0 ]; then
    printf '[]'
    return
  fi

  printf '['
  for ((sysfs_idx = 0; sysfs_idx < ${#amd_sysfs_paths[@]}; sysfs_idx++)); do
    idx="${sysfs_idx}"
    uuid="${amd_sysfs_uuids[sysfs_idx]}"
    name="${amd_sysfs_names[sysfs_idx]}"
    mem_used="$(amd_sysfs_mem_used_mib "${sysfs_idx}")"
    mem_total="$(amd_sysfs_mem_total_mib "${sysfs_idx}")"
    mem_util="$(amd_percent_from_mib "${mem_used}" "${mem_total}")"
    temp="$(amd_sysfs_temp_c "${sysfs_idx}")"
    power_draw="$(amd_sysfs_power_draw_w "${sysfs_idx}")"
    power_limit="$(amd_sysfs_power_limit_w "${sysfs_idx}")"
    fan="$(amd_sysfs_fan_percent "${sysfs_idx}")"
    sm_clock="$(amd_sysfs_clock_from_levels "${sysfs_idx}" "pp_dpm_sclk")"
    sm_clock_max='-1'
    mem_clock="$(amd_sysfs_clock_from_levels "${sysfs_idx}" "pp_dpm_mclk")"
    mem_clock_max='-1'
    pstate="$(amd_sysfs_pstate "${sysfs_idx}")"

    emit_amd_gpu_json_object "${comma}" "${idx}" "${uuid}" "${name}" '-1' "${mem_util}" '-1' '-1' "${mem_used}" "${mem_total}" "${temp}" "${power_draw}" "${power_limit}" "${fan}" "${sm_clock}" "${sm_clock_max}" "${mem_clock}" "${mem_clock_max}" "${sm_clock}" '-1' '-1' '-1' '-1' '-1' '' "${pstate}"
    comma=','
  done
  printf ']'
}

build_amd_gpu_json() {
  local amd_smi_json rocm_smi_json
  discover_amd_drm_devices

  amd_smi_json="$(build_amd_smi_gpu_json)"
  if [ "$(json_array_count "${amd_smi_json}")" -gt 0 ]; then
    printf '%s' "${amd_smi_json}"
    return
  fi

  rocm_smi_json="$(build_rocm_smi_gpu_json)"
  if [ "$(json_array_count "${rocm_smi_json}")" -gt 0 ]; then
    printf '%s' "${rocm_smi_json}"
    return
  fi

  build_amd_sysfs_gpu_json
}
