boolish_active() {
  local value
  value="$(trim "${1:-}")"
  value="$(printf '%s' "${value}" | tr '[:upper:]' '[:lower:]')"
  value="${value// /}"
  case "${value}" in
    '' | n/a | -1 | 0 | false | no | disabled | inactive | notactive)
      return 1
      ;;
  esac
  return 0
}

summarize_gpu_throttle_reasons() {
  local reasons=''

  add_reason() {
    if [ -z "${reasons}" ]; then
      reasons="$1"
    else
      reasons="${reasons} • $1"
    fi
  }

  if boolish_active "${1:-}"; then add_reason 'power cap'; fi
  if boolish_active "${2:-}" || boolish_active "${3:-}"; then add_reason 'thermal'; fi
  if boolish_active "${4:-}"; then add_reason 'hw slow'; fi
  if boolish_active "${5:-}"; then add_reason 'sync boost'; fi
  if boolish_active "${6:-}"; then add_reason 'app clocks'; fi
  if boolish_active "${7:-}"; then add_reason 'display'; fi
  if boolish_active "${8:-}"; then add_reason 'idle'; fi
  if [ -z "${reasons}" ] && boolish_active "${9:-}"; then
    add_reason 'active'
  fi
  if [ -z "${reasons}" ]; then
    reasons='none'
  fi
  printf '%s' "${reasons}"
}

build_gpu_process_json() {
  local proc_output=''
  local gpu_uuid pid command used_mem
  local comma=''
  local count=0

  if [ -z "${nvidia_smi_path}" ]; then
    printf '[]'
    return
  fi

  if ! proc_output="$("${nvidia_smi_path}" \
    --query-compute-apps=gpu_uuid,pid,process_name,used_memory \
    --format=csv,noheader,nounits 2>/dev/null | sort -t',' -k4 -nr)"; then
    printf '[]'
    return
  fi

  printf '['
  while IFS=',' read -r gpu_uuid pid command used_mem; do
    gpu_uuid="$(trim "${gpu_uuid}")"
    pid="$(normalize_int "${pid}")"
    command="$(trim "${command}")"
    used_mem="$(normalize_int "${used_mem}")"

    if [ -z "${gpu_uuid}" ] || [ "${pid}" -lt 0 ] || [ "${used_mem}" -lt 0 ]; then
      continue
    fi

    printf '%s{"gpu_uuid":"%s","pid":%s,"command":"%s","used_mem_mib":%s}' \
      "${comma}" \
      "$(json_escape "${gpu_uuid}")" \
      "${pid}" \
      "$(json_escape "${command}")" \
      "${used_mem}"
    comma=','
    count=$((count + 1))
    if [ "${count}" -ge 4 ]; then
      break
    fi
  done <<<"${proc_output}"
  printf ']'
}

