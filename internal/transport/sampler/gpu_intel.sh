intel_sysfs_paths=()
intel_sysfs_bdfs=()
intel_sysfs_names=()
intel_sysfs_uuids=()
xpu_smi_bdfs=()

round_float_to_int() {
  local value
  value="$(normalize_float "${1:-}")"
  if [ "${value}" = "-1" ]; then
    printf '%s' '-1'
    return
  fi

  awk -v value="${value}" 'BEGIN { printf "%d", value + 0.5 }'
}

percent_from_mib() {
  local used total
  used="$(normalize_int "${1:-}")"
  total="$(normalize_int "${2:-}")"
  if [ "${used}" -lt 0 ] || [ "${total}" -le 0 ]; then
    printf '%s' '-1'
    return
  fi

  awk -v used="${used}" -v total="${total}" 'BEGIN { printf "%d", ((used * 100) / total) + 0.5 }'
}

bytes_to_mib() {
  local value
  value="$(normalize_int "${1:-}")"
  if [ "${value}" -lt 0 ]; then
    printf '%s' '-1'
    return
  fi

  awk -v value="${value}" 'BEGIN { printf "%d", value / 1048576 }'
}

microwatts_to_watts() {
  local value
  value="$(normalize_int "${1:-}")"
  if [ "${value}" -lt 0 ]; then
    printf '%s' '-1'
    return
  fi

  awk -v value="${value}" 'BEGIN { printf "%.2f", value / 1000000 }'
}

millicelsius_to_celsius() {
  local value
  value="$(normalize_int "${1:-}")"
  if [ "${value}" -lt 0 ]; then
    printf '%s' '-1'
    return
  fi

  awk -v value="${value}" 'BEGIN { printf "%d", (value / 1000) + 0.5 }'
}

max_int_value() {
  local best='-1'
  local value
  for value in "$@"; do
    value="$(normalize_int "${value}")"
    if [ "${value}" -gt "${best}" ]; then
      best="${value}"
    fi
  done

  printf '%s' "${best}"
}

read_first_existing_file() {
  local path line
  for path in "$@"; do
    if [ -r "${path}" ]; then
      IFS= read -r line < "${path}" || line=''
      trim "${line}"
      return
    fi
  done

  printf ''
}

read_sysfs_uevent_field() {
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
  done < "${path}"

  printf ''
}

normalize_pci_id() {
  local value
  value="$(trim "${1:-}")"
  value="${value#0x}"
  value="${value#0X}"
  printf '%s' "${value}" | tr '[:lower:]' '[:upper:]'
}

discover_intel_drm_devices() {
  local card_path card_name vendor device_id pci_id pci_bdf display_id
  intel_sysfs_paths=()
  intel_sysfs_bdfs=()
  intel_sysfs_names=()
  intel_sysfs_uuids=()

  if [ -z "${intel_drm_class_path:-}" ]; then
    return
  fi

  for card_path in "${intel_drm_class_path}"/card*; do
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

    vendor="$(read_first_existing_file "${card_path}/device/vendor")"
    vendor="$(printf '%s' "${vendor}" | tr '[:upper:]' '[:lower:]')"
    if [ "${vendor}" != "0x8086" ]; then
      continue
    fi

    device_id="$(normalize_pci_id "$(read_first_existing_file "${card_path}/device/device")")"
    pci_id="$(read_sysfs_uevent_field "${card_path}/device/uevent" "PCI_ID")"
    pci_bdf="$(read_sysfs_uevent_field "${card_path}/device/uevent" "PCI_SLOT_NAME")"
    if [ -z "${pci_id}" ] && [ -n "${device_id}" ]; then
      pci_id="8086:${device_id}"
    fi
    display_id="${pci_id:-8086:${device_id:-unknown}}"

    intel_sysfs_paths+=("${card_path}")
    intel_sysfs_bdfs+=("${pci_bdf}")
    intel_sysfs_names+=("Intel GPU ${display_id}")
    if [ -n "${pci_bdf}" ]; then
      intel_sysfs_uuids+=("intel-${pci_bdf}")
    else
      intel_sysfs_uuids+=("intel-${card_name}")
    fi
  done
}

