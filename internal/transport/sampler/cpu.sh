read_cpu_snapshot() {
  local label user nice system idle iowait irq softirq steal guest guest_nice
  local idle_all
  local total
  local user_all
  local system_all

  cpu_labels=()
  cpu_idle=()
  cpu_total=()
  cpu_user=()
  cpu_system=()
  cpu_iowait=()
  cpu_steal=()

  while read -r label user nice system idle iowait irq softirq steal guest guest_nice; do
    case "${label}" in
      cpu|cpu[0-9]*)
        user_all=$((user + nice))
        system_all=$((system + irq + softirq))
        idle_all=$((idle + iowait))
        total=$((user_all + system_all + idle + iowait + steal))
        cpu_labels+=("${label}")
        cpu_idle+=("${idle_all}")
        cpu_total+=("${total}")
        cpu_user+=("${user_all}")
        cpu_system+=("${system_all}")
        cpu_iowait+=("${iowait}")
        cpu_steal+=("${steal}")
        ;;
    esac
  done < /proc/stat
}

now_ns() {
  awk '{ printf "%.0f\n", $1 * 1000000000 }' /proc/uptime
}

elapsed_ms_or_default() {
  local elapsed_ms="${sample_elapsed_ms:-0}"
  if [ "${elapsed_ms}" -lt 1 ]; then
    elapsed_ms=$((interval * 1000))
  fi
  if [ "${elapsed_ms}" -lt 1 ]; then
    elapsed_ms=1000
  fi
  printf '%s\n' "${elapsed_ms}"
}

build_cpu_core_json() {
  local i
  local core_label
  local core_index
  local core_pct
  local diff_idle_core
  local diff_total_core
  local comma=""

  printf '['
  for ((i = 1; i < ${#cpu_labels[@]}; i++)); do
    core_label="${cpu_labels[i]}"
    core_index="${core_label#cpu}"
    core_pct=-1

    if [ "${i}" -lt "${#prev_cpu_total[@]}" ] && [ "${prev_cpu_labels[i]}" = "${core_label}" ]; then
      diff_idle_core=$((cpu_idle[i] - prev_cpu_idle[i]))
      diff_total_core=$((cpu_total[i] - prev_cpu_total[i]))
      core_pct=0
      if [ "${diff_total_core}" -gt 0 ]; then
        core_pct=$(((100 * (diff_total_core - diff_idle_core)) / diff_total_core))
      fi
    fi

    printf '%s{"index":%s,"percent":%s}' "${comma}" "${core_index}" "${core_pct}"
    comma=","
  done
  printf ']'
}

read_cpu_freq_stats() {
  local current_sum=0
  local current_count=0
  local max_sum=0
  local max_count=0
  local current_mhz='-1'
  local max_mhz='-1'
  local path
  local value
  local max_path

  shopt -s nullglob
  for path in /sys/devices/system/cpu/cpu[0-9]*/cpufreq/scaling_cur_freq; do
    value="$(tr -d '[:space:]' < "${path}" 2>/dev/null || printf '%s' '')"
    case "${value}" in
      ''|*[!0-9]*)
        continue
        ;;
    esac
    current_sum=$((current_sum + value))
    current_count=$((current_count + 1))

    max_path="${path%/scaling_cur_freq}/cpuinfo_max_freq"
    if [ -r "${max_path}" ]; then
      value="$(tr -d '[:space:]' < "${max_path}" 2>/dev/null || printf '%s' '')"
      case "${value}" in
        ''|*[!0-9]*)
          ;;
        *)
          max_sum=$((max_sum + value))
          max_count=$((max_count + 1))
          ;;
      esac
    fi
  done
  shopt -u nullglob

  if [ "${current_count}" -gt 0 ]; then
    current_mhz=$((current_sum / current_count / 1000))
  else
    current_mhz="$(awk -F: '
      /^cpu MHz/ { sum += $2; count++ }
      END {
        if (count > 0) {
          printf "%d\n", sum / count
        } else {
          print -1
        }
      }
    ' /proc/cpuinfo 2>/dev/null)"
  fi

  if [ "${max_count}" -gt 0 ]; then
    max_mhz=$((max_sum / max_count / 1000))
  elif command -v lscpu >/dev/null 2>&1; then
    max_mhz="$(LC_ALL=C lscpu 2>/dev/null | awk -F: '
      /^CPU max MHz/ || /^Max MHz/ {
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", $2)
        if ($2 ~ /^[0-9]+([.][0-9]+)?$/) {
          printf "%d\n", $2
          found=1
          exit
        }
      }
      END {
        if (!found) {
          print -1
        }
      }
    ')"
  fi

  printf '%s|%s\n' "${current_mhz}" "${max_mhz}"
}

read_cpu_model_name() {
  local cpu_name

  cpu_name="$(awk -F: '
    /^model name[[:space:]]*:/ || /^Processor[[:space:]]*:/ || /^Hardware[[:space:]]*:/ {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", $2)
      if ($2 != "") {
        print $2
        exit
      }
    }
  ' /proc/cpuinfo 2>/dev/null)"

  if [ -z "${cpu_name}" ] && command -v lscpu >/dev/null 2>&1; then
    cpu_name="$(LC_ALL=C lscpu 2>/dev/null | awk -F: '
      /^Model name/ {
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", $2)
        if ($2 != "") {
          print $2
          exit
        }
      }
    ')"
  fi

  printf '%s\n' "${cpu_name}"
}

read_cpu_temp_c() {
  local best='-1'
  local type_path
  local type_name
  local temp_path
  local raw
  local value
  local hwmon_path
  local hwmon_name

  shopt -s nullglob
  for type_path in /sys/class/thermal/thermal_zone*/type; do
    [ -r "${type_path}" ] || continue
    type_name="$(tr '[:upper:]' '[:lower:]' < "${type_path}" 2>/dev/null || printf '%s' '')"
    case "${type_name}" in
      *x86_pkg_temp*|*package*|*coretemp*|*cpu*|*k10temp*|*tctl*|*tdie*)
        temp_path="${type_path%/type}/temp"
        if [ -r "${temp_path}" ]; then
          raw="$(tr -d '[:space:]' < "${temp_path}" 2>/dev/null || printf '%s' '')"
          case "${raw}" in
            ''|*[!0-9-]*)
              continue
              ;;
          esac
          value="${raw}"
          if [ "${value}" -gt 1000 ] 2>/dev/null; then
            value=$((value / 1000))
          fi
          if [ "${value}" -gt "${best}" ]; then
            best="${value}"
          fi
        fi
        ;;
    esac
  done

  if [ "${best}" -lt 0 ] 2>/dev/null; then
    for hwmon_path in /sys/class/hwmon/hwmon*; do
      [ -r "${hwmon_path}/name" ] || continue
      hwmon_name="$(tr '[:upper:]' '[:lower:]' < "${hwmon_path}/name" 2>/dev/null || printf '%s' '')"
      case "${hwmon_name}" in
        coretemp|k10temp|zenpower|cpu_thermal)
          for temp_path in "${hwmon_path}"/temp*_input; do
            [ -r "${temp_path}" ] || continue
            raw="$(tr -d '[:space:]' < "${temp_path}" 2>/dev/null || printf '%s' '')"
            case "${raw}" in
              ''|*[!0-9-]*)
                continue
                ;;
            esac
            value="${raw}"
            if [ "${value}" -gt 1000 ] 2>/dev/null; then
              value=$((value / 1000))
            fi
            if [ "${value}" -gt "${best}" ]; then
              best="${value}"
            fi
          done
          ;;
      esac
    done
  fi
  shopt -u nullglob

  printf '%s\n' "${best}"
}