build_nvidia_gpu_json() {
  local idx uuid name util mem_util mem_used mem_total temp power_draw power_limit fan sm_clock sm_clock_max mem_clock mem_clock_max pstate
  local gpu_combined_output=''
  local gpu_output=''
  local gpu_extra_output=''
  local gpu_throttle_output=''
  local _attempt
  local comma=''
  local -a encoder_utils decoder_utils graphics_clocks video_clocks pcie_gen_current pcie_gen_max pcie_width_current pcie_width_max throttle_reasons
  local encoder_util decoder_util graphics_clock video_clock pcie_gen_cur pcie_gen_cap pcie_width_cur pcie_width_cap throttle_reason
  local sw_power_cap hw_thermal sw_thermal hw_slow sync_boost app_clocks display_clocks idle_reason active_reason

  if [ -z "${nvidia_smi_path}" ]; then
    printf '[]'
    return
  fi

  if gpu_combined_output="$("${nvidia_smi_path}" \
    --query-gpu=index,uuid,name,utilization.gpu,utilization.memory,utilization.encoder,utilization.decoder,memory.used,memory.total,temperature.gpu,power.draw,power.limit,fan.speed,clocks.sm,clocks.max.sm,clocks.mem,clocks.max.mem,clocks.gr,clocks.video,pcie.link.gen.current,pcie.link.gen.max,pcie.link.width.current,pcie.link.width.max,clocks_throttle_reasons.sw_power_cap,clocks_throttle_reasons.hw_thermal_slowdown,clocks_throttle_reasons.sw_thermal_slowdown,clocks_throttle_reasons.hw_slowdown,clocks_throttle_reasons.sync_boost,clocks_throttle_reasons.applications_clocks_setting,clocks_throttle_reasons.display_clock_setting,clocks_throttle_reasons.idle,clocks_throttle_reasons.active,pstate \
    --format=csv,noheader,nounits 2>/dev/null)" && [ -n "${gpu_combined_output}" ]; then
    printf '['
    while IFS=',' read -r idx uuid name util mem_util encoder_util decoder_util mem_used mem_total temp power_draw power_limit fan sm_clock sm_clock_max mem_clock mem_clock_max graphics_clock video_clock pcie_gen_cur pcie_gen_cap pcie_width_cur pcie_width_cap sw_power_cap hw_thermal sw_thermal hw_slow sync_boost app_clocks display_clocks idle_reason active_reason pstate; do
      idx="$(normalize_int "${idx}")"
      uuid="$(trim "${uuid}")"
      name="$(trim "${name}")"
      util="$(normalize_int "${util}")"
      mem_util="$(normalize_int "${mem_util}")"
      encoder_util="$(normalize_int "${encoder_util}")"
      decoder_util="$(normalize_int "${decoder_util}")"
      mem_used="$(normalize_int "${mem_used}")"
      mem_total="$(normalize_int "${mem_total}")"
      temp="$(normalize_int "${temp}")"
      power_draw="$(normalize_float "${power_draw}")"
      power_limit="$(normalize_float "${power_limit}")"
      fan="$(normalize_int "${fan}")"
      sm_clock="$(normalize_int "${sm_clock}")"
      sm_clock_max="$(normalize_int "${sm_clock_max}")"
      mem_clock="$(normalize_int "${mem_clock}")"
      mem_clock_max="$(normalize_int "${mem_clock_max}")"
      graphics_clock="$(normalize_int "${graphics_clock}")"
      video_clock="$(normalize_int "${video_clock}")"
      pcie_gen_cur="$(normalize_int "${pcie_gen_cur}")"
      pcie_gen_cap="$(normalize_int "${pcie_gen_cap}")"
      pcie_width_cur="$(normalize_int "${pcie_width_cur}")"
      pcie_width_cap="$(normalize_int "${pcie_width_cap}")"
      throttle_reason="$(summarize_gpu_throttle_reasons "${sw_power_cap}" "${hw_thermal}" "${sw_thermal}" "${hw_slow}" "${sync_boost}" "${app_clocks}" "${display_clocks}" "${idle_reason}" "${active_reason}")"
      pstate="$(trim "${pstate}")"

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
      comma=','
    done <<<"${gpu_combined_output}"
    printf ']'
    return
  fi

  for _attempt in 1 2; do
    if gpu_output="$("${nvidia_smi_path}" \
      --query-gpu=index,uuid,name,utilization.gpu,utilization.memory,memory.used,memory.total,temperature.gpu,power.draw,power.limit,fan.speed,clocks.sm,clocks.max.sm,clocks.mem,clocks.max.mem,pstate \
      --format=csv,noheader,nounits 2>/dev/null)" && [ -n "${gpu_output}" ]; then
      break
    fi
    gpu_output=''
    sleep 0.1
  done

  if [ -z "${gpu_output}" ]; then
    printf '[]'
    return
  fi

  if gpu_extra_output="$("${nvidia_smi_path}" \
    --query-gpu=index,utilization.encoder,utilization.decoder,clocks.gr,clocks.video,pcie.link.gen.current,pcie.link.gen.max,pcie.link.width.current,pcie.link.width.max \
    --format=csv,noheader,nounits 2>/dev/null)" && [ -n "${gpu_extra_output}" ]; then
    while IFS=',' read -r idx encoder_util decoder_util graphics_clock video_clock pcie_gen_cur pcie_gen_cap pcie_width_cur pcie_width_cap; do
      idx="$(normalize_int "${idx}")"
      if [ "${idx}" -lt 0 ]; then
        continue
      fi
      encoder_utils[idx]="$(normalize_int "${encoder_util}")"
      decoder_utils[idx]="$(normalize_int "${decoder_util}")"
      graphics_clocks[idx]="$(normalize_int "${graphics_clock}")"
      video_clocks[idx]="$(normalize_int "${video_clock}")"
      pcie_gen_current[idx]="$(normalize_int "${pcie_gen_cur}")"
      pcie_gen_max[idx]="$(normalize_int "${pcie_gen_cap}")"
      pcie_width_current[idx]="$(normalize_int "${pcie_width_cur}")"
      pcie_width_max[idx]="$(normalize_int "${pcie_width_cap}")"
    done <<<"${gpu_extra_output}"
  fi

  if gpu_throttle_output="$("${nvidia_smi_path}" \
    --query-gpu=index,clocks_throttle_reasons.sw_power_cap,clocks_throttle_reasons.hw_thermal_slowdown,clocks_throttle_reasons.sw_thermal_slowdown,clocks_throttle_reasons.hw_slowdown,clocks_throttle_reasons.sync_boost,clocks_throttle_reasons.applications_clocks_setting,clocks_throttle_reasons.display_clock_setting,clocks_throttle_reasons.idle,clocks_throttle_reasons.active \
    --format=csv,noheader,nounits 2>/dev/null)" && [ -n "${gpu_throttle_output}" ]; then
    while IFS=',' read -r idx sw_power_cap hw_thermal sw_thermal hw_slow sync_boost app_clocks display_clocks idle_reason active_reason; do
      idx="$(normalize_int "${idx}")"
      if [ "${idx}" -lt 0 ]; then
        continue
      fi
      throttle_reasons[idx]="$(summarize_gpu_throttle_reasons "${sw_power_cap}" "${hw_thermal}" "${sw_thermal}" "${hw_slow}" "${sync_boost}" "${app_clocks}" "${display_clocks}" "${idle_reason}" "${active_reason}")"
    done <<<"${gpu_throttle_output}"
  fi

  printf '['
  while IFS=',' read -r idx uuid name util mem_util mem_used mem_total temp power_draw power_limit fan sm_clock sm_clock_max mem_clock mem_clock_max pstate; do
    idx="$(normalize_int "${idx}")"
    uuid="$(trim "${uuid}")"
    name="$(trim "${name}")"
    util="$(normalize_int "${util}")"
    mem_util="$(normalize_int "${mem_util}")"
    mem_used="$(normalize_int "${mem_used}")"
    mem_total="$(normalize_int "${mem_total}")"
    temp="$(normalize_int "${temp}")"
    power_draw="$(normalize_float "${power_draw}")"
    power_limit="$(normalize_float "${power_limit}")"
    fan="$(normalize_int "${fan}")"
    sm_clock="$(normalize_int "${sm_clock}")"
    sm_clock_max="$(normalize_int "${sm_clock_max}")"
    mem_clock="$(normalize_int "${mem_clock}")"
    mem_clock_max="$(normalize_int "${mem_clock_max}")"
    encoder_util="${encoder_utils[idx]:--1}"
    decoder_util="${decoder_utils[idx]:--1}"
    graphics_clock="${graphics_clocks[idx]:--1}"
    video_clock="${video_clocks[idx]:--1}"
    pcie_gen_cur="${pcie_gen_current[idx]:--1}"
    pcie_gen_cap="${pcie_gen_max[idx]:--1}"
    pcie_width_cur="${pcie_width_current[idx]:--1}"
    pcie_width_cap="${pcie_width_max[idx]:--1}"
    throttle_reason="${throttle_reasons[idx]:-}"
    pstate="$(trim "${pstate}")"

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
    comma=','
  done <<<"${gpu_output}"
  printf ']'
}