intel_sysfs_index_for_bdf() {
  local bdf i
  bdf="$(trim "${1:-}")"
  if [ -z "${bdf}" ]; then
    printf '%s' '-1'
    return
  fi
  for ((i = 0; i < ${#intel_sysfs_bdfs[@]}; i++)); do
    if [ "${intel_sysfs_bdfs[i]}" = "${bdf}" ]; then
      printf '%s' "${i}"
      return
    fi
  done

  printf '%s' '-1'
}

xpu_smi_has_bdf() {
  local bdf xpu_bdf
  bdf="$(trim "${1:-}")"
  if [ -z "${bdf}" ]; then
    return 1
  fi

  for xpu_bdf in "${xpu_smi_bdfs[@]}"; do
    if [ "${xpu_bdf}" = "${bdf}" ]; then
      return 0
    fi
  done

  return 1
}

intel_sysfs_has_unmatched_devices() {
  local sysfs_idx
  for ((sysfs_idx = 0; sysfs_idx < ${#intel_sysfs_paths[@]}; sysfs_idx++)); do
    if ! xpu_smi_has_bdf "${intel_sysfs_bdfs[sysfs_idx]}"; then
      return 0
    fi
  done

  return 1
}

intel_sysfs_mem_total_mib() {
  local idx value
  idx="$1"
  value="$(read_first_existing_file \
    "${intel_sysfs_paths[idx]}/device/mem_info_vram_total" \
    "${intel_sysfs_paths[idx]}/device/mem_info_vis_vram_total")"
  bytes_to_mib "${value}"
}

intel_sysfs_mem_used_mib() {
  local idx value
  idx="$1"
  value="$(read_first_existing_file \
    "${intel_sysfs_paths[idx]}/device/mem_info_vram_used" \
    "${intel_sysfs_paths[idx]}/device/mem_info_vis_vram_used")"
  bytes_to_mib "${value}"
}

intel_sysfs_temp_c() {
  local idx temp_path value best='-1' temp
  idx="$1"
  for temp_path in "${intel_sysfs_paths[idx]}"/device/hwmon/hwmon*/temp*_input; do
    if [ ! -r "${temp_path}" ]; then
      continue
    fi
    value="$(read_first_existing_file "${temp_path}")"
    temp="$(millicelsius_to_celsius "${value}")"
    if [ "${temp}" -gt "${best}" ]; then
      best="${temp}"
    fi
  done

  printf '%s' "${best}"
}

intel_sysfs_power_draw_w() {
  local idx value
  idx="$1"
  value="$(read_first_existing_file \
    "${intel_sysfs_paths[idx]}/device/hwmon/hwmon0/power1_average" \
    "${intel_sysfs_paths[idx]}/device/hwmon/hwmon0/power1_input" \
    "${intel_sysfs_paths[idx]}"/device/hwmon/hwmon*/power1_average \
    "${intel_sysfs_paths[idx]}"/device/hwmon/hwmon*/power1_input)"
  microwatts_to_watts "${value}"
}

intel_sysfs_power_limit_w() {
  local idx value
  idx="$1"
  value="$(read_first_existing_file \
    "${intel_sysfs_paths[idx]}/device/hwmon/hwmon0/power1_cap" \
    "${intel_sysfs_paths[idx]}"/device/hwmon/hwmon*/power1_cap)"
  microwatts_to_watts "${value}"
}

intel_sysfs_graphics_clock_mhz() {
  local idx value
  idx="$1"
  value="$(read_first_existing_file \
    "${intel_sysfs_paths[idx]}/device/gt_cur_freq_mhz" \
    "${intel_sysfs_paths[idx]}/device/gt/gt0/rps_cur_freq_mhz")"
  normalize_int "${value}"
}

intel_sysfs_max_graphics_clock_mhz() {
  local idx value
  idx="$1"
  value="$(read_first_existing_file \
    "${intel_sysfs_paths[idx]}/device/gt_RP0_freq_mhz" \
    "${intel_sysfs_paths[idx]}/device/gt_max_freq_mhz" \
    "${intel_sysfs_paths[idx]}/device/gt/gt0/rps_RP0_freq_mhz")"
  normalize_int "${value}"
}

json_one_line() {
  printf '%s' "${1:-}" | tr '\n\r\t' '   '
}

json_number_after_regex() {
  local json pattern
  json="$(json_one_line "${1:-}")"
  pattern="$2"
  if [[ "${json}" =~ ${pattern} ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
    return
  fi

  printf '%s' '-1'
}

intel_gpu_top_object_number() {
  local json object key pattern
  json="$1"
  object="$2"
  key="$3"
  pattern="\"${object}\"[[:space:]]*:[[:space:]]*\\{[^}]*\"${key}\"[[:space:]]*:[[:space:]]*\"?(-?[0-9]+([.][0-9]+)?)"
  json_number_after_regex "${json}" "${pattern}"
}

intel_gpu_top_engine_busy() {
  local json engine pattern
  json="$1"
  engine="$2"
  pattern="\"${engine}\"[[:space:]]*:[[:space:]]*\\{[^}]*\"busy\"[[:space:]]*:[[:space:]]*\"?(-?[0-9]+([.][0-9]+)?)"
  round_float_to_int "$(json_number_after_regex "${json}" "${pattern}")"
}

read_intel_gpu_top_sample() {
  local output=''
  if [ -z "${intel_gpu_top_path:-}" ]; then
    printf ''
    return
  fi

  if command -v timeout >/dev/null 2>&1; then
    output="$(timeout 2s "${intel_gpu_top_path}" -J -s 100 -n 1 -o - 2>/dev/null || true)"
  else
    output="$("${intel_gpu_top_path}" -J -s 100 -n 1 -o - 2>/dev/null || true)"
  fi

  printf '%s' "${output}"
}

csv_clean_field() {
  local value
  value="$(trim "${1:-}")"
  value="${value%\"}"
  value="${value#\"}"
  trim "${value}"
}

read_xpu_smi_discovery_dump() {
  if [ -z "${xpu_smi_path:-}" ]; then
    printf ''
    return
  fi

  "${xpu_smi_path}" discovery --dump 1,2,4,11,16 2>/dev/null || true
}

read_xpu_smi_stats_dump() {
  local device_id
  device_id="$1"
  if [ -z "${xpu_smi_path:-}" ]; then
    printf ''
    return
  fi

  "${xpu_smi_path}" dump -d "${device_id}" -m 0,1,2,3,18,22,23,24,25,35,36 -i 1 -n 1 2>/dev/null || true
}

record_xpu_smi_bdfs_from_discovery() {
  local discovery line device_id name uuid bdf mem_total_text
  discovery="${1:-}"
  xpu_smi_bdfs=()

  while IFS= read -r line || [ -n "${line}" ]; do
    case "${line}" in
      ''|Device\ ID,*)
        continue
        ;;
    esac

    IFS=',' read -r device_id name uuid bdf mem_total_text _ <<< "${line}"
    device_id="$(normalize_int "$(csv_clean_field "${device_id}")")"
    if [ "${device_id}" -lt 0 ]; then
      continue
    fi
    bdf="$(csv_clean_field "${bdf}")"
    if [ -n "${bdf}" ]; then
      xpu_smi_bdfs+=("${bdf}")
    fi
  done <<< "${discovery}"
}

build_xpu_smi_gpu_json() {
  local discovery line comma='' index
  local device_id name uuid bdf mem_total_text mem_total
  local stats stats_line timestamp stat_device_id util power_draw sm_clock temp mem_util mem_used compute_util render_util decoder_util encoder_util throttle video_clock
  local sysfs_idx sysfs_temp sysfs_power_limit

  discovery="${1:-}"
  index="$(normalize_int "${2:-0}")"
  if [ "${index}" -lt 0 ]; then
    index='0'
  fi
  if [ -z "${discovery}" ]; then
    printf '[]'
    return
  fi

  printf '['
  while IFS= read -r line || [ -n "${line}" ]; do
    case "${line}" in
      ''|Device\ ID,*)
        continue
        ;;
    esac

    IFS=',' read -r device_id name uuid bdf mem_total_text _ <<< "${line}"
    device_id="$(normalize_int "$(csv_clean_field "${device_id}")")"
    if [ "${device_id}" -lt 0 ]; then
      continue
    fi
    name="$(csv_clean_field "${name}")"
    uuid="$(csv_clean_field "${uuid}")"
    bdf="$(csv_clean_field "${bdf}")"
    mem_total_text="$(csv_clean_field "${mem_total_text}")"
    mem_total="$(round_float_to_int "${mem_total_text%% *}")"
    if [ "${mem_total}" -lt 0 ]; then
      mem_total='-1'
    fi

    util='-1'
    power_draw='-1'
    sm_clock='-1'
    temp='-1'
    mem_util='-1'
    mem_used='-1'
    compute_util='-1'
    render_util='-1'
    decoder_util='-1'
    encoder_util='-1'
    throttle=''
    video_clock='-1'

    stats="$(read_xpu_smi_stats_dump "${device_id}")"
    stats_line="$(printf '%s\n' "${stats}" | awk 'NF && $1 !~ /^Timestamp/ { line=$0 } END { print line }')"
    if [ -n "${stats_line}" ]; then
      IFS=',' read -r timestamp stat_device_id util power_draw sm_clock temp mem_used compute_util render_util decoder_util encoder_util throttle video_clock _ <<< "${stats_line}"
      util="$(round_float_to_int "${util}")"
      power_draw="$(normalize_float "${power_draw}")"
      sm_clock="$(normalize_int "${sm_clock}")"
      temp="$(normalize_int "${temp}")"
      mem_used="$(round_float_to_int "${mem_used}")"
      compute_util="$(round_float_to_int "${compute_util}")"
      render_util="$(round_float_to_int "${render_util}")"
      decoder_util="$(round_float_to_int "${decoder_util}")"
      encoder_util="$(round_float_to_int "${encoder_util}")"
      throttle="$(csv_clean_field "${throttle}")"
      video_clock="$(normalize_int "${video_clock}")"
    fi

    if [ "${mem_util}" -lt 0 ]; then
      mem_util="$(percent_from_mib "${mem_used}" "${mem_total}")"
    fi
    if [ "${util}" -lt 0 ]; then
      util="$(max_int_value "${compute_util}" "${render_util}" "${decoder_util}" "${encoder_util}")"
    fi
    if [ -z "${uuid}" ]; then
      uuid="intel-${bdf:-xpu-${device_id}}"
    fi
    if [ -z "${name}" ]; then
      name="Intel GPU"
    fi

    sysfs_idx="$(intel_sysfs_index_for_bdf "${bdf}")"
    sysfs_power_limit='-1'
    if [ "${sysfs_idx}" -ge 0 ]; then
      sysfs_temp="$(intel_sysfs_temp_c "${sysfs_idx}")"
      if [ "${temp}" -lt 0 ]; then
        temp="${sysfs_temp}"
      fi
      sysfs_power_limit="$(intel_sysfs_power_limit_w "${sysfs_idx}")"
    fi

    printf '%s{"index":%s,"uuid":"%s","name":"%s","util_percent":%s,"mem_util_percent":%s,"encoder_util_percent":%s,"decoder_util_percent":%s,"mem_used_mib":%s,"mem_total_mib":%s,"temp_c":%s,"power_draw_w":%s,"power_limit_w":%s,"fan_percent":-1,"sm_clock_mhz":%s,"sm_clock_max_mhz":-1,"mem_clock_mhz":-1,"mem_clock_max_mhz":-1,"graphics_clock_mhz":%s,"video_clock_mhz":%s,"pcie_gen_current":-1,"pcie_gen_max":-1,"pcie_width_current":-1,"pcie_width_max":-1,"throttle_reasons":"%s","p_state":""}' \
      "${comma}" \
      "${index}" \
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
      "${sysfs_power_limit}" \
      "${sm_clock}" \
      "${sm_clock}" \
      "${video_clock}" \
      "$(json_escape "${throttle}")"
    comma=','
    index=$((index + 1))
  done <<< "${discovery}"
  printf ']'
}

build_intel_gpu_top_or_sysfs_json() {
  local top_json top_util render_util compute_util video_util video_enhance_util
  local actual_clock requested_clock power_draw
  local idx sysfs_idx uuid name mem_used mem_total mem_util temp power_limit sysfs_clock sysfs_max_clock emit_clock emit_max_clock
  local base_index emitted comma=''

  base_index="$(normalize_int "${1:-0}")"
  if [ "${base_index}" -lt 0 ]; then
    base_index='0'
  fi
  emitted='0'

  if [ "${base_index}" -gt 0 ] && ! intel_sysfs_has_unmatched_devices; then
    printf '[]'
    return
  fi

  top_json="$(read_intel_gpu_top_sample)"
  top_util='-1'
  render_util='-1'
  compute_util='-1'
  video_util='-1'
  video_enhance_util='-1'
  actual_clock='-1'
  requested_clock='-1'
  power_draw='-1'

  if [ -n "${top_json}" ]; then
    render_util="$(intel_gpu_top_engine_busy "${top_json}" "Render/3D")"
    if [ "${render_util}" -lt 0 ]; then
      render_util="$(intel_gpu_top_engine_busy "${top_json}" "Render")"
    fi
    compute_util="$(intel_gpu_top_engine_busy "${top_json}" "Compute")"
    video_util="$(intel_gpu_top_engine_busy "${top_json}" "Video")"
    video_enhance_util="$(intel_gpu_top_engine_busy "${top_json}" "VideoEnhance")"
    top_util="$(max_int_value "${render_util}" "${compute_util}" "${video_util}" "${video_enhance_util}")"
    actual_clock="$(round_float_to_int "$(intel_gpu_top_object_number "${top_json}" "frequency" "actual")")"
    requested_clock="$(round_float_to_int "$(intel_gpu_top_object_number "${top_json}" "frequency" "requested")")"
    power_draw="$(normalize_float "$(intel_gpu_top_object_number "${top_json}" "power" "GPU")")"
  fi

  if [ "${#intel_sysfs_paths[@]}" -eq 0 ] && [ "${top_util}" -lt 0 ] && [ "${actual_clock}" -lt 0 ] && [ "${power_draw}" = "-1" ]; then
    printf '[]'
    return
  fi

  printf '['
  if [ "${#intel_sysfs_paths[@]}" -gt 0 ]; then
    for ((sysfs_idx = 0; sysfs_idx < ${#intel_sysfs_paths[@]}; sysfs_idx++)); do
      if xpu_smi_has_bdf "${intel_sysfs_bdfs[sysfs_idx]}"; then
        continue
      fi

      idx=$((base_index + emitted))
      uuid="${intel_sysfs_uuids[sysfs_idx]}"
      name="${intel_sysfs_names[sysfs_idx]}"
      mem_used="$(intel_sysfs_mem_used_mib "${sysfs_idx}")"
      mem_total="$(intel_sysfs_mem_total_mib "${sysfs_idx}")"
      mem_util="$(percent_from_mib "${mem_used}" "${mem_total}")"
      temp="$(intel_sysfs_temp_c "${sysfs_idx}")"
      power_limit="$(intel_sysfs_power_limit_w "${sysfs_idx}")"
      sysfs_clock="$(intel_sysfs_graphics_clock_mhz "${sysfs_idx}")"
      sysfs_max_clock="$(intel_sysfs_max_graphics_clock_mhz "${sysfs_idx}")"
      emit_clock="${actual_clock}"
      emit_max_clock="${requested_clock}"
      if [ "${emit_clock}" -lt 0 ]; then
        emit_clock="${sysfs_clock}"
      fi
      if [ "${emit_max_clock}" -lt 0 ]; then
        emit_max_clock="${sysfs_max_clock}"
      fi

      printf '%s{"index":%s,"uuid":"%s","name":"%s","util_percent":%s,"mem_util_percent":%s,"encoder_util_percent":-1,"decoder_util_percent":%s,"mem_used_mib":%s,"mem_total_mib":%s,"temp_c":%s,"power_draw_w":%s,"power_limit_w":%s,"fan_percent":-1,"sm_clock_mhz":%s,"sm_clock_max_mhz":%s,"mem_clock_mhz":-1,"mem_clock_max_mhz":-1,"graphics_clock_mhz":%s,"video_clock_mhz":-1,"pcie_gen_current":-1,"pcie_gen_max":-1,"pcie_width_current":-1,"pcie_width_max":-1,"throttle_reasons":"","p_state":""}' \
        "${comma}" \
        "${idx}" \
        "$(json_escape "${uuid}")" \
        "$(json_escape "${name}")" \
        "${top_util}" \
        "${mem_util}" \
        "${video_util}" \
        "${mem_used}" \
        "${mem_total}" \
        "${temp}" \
        "${power_draw}" \
        "${power_limit}" \
        "${emit_clock}" \
        "${emit_max_clock}" \
        "${emit_clock}"
      comma=','
      emitted=$((emitted + 1))
    done
  else
    if [ "${base_index}" -gt 0 ]; then
      printf ']'
      return
    fi
    printf '{"index":0,"uuid":"intel-gpu","name":"Intel GPU","util_percent":%s,"mem_util_percent":-1,"encoder_util_percent":-1,"decoder_util_percent":%s,"mem_used_mib":-1,"mem_total_mib":-1,"temp_c":-1,"power_draw_w":%s,"power_limit_w":-1,"fan_percent":-1,"sm_clock_mhz":%s,"sm_clock_max_mhz":%s,"mem_clock_mhz":-1,"mem_clock_max_mhz":-1,"graphics_clock_mhz":%s,"video_clock_mhz":-1,"pcie_gen_current":-1,"pcie_gen_max":-1,"pcie_width_current":-1,"pcie_width_max":-1,"throttle_reasons":"","p_state":""}' \
      "${top_util}" \
      "${video_util}" \
      "${power_draw}" \
      "${actual_clock}" \
      "${requested_clock}" \
      "${actual_clock}"
  fi
  printf ']'
}

json_array_body() {
  local value
  value="$(trim "${1:-}")"
  value="${value#[}"
  value="${value%]}"
  trim "${value}"
}

combine_gpu_json_arrays() {
  local first second first_body second_body
  first_body="$(json_array_body "${1:-[]}")"
  second_body="$(json_array_body "${2:-[]}")"

  printf '['
  if [ -n "${first_body}" ]; then
    printf '%s' "${first_body}"
  fi
  if [ -n "${first_body}" ] && [ -n "${second_body}" ]; then
    printf ','
  fi
  if [ -n "${second_body}" ]; then
    printf '%s' "${second_body}"
  fi
  printf ']'
}

json_array_count() {
  local body
  body="$(json_array_body "${1:-[]}")"
  if [ -z "${body}" ]; then
    printf '%s' '0'
    return
  fi

  printf '%s' "${body}" | awk '{ count += gsub(/"index"[[:space:]]*:/, "&") } END { print count + 0 }'
}

build_intel_gpu_json() {
  local xpu_discovery xpu_json intel_json xpu_count
  discover_intel_drm_devices

  xpu_discovery="$(read_xpu_smi_discovery_dump)"
  record_xpu_smi_bdfs_from_discovery "${xpu_discovery}"
  xpu_json="$(build_xpu_smi_gpu_json "${xpu_discovery}" 0)"
  xpu_count="$(json_array_count "${xpu_json}")"

  intel_json="$(build_intel_gpu_top_or_sysfs_json "${xpu_count}")"
  combine_gpu_json_arrays "${xpu_json}" "${intel_json}"
}

build_gpu_json() {
  combine_gpu_json_arrays "$(build_nvidia_gpu_json)" "$(build_intel_gpu_json)"
}
