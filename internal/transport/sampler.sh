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

read_top_process_snapshot() {
  if ! command -v ps >/dev/null 2>&1; then
    return
  fi

  LC_ALL=C ps -eo pid=,pcpu=,rss=,comm= --sort=-pcpu,-rss 2>/dev/null | awk -v self="$$" -v limit=4 '
    BEGIN { count = 0 }
    {
      pid = $1
      cpu = $2 + 0
      rss = $3 + 0
      cmd = $4

      if (pid == self || cmd == "") {
        next
      }
      if (cmd ~ /^(ps|awk|sort|head|tail|sleep|sampler\.sh)$/) {
        next
      }

      cpu_int = int(cpu + 0.5)
      if (cpu_int < 0) {
        cpu_int = 0
      }
      rss_mib = int(rss / 1024)
      if (rss_mib < 0) {
        rss_mib = 0
      }

      printf "%s|%s|%s|%s\n", pid, cpu_int, rss_mib, cmd
      count++
      if (count >= limit) {
        exit
      }
    }
  '
}

build_top_process_json() {
  local pid cpu rss cmd
  local comma=''

  printf '['
  while IFS='|' read -r pid cpu rss cmd; do
    [ -n "${pid}" ] || continue
    printf '%s{"pid":%s,"command":"%s","cpu_percent":%s,"rss_mib":%s}' \
      "${comma}" \
      "${pid}" \
      "$(json_escape "${cmd}")" \
      "${cpu}" \
      "${rss}"
    comma=','
  done < <(read_top_process_snapshot)
  printf ']'
}

read_swap_stats() {
  awk '
    /^SwapFree:/ { swap_free=$2 }
    /^SwapTotal:/ { swap_total=$2 }
    END {
      printf "%s|%s\n", swap_free+0, swap_total+0
    }
  ' /proc/meminfo
}

read_swap_io_sample() {
  awk '
    /^pswpin / { in_pages=$2 }
    /^pswpout / { out_pages=$2 }
    END {
      printf "%s|%s\n", in_pages+0, out_pages+0
    }
  ' /proc/vmstat 2>/dev/null
}

read_ram_stats() {
  awk '
    /^MemTotal:/ { total_kib=$2 }
    /^MemAvailable:/ { avail_kib=$2 }
    /^MemFree:/ { free_kib=$2 }
    /^Buffers:/ { buffers_kib=$2 }
    /^Cached:/ { cached_kib=$2 }
    /^SReclaimable:/ { reclaimable_kib=$2 }
    /^Shmem:/ { shmem_kib=$2 }
    END {
      if (avail_kib == 0) {
        avail_kib = free_kib + buffers_kib + cached_kib + reclaimable_kib - shmem_kib
      }
      if (avail_kib < 0) {
        avail_kib = 0
      }
      total_mib = int(total_kib / 1024)
      used_mib = int((total_kib - avail_kib) / 1024)
      cache_mib = int((cached_kib + buffers_kib + reclaimable_kib) / 1024)
      if (used_mib < 0) {
        used_mib = 0
      }
      if (cache_mib < 0) {
        cache_mib = 0
      }
      printf "%s|%s|%s|%s|%s|%s|%s|%s\n", used_mib, total_mib, int(avail_kib / 1024), int(free_kib / 1024), int(cached_kib / 1024), int(buffers_kib / 1024), int(reclaimable_kib / 1024), int(shmem_kib / 1024)
    }
  ' /proc/meminfo
}

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
  done < "${path}"

  printf '%s|%s\n' "$(normalize_float "${some}")" "$(normalize_float "${full}")"
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

is_wsl_environment() {
  local osrelease_path="${1:-/proc/sys/kernel/osrelease}"
  local version_path="${2:-/proc/version}"
  local content

  if [ -n "${WSL_DISTRO_NAME:-}" ] || [ -n "${WSL_INTEROP:-}" ]; then
    return 0
  fi

  if [ -r "${osrelease_path}" ]; then
    content="$(< "${osrelease_path}")"
    content="${content,,}"
    case "${content}" in
      *microsoft*|*wsl*)
        return 0
        ;;
    esac
  fi

  if [ -r "${version_path}" ]; then
    content="$(< "${version_path}")"
    content="${content,,}"
    case "${content}" in
      *microsoft*|*wsl*)
        return 0
        ;;
    esac
  fi

  return 1
}

wsl_host_metrics_enabled() {
  local value
  value="$(trim "${REMOTE_MONITOR_WSL_HOST_METRICS:-}")"
  value="${value,,}"
  value="${value// /}"

  case "${value}" in
    0|false|no|off|disabled)
      return 1
      ;;
  esac

  return 0
}

find_wsl_powershell_from_candidates() {
  local candidate

  for candidate in "$@"; do
    [ -n "${candidate}" ] || continue
    case "${candidate}" in
      */*)
        if [ -x "${candidate}" ]; then
          printf '%s\n' "${candidate}"
          return 0
        fi
        ;;
      *)
        if command -v "${candidate}" >/dev/null 2>&1; then
          command -v "${candidate}"
          return 0
        fi
        ;;
    esac
  done

  return 1
}

find_wsl_powershell() {
  find_wsl_powershell_from_candidates \
    powershell.exe \
    pwsh.exe \
    '/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe' \
    '/mnt/c/Program Files/PowerShell/7/pwsh.exe' \
    '/mnt/c/Program Files/PowerShell/7-preview/pwsh.exe'
}

json_int_field() {
  local json="$1"
  local field="$2"
  local pattern

  pattern="\"${field}\"[[:space:]]*:[[:space:]]*\"?(-?[0-9]+)\"?"
  if [[ "${json}" =~ ${pattern} ]]; then
    normalize_int "${BASH_REMATCH[1]}"
    return
  fi

  printf '%s\n' '-1'
}

json_string_field() {
  local json="$1"
  local field="$2"
  local pattern
  local value

  pattern="\"${field}\"[[:space:]]*:[[:space:]]*\"([^\"]*)\""
  if [[ "${json}" =~ ${pattern} ]]; then
    value="${BASH_REMATCH[1]}"
    value="${value//\\\"/\"}"
    value="${value//\\\\/\\}"
    value="${value//\\r/ }"
    value="${value//\\n/ }"
    value="${value//\\t/ }"
    printf '%s\n' "$(trim "${value}")"
    return
  fi

  printf '\n'
}

read_wsl_windows_host_metrics_json() {
  local osrelease_path="${1:-/proc/sys/kernel/osrelease}"
  local version_path="${2:-/proc/version}"
  local powershell_path
  local powershell_timeout="${REMOTE_MONITOR_WSL_HOST_METRICS_TIMEOUT:-2s}"
  local powershell_script
  local output

  if ! wsl_host_metrics_enabled; then
    printf '\n'
    return
  fi
  if ! is_wsl_environment "${osrelease_path}" "${version_path}"; then
    printf '\n'
    return
  fi
  if ! powershell_path="$(find_wsl_powershell)"; then
    printf '\n'
    return
  fi
  if ! command -v timeout >/dev/null 2>&1; then
    printf '\n'
    return
  fi

  powershell_script='$ErrorActionPreference = "SilentlyContinue"; $processors = @(Get-CimInstance -ClassName "Win32_Processor"); $os = Get-CimInstance -ClassName "Win32_OperatingSystem"; $cpuTemp = $null; foreach ($sensorNamespace in @("root/LibreHardwareMonitor","root/OpenHardwareMonitor")) { $sensorValues = @(Get-CimInstance -Namespace $sensorNamespace -ClassName "Sensor" | Where-Object { ($_.SensorType -eq "Temperature" -or $_.SensorType -eq 2) -and (($_.Name -match "CPU|Package|Core|Tctl|Tdie|CCD") -or ($_.Identifier -match "/(cpu|intelcpu|amdcpu)(/|$)" -and $_.Identifier -match "/temperature/")) -and ($_.Value -ge 1 -and $_.Value -le 125) } | ForEach-Object { [math]::Round([double]$_.Value) }); if ($sensorValues.Count -gt 0) { $cpuTemp = ($sensorValues | Measure-Object -Maximum).Maximum; break } }; $cpuName = $null; $cpuCores = $null; $cpuFreq = $null; $cpuMaxFreq = $null; if ($processors.Count -gt 0) { $cpuName = ($processors | Select-Object -First 1).Name; $cpuCores = ($processors | Measure-Object -Property NumberOfLogicalProcessors -Sum).Sum; $cpuFreq = [math]::Round(($processors | Measure-Object -Property CurrentClockSpeed -Average).Average); $cpuMaxFreq = ($processors | Measure-Object -Property MaxClockSpeed -Maximum).Maximum }; $ramTotal = $null; $ramFree = $null; $ramUsed = $null; if ($null -ne $os) { $totalKiB = [double]$os.TotalVisibleMemorySize; $freeKiB = [double]$os.FreePhysicalMemory; if ($totalKiB -gt 0 -and $freeKiB -ge 0) { $ramTotal = [math]::Round($totalKiB / 1024); $ramFree = [math]::Round($freeKiB / 1024); $ramUsed = [math]::Max(0, $ramTotal - $ramFree) } }; [ordered]@{cpu_temp_c=$cpuTemp; cpu_name=$cpuName; cpu_cores=$cpuCores; cpu_freq_mhz=$cpuFreq; cpu_max_freq_mhz=$cpuMaxFreq; ram_used_mib=$ramUsed; ram_total_mib=$ramTotal; ram_available_mib=$ramFree; ram_free_mib=$ramFree} | ConvertTo-Json -Compress'
  if ! output="$(timeout "${powershell_timeout}" "${powershell_path}" -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "${powershell_script}" 2>/dev/null)"; then
    printf '\n'
    return
  fi

  printf '%s\n' "$(trim "${output}")"
}

apply_wsl_host_metrics() {
  local host_cpu_name
  local host_cpu_cores
  local host_cpu_freq
  local host_cpu_max_freq
  local host_cpu_temp
  local host_ram_used
  local host_ram_total
  local host_ram_available
  local host_ram_free

  if [ -z "${wsl_host_metrics_json:-}" ]; then
    return
  fi

  host_cpu_name="$(json_string_field "${wsl_host_metrics_json}" 'cpu_name')"
  if [ -n "${host_cpu_name}" ]; then
    remote_cpu_name="${host_cpu_name}"
  fi

  host_cpu_cores="$(json_int_field "${wsl_host_metrics_json}" 'cpu_cores')"
  if [ "${host_cpu_cores}" -gt 0 ] 2>/dev/null && [ "${host_cpu_cores}" -le 1024 ] 2>/dev/null; then
    remote_cpu_cores="${host_cpu_cores}"
  fi

  host_cpu_freq="$(json_int_field "${wsl_host_metrics_json}" 'cpu_freq_mhz')"
  if [ "${host_cpu_freq}" -gt 0 ] 2>/dev/null && [ "${host_cpu_freq}" -le 100000 ] 2>/dev/null; then
    cpu_freq_mhz="${host_cpu_freq}"
  fi

  host_cpu_max_freq="$(json_int_field "${wsl_host_metrics_json}" 'cpu_max_freq_mhz')"
  if [ "${host_cpu_max_freq}" -gt 0 ] 2>/dev/null && [ "${host_cpu_max_freq}" -le 100000 ] 2>/dev/null; then
    cpu_max_freq_mhz="${host_cpu_max_freq}"
  fi

  host_cpu_temp="$(json_int_field "${wsl_host_metrics_json}" 'cpu_temp_c')"
  if [ "${cpu_temp_c}" -lt 0 ] 2>/dev/null && \
     [ "${host_cpu_temp}" -ge 1 ] 2>/dev/null && [ "${host_cpu_temp}" -le 125 ] 2>/dev/null; then
    cpu_temp_c="${host_cpu_temp}"
  fi

  host_ram_used="$(json_int_field "${wsl_host_metrics_json}" 'ram_used_mib')"
  host_ram_total="$(json_int_field "${wsl_host_metrics_json}" 'ram_total_mib')"
  host_ram_available="$(json_int_field "${wsl_host_metrics_json}" 'ram_available_mib')"
  host_ram_free="$(json_int_field "${wsl_host_metrics_json}" 'ram_free_mib')"
  if [ "${host_ram_total}" -gt 0 ] 2>/dev/null && \
     [ "${host_ram_used}" -ge 0 ] 2>/dev/null && [ "${host_ram_used}" -le "${host_ram_total}" ] 2>/dev/null && \
     [ "${host_ram_available}" -ge 0 ] 2>/dev/null && [ "${host_ram_available}" -le "${host_ram_total}" ] 2>/dev/null && \
     [ "${host_ram_free}" -ge 0 ] 2>/dev/null && [ "${host_ram_free}" -le "${host_ram_total}" ] 2>/dev/null; then
    ram_used="${host_ram_used}"
    ram_total="${host_ram_total}"
    ram_available="${host_ram_available}"
    ram_free="${host_ram_free}"
    ram_cache='-1'
    ram_buffers='-1'
    ram_reclaimable='-1'
    ram_shared='-1'
    mem_pressure_some='-1'
    mem_pressure_full='-1'
  fi
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

read_disk_sample() {
  local device="$1"

  if [ -z "${device}" ]; then
    printf '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n' '-1' '-1' '-1' '-1' '-1' '-1' '-1' '-1' '-1' '-1' '-1'
    return
  fi

  awk -v dev="${device}" '
    $3 == dev {
      printf "%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n", $6, $10, $13, $4, $5, $7, $8, $9, $11, $12, $14
      found=1
    }
    END {
      if (!found) {
        printf "%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n", -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1
      }
    }
  ' /proc/diskstats
}

build_disk_json() {
  local root_source root_used_kib root_total_kib root_used_pct
  local disk_sectors_read disk_sectors_written disk_io_ms
  local disk_reads_completed disk_reads_merged disk_read_ms disk_writes_completed disk_writes_merged disk_write_ms disk_in_flight disk_weighted_ms
  local read_bps='-1'
  local write_bps='-1'
  local read_merged_per_sec='-1'
  local write_merged_per_sec='-1'
  local disk_util='-1'
  local disk_await='-1'
  local disk_queue='-1'
  local elapsed_ms

  elapsed_ms="$(elapsed_ms_or_default)"

  IFS='|' read -r root_source root_used_kib combined < <(cached_root_usage)
  root_total_kib="${combined%%|*}"
  root_used_pct="${combined##*|}"
  IFS='|' read -r disk_sectors_read disk_sectors_written disk_io_ms disk_reads_completed disk_reads_merged disk_read_ms disk_writes_completed disk_writes_merged disk_write_ms disk_in_flight disk_weighted_ms < <(read_disk_sample "${root_device}")

  if [ "${disk_sectors_read}" -ge 0 ] && [ "${prev_disk_sectors_read}" -ge 0 ]; then
    read_bps=$((((disk_sectors_read - prev_disk_sectors_read) * 512 * 1000) / elapsed_ms))
    if [ "${read_bps}" -lt 0 ]; then
      read_bps=0
    fi
  fi

  if [ "${disk_sectors_written}" -ge 0 ] && [ "${prev_disk_sectors_written}" -ge 0 ]; then
    write_bps=$((((disk_sectors_written - prev_disk_sectors_written) * 512 * 1000) / elapsed_ms))
    if [ "${write_bps}" -lt 0 ]; then
      write_bps=0
    fi
  fi

  if [ "${disk_io_ms}" -ge 0 ] && [ "${prev_disk_io_ms}" -ge 0 ]; then
    disk_util=$((((disk_io_ms - prev_disk_io_ms) * 100) / elapsed_ms))
    if [ "${disk_util}" -lt 0 ]; then
      disk_util=0
    elif [ "${disk_util}" -gt 100 ]; then
      disk_util=100
    fi
  fi

  if [ "${disk_reads_merged}" -ge 0 ] && [ "${prev_disk_reads_merged}" -ge 0 ]; then
    read_merged_per_sec=$((((disk_reads_merged - prev_disk_reads_merged) * 1000) / elapsed_ms))
    if [ "${read_merged_per_sec}" -lt 0 ]; then
      read_merged_per_sec=0
    fi
  fi

  if [ "${disk_writes_merged}" -ge 0 ] && [ "${prev_disk_writes_merged}" -ge 0 ]; then
    write_merged_per_sec=$((((disk_writes_merged - prev_disk_writes_merged) * 1000) / elapsed_ms))
    if [ "${write_merged_per_sec}" -lt 0 ]; then
      write_merged_per_sec=0
    fi
  fi

  if [ "${disk_reads_completed}" -ge 0 ] && [ "${prev_disk_reads_completed}" -ge 0 ] && \
     [ "${disk_writes_completed}" -ge 0 ] && [ "${prev_disk_writes_completed}" -ge 0 ] && \
     [ "${disk_read_ms}" -ge 0 ] && [ "${prev_disk_read_ms}" -ge 0 ] && \
     [ "${disk_write_ms}" -ge 0 ] && [ "${prev_disk_write_ms}" -ge 0 ]; then
    disk_ops_delta=$(((disk_reads_completed - prev_disk_reads_completed) + (disk_writes_completed - prev_disk_writes_completed)))
    disk_service_ms_delta=$(((disk_read_ms - prev_disk_read_ms) + (disk_write_ms - prev_disk_write_ms)))
    if [ "${disk_ops_delta}" -gt 0 ]; then
      disk_await="$(awk -v service_ms="${disk_service_ms_delta}" -v ops="${disk_ops_delta}" 'BEGIN { printf "%.2f", service_ms / ops }')"
    else
      disk_await='0.00'
    fi
  fi

  if [ "${disk_weighted_ms}" -ge 0 ] && [ "${prev_disk_weighted_ms}" -ge 0 ]; then
    disk_queue="$(awk -v weighted_ms="$((disk_weighted_ms - prev_disk_weighted_ms))" -v elapsed_ms="${elapsed_ms}" 'BEGIN {
      if (elapsed_ms <= 0 || weighted_ms < 0) {
        printf "%.2f", 0
      } else {
        printf "%.2f", weighted_ms / elapsed_ms
      }
    }')"
  fi

  printf '{"root_source":"%s","root_used_kib":%s,"root_total_kib":%s,"root_used_percent":%s,"device":"%s","read_bps":%s,"write_bps":%s,"read_merged_per_sec":%s,"write_merged_per_sec":%s,"util_percent":%s,"await_ms":%s,"queue_depth":%s,"inflight":%s}' \
    "$(json_escape "${root_source}")" \
    "${root_used_kib}" \
    "${root_total_kib}" \
    "${root_used_pct}" \
    "$(json_escape "${root_device}")" \
    "${read_bps}" \
    "${write_bps}" \
    "${read_merged_per_sec}" \
    "${write_merged_per_sec}" \
    "${disk_util}" \
    "${disk_await}" \
    "${disk_queue}" \
    "$(normalize_int "${disk_in_flight}")"

  prev_disk_sectors_read="${disk_sectors_read}"
  prev_disk_sectors_written="${disk_sectors_written}"
  prev_disk_io_ms="${disk_io_ms}"
  prev_disk_reads_completed="${disk_reads_completed}"
  prev_disk_reads_merged="${disk_reads_merged}"
  prev_disk_read_ms="${disk_read_ms}"
  prev_disk_writes_completed="${disk_writes_completed}"
  prev_disk_writes_merged="${disk_writes_merged}"
  prev_disk_write_ms="${disk_write_ms}"
  prev_disk_in_flight="${disk_in_flight}"
  prev_disk_weighted_ms="${disk_weighted_ms}"
}

add_tracked_net_iface() {
  local iface="$1"
  local i

  if [ -z "${iface}" ] || [ "${iface}" = 'lo' ]; then
    return
  fi

  for ((i = 0; i < ${#tracked_net_ifaces[@]}; i++)); do
    if [ "${tracked_net_ifaces[i]}" = "${iface}" ]; then
      return
    fi
  done

  tracked_net_ifaces+=("${iface}")
}

discover_net_ifaces() {
  local iface
  local primary_iface=''

  tracked_net_ifaces=()
  primary_iface="$(ip route show default 2>/dev/null | awk 'NR==1 {print $5}')"
  add_tracked_net_iface "${primary_iface}"

  if [ -d /sys/class/net/tailscale0 ]; then
    add_tracked_net_iface 'tailscale0'
  fi

  while read -r iface; do
    add_tracked_net_iface "${iface}"
  done < <(awk -F: 'NR > 2 { gsub(/ /, "", $1); print $1 }' /proc/net/dev)
}

read_net_sample() {
  local iface="$1"

  awk -v iface="${iface}" -F ':' '
    {
      name=$1
      gsub(/ /, "", name)
    }
    name == iface {
      split($2, fields, /[[:space:]]+/)
      printf "%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n", fields[2], fields[10], fields[3], fields[11], fields[5], fields[4], fields[6], fields[13], fields[12], fields[14]
      found=1
    }
    END {
      if (!found) {
        printf "%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n", -1, -1, -1, -1, -1, -1, -1, -1, -1, -1
      }
    }
  ' /proc/net/dev
}

read_net_speed_mbps() {
  local iface="$1"
  local speed_path="/sys/class/net/${iface}/speed"
  local speed='-1'

  if [ -r "${speed_path}" ]; then
    speed="$(tr -d '[:space:]' < "${speed_path}" 2>/dev/null || printf '%s' '-1')"
    case "${speed}" in
      ''|*[!0-9-]*)
        speed='-1'
        ;;
    esac
    if [ "${speed}" -le 0 ] 2>/dev/null; then
      speed='-1'
    fi
  fi

  printf '%s\n' "${speed}"
}

build_net_json() {
  local i
  local current_rx
  local current_tx
  local current_rx_packets current_tx_packets
  local speed_mbps
  local current_rx_drops current_rx_errors current_rx_overruns current_tx_drops current_tx_errors current_tx_overruns
  local rx_bps='-1'
  local tx_bps='-1'
  local rx_pps='-1'
  local tx_pps='-1'
  local rx_drops='-1'
  local rx_errors='-1'
  local rx_overruns='-1'
  local tx_drops='-1'
  local tx_errors='-1'
  local tx_overruns='-1'
  local comma=''
  local elapsed_ms

  elapsed_ms="$(elapsed_ms_or_default)"

  printf '['
  for ((i = 0; i < ${#tracked_net_ifaces[@]}; i++)); do
    IFS='|' read -r current_rx current_tx current_rx_packets current_tx_packets current_rx_drops current_rx_errors current_rx_overruns current_tx_drops current_tx_errors current_tx_overruns < <(read_net_sample "${tracked_net_ifaces[i]}")
    speed_mbps="$(read_net_speed_mbps "${tracked_net_ifaces[i]}")"

    if [ "${current_rx}" -ge 0 ] && [ "${prev_net_rx[i]:--1}" -ge 0 ]; then
      rx_bps=$((((current_rx - prev_net_rx[i]) * 1000) / elapsed_ms))
      if [ "${rx_bps}" -lt 0 ]; then
        rx_bps=0
      fi
    else
      rx_bps='-1'
    fi

    if [ "${current_tx}" -ge 0 ] && [ "${prev_net_tx[i]:--1}" -ge 0 ]; then
      tx_bps=$((((current_tx - prev_net_tx[i]) * 1000) / elapsed_ms))
      if [ "${tx_bps}" -lt 0 ]; then
        tx_bps=0
      fi
    else
      tx_bps='-1'
    fi

    if [ "${current_rx_packets}" -ge 0 ] && [ "${prev_net_rx_packets[i]:--1}" -ge 0 ]; then
      rx_pps=$((((current_rx_packets - prev_net_rx_packets[i]) * 1000) / elapsed_ms))
      if [ "${rx_pps}" -lt 0 ]; then
        rx_pps=0
      fi
    else
      rx_pps='-1'
    fi

    if [ "${current_tx_packets}" -ge 0 ] && [ "${prev_net_tx_packets[i]:--1}" -ge 0 ]; then
      tx_pps=$((((current_tx_packets - prev_net_tx_packets[i]) * 1000) / elapsed_ms))
      if [ "${tx_pps}" -lt 0 ]; then
        tx_pps=0
      fi
    else
      tx_pps='-1'
    fi

    if [ "${current_rx_drops}" -ge 0 ] && [ "${prev_net_rx_drops[i]:--1}" -ge 0 ]; then
      rx_drops=$((current_rx_drops - prev_net_rx_drops[i]))
      if [ "${rx_drops}" -lt 0 ]; then
        rx_drops=0
      fi
    else
      rx_drops='-1'
    fi

    if [ "${current_rx_errors}" -ge 0 ] && [ "${prev_net_rx_errors[i]:--1}" -ge 0 ]; then
      rx_errors=$((current_rx_errors - prev_net_rx_errors[i]))
      if [ "${rx_errors}" -lt 0 ]; then
        rx_errors=0
      fi
    else
      rx_errors='-1'
    fi

    if [ "${current_rx_overruns}" -ge 0 ] && [ "${prev_net_rx_overruns[i]:--1}" -ge 0 ]; then
      rx_overruns=$((current_rx_overruns - prev_net_rx_overruns[i]))
      if [ "${rx_overruns}" -lt 0 ]; then
        rx_overruns=0
      fi
    else
      rx_overruns='-1'
    fi

    if [ "${current_tx_drops}" -ge 0 ] && [ "${prev_net_tx_drops[i]:--1}" -ge 0 ]; then
      tx_drops=$((current_tx_drops - prev_net_tx_drops[i]))
      if [ "${tx_drops}" -lt 0 ]; then
        tx_drops=0
      fi
    else
      tx_drops='-1'
    fi

    if [ "${current_tx_errors}" -ge 0 ] && [ "${prev_net_tx_errors[i]:--1}" -ge 0 ]; then
      tx_errors=$((current_tx_errors - prev_net_tx_errors[i]))
      if [ "${tx_errors}" -lt 0 ]; then
        tx_errors=0
      fi
    else
      tx_errors='-1'
    fi

    if [ "${current_tx_overruns}" -ge 0 ] && [ "${prev_net_tx_overruns[i]:--1}" -ge 0 ]; then
      tx_overruns=$((current_tx_overruns - prev_net_tx_overruns[i]))
      if [ "${tx_overruns}" -lt 0 ]; then
        tx_overruns=0
      fi
    else
      tx_overruns='-1'
    fi

    printf '%s{"iface":"%s","rx_bps":%s,"tx_bps":%s,"rx_pps":%s,"tx_pps":%s,"speed_mbps":%s,"rx_drops":%s,"rx_errors":%s,"rx_overruns":%s,"tx_drops":%s,"tx_errors":%s,"tx_overruns":%s}' \
      "${comma}" \
      "$(json_escape "${tracked_net_ifaces[i]}")" \
      "${rx_bps}" \
      "${tx_bps}" \
      "${rx_pps}" \
      "${tx_pps}" \
      "${speed_mbps}" \
      "${rx_drops}" \
      "${rx_errors}" \
      "${rx_overruns}" \
      "${tx_drops}" \
      "${tx_errors}" \
      "${tx_overruns}"
    comma=','
    prev_net_rx[i]="${current_rx}"
    prev_net_tx[i]="${current_tx}"
    prev_net_rx_packets[i]="${current_rx_packets}"
    prev_net_tx_packets[i]="${current_tx_packets}"
    prev_net_rx_drops[i]="${current_rx_drops}"
    prev_net_rx_errors[i]="${current_rx_errors}"
    prev_net_rx_overruns[i]="${current_rx_overruns}"
    prev_net_tx_drops[i]="${current_tx_drops}"
    prev_net_tx_errors[i]="${current_tx_errors}"
    prev_net_tx_overruns[i]="${current_tx_overruns}"
  done
  printf ']'
}

read_tcp_counter_sample() {
  awk '
    /^Tcp:/ {
      if (!seen) {
        for (i = 1; i <= NF; i++) {
          header[$i] = i
        }
        seen = 1
        next
      }
      retrans = (header["RetransSegs"] ? $(header["RetransSegs"]) : 0)
      resets = 0
      if (header["EstabResets"]) {
        resets += $(header["EstabResets"])
      }
      if (header["OutRsts"]) {
        resets += $(header["OutRsts"])
      }
      printf "%s|%s\n", retrans + 0, resets + 0
      emitted = 1
      exit
    }
    END {
      if (!emitted) {
        printf "%s|%s\n", 0, 0
      }
    }
  ' /proc/net/snmp 2>/dev/null
}

boolish_active() {
  local value
  value="$(trim "${1:-}")"
  value="$(printf '%s' "${value}" | tr '[:upper:]' '[:lower:]')"
  value="${value// /}"
  case "${value}" in
    ''|n/a|-1|0|false|no|disabled|inactive|notactive)
      return 1
      ;;
  esac
  return 0
}

summarize_gpu_throttle_reasons() {
  local reasons=''
  local add_reason

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
  done <<< "${proc_output}"
  printf ']'
}

build_gpu_json() {
  local idx uuid name util mem_util mem_used mem_total temp power_draw power_limit fan sm_clock sm_clock_max mem_clock mem_clock_max pstate
  local gpu_combined_output=''
  local gpu_output=''
  local gpu_extra_output=''
  local gpu_throttle_output=''
  local attempt
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
    done <<< "${gpu_combined_output}"
    printf ']'
    return
  fi

  for attempt in 1 2; do
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
    done <<< "${gpu_extra_output}"
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
    done <<< "${gpu_throttle_output}"
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
  done <<< "${gpu_output}"
  printf ']'
}

remote_name="$(hostname)"
remote_cpu_cores="$(nproc 2>/dev/null || getconf _NPROCESSORS_ONLN 2>/dev/null || printf '0')"
remote_cpu_name="$(read_cpu_model_name)"
root_source="$(df -P / | awk 'NR==2 {print $1}')"
root_device=''
page_size_bytes="$(getconf PAGESIZE 2>/dev/null || printf '4096')"
nvidia_smi_path="$(command -v nvidia-smi 2>/dev/null || true)"
filesystem_refresh_samples="$(refresh_samples_for_seconds "${filesystem_refresh_seconds}")"
sample_index=0
root_usage_cache=''
filesystems_json_cache=''

case "${remote_cpu_cores}" in
  ''|*[!0-9]*)
    remote_cpu_cores='0'
    ;;
esac

case "${root_source}" in
  /dev/*)
    root_device="${root_source#/dev/}"
    ;;
esac

cpu_labels=()
cpu_idle=()
cpu_total=()
cpu_user=()
cpu_system=()
cpu_iowait=()
cpu_steal=()
prev_cpu_labels=()
prev_cpu_idle=()
prev_cpu_total=()
prev_cpu_user=()
prev_cpu_system=()
prev_cpu_iowait=()
prev_cpu_steal=()
tracked_net_ifaces=()
prev_net_rx=()
prev_net_tx=()
prev_net_rx_packets=()
prev_net_tx_packets=()
prev_net_rx_drops=()
prev_net_rx_errors=()
prev_net_rx_overruns=()
prev_net_tx_drops=()
prev_net_tx_errors=()
prev_net_tx_overruns=()
prev_disk_sectors_read='-1'
prev_disk_sectors_written='-1'
prev_disk_io_ms='-1'
prev_disk_reads_completed='-1'
prev_disk_reads_merged='-1'
prev_disk_read_ms='-1'
prev_disk_writes_completed='-1'
prev_disk_writes_merged='-1'
prev_disk_write_ms='-1'
prev_disk_in_flight='-1'
prev_disk_weighted_ms='-1'
prev_swap_in_pages='-1'
prev_swap_out_pages='-1'
prev_tcp_retrans='-1'
prev_tcp_resets='-1'

read_cpu_snapshot
prev_cpu_labels=("${cpu_labels[@]}")
prev_cpu_idle=("${cpu_idle[@]}")
prev_cpu_total=("${cpu_total[@]}")
prev_cpu_user=("${cpu_user[@]}")
prev_cpu_system=("${cpu_system[@]}")
prev_cpu_iowait=("${cpu_iowait[@]}")
prev_cpu_steal=("${cpu_steal[@]}")
discover_net_ifaces

for ((i = 0; i < ${#tracked_net_ifaces[@]}; i++)); do
  IFS='|' read -r prev_net_rx[i] prev_net_tx[i] prev_net_rx_packets[i] prev_net_tx_packets[i] prev_net_rx_drops[i] prev_net_rx_errors[i] prev_net_rx_overruns[i] prev_net_tx_drops[i] prev_net_tx_errors[i] prev_net_tx_overruns[i] < <(read_net_sample "${tracked_net_ifaces[i]}")
done

IFS='|' read -r prev_disk_sectors_read prev_disk_sectors_written prev_disk_io_ms prev_disk_reads_completed prev_disk_reads_merged prev_disk_read_ms prev_disk_writes_completed prev_disk_writes_merged prev_disk_write_ms prev_disk_in_flight prev_disk_weighted_ms < <(read_disk_sample "${root_device}")
IFS='|' read -r prev_swap_in_pages prev_swap_out_pages < <(read_swap_io_sample)
IFS='|' read -r prev_tcp_retrans prev_tcp_resets < <(read_tcp_counter_sample)
prev_sample_ns="$(now_ns)"
sample_elapsed_ms=$((interval * 1000))
next_tick_ns="${prev_sample_ns}"

while true; do
  current_ns="$(now_ns)"
  if [ "${next_tick_ns}" -gt "${current_ns}" ]; then
    sleep_ns=$((next_tick_ns - current_ns))
    sleep "$(awk -v ns="${sleep_ns}" 'BEGIN { printf "%.3f\n", ns / 1000000000 }')"
  fi
  next_tick_ns=$((next_tick_ns + interval_ns))
  sample_now_ns="$(now_ns)"
  sample_elapsed_ms=$(((sample_now_ns - prev_sample_ns) / 1000000))
  if [ "${sample_elapsed_ms}" -lt 1 ]; then
    sample_elapsed_ms=$((interval * 1000))
  fi
  sample_index=$((sample_index + 1))

  read_cpu_snapshot
  cpu_pct=0
  cpu_user_pct=-1
  cpu_system_pct=-1
  cpu_iowait_pct=-1
  cpu_steal_pct=-1
  swap_in_bps='-1'
  swap_out_bps='-1'
  tcp_retrans_per_sec='-1'
  tcp_resets_per_sec='-1'

  if [ "${#cpu_labels[@]}" -gt 0 ] && [ "${#prev_cpu_total[@]}" -gt 0 ] && [ "${cpu_labels[0]}" = "${prev_cpu_labels[0]}" ]; then
    diff_idle=$((cpu_idle[0] - prev_cpu_idle[0]))
    diff_total=$((cpu_total[0] - prev_cpu_total[0]))
    if [ "${diff_total}" -gt 0 ]; then
      cpu_pct=$(((100 * (diff_total - diff_idle)) / diff_total))
      cpu_user_pct=$((((cpu_user[0] - prev_cpu_user[0]) * 100) / diff_total))
      cpu_system_pct=$((((cpu_system[0] - prev_cpu_system[0]) * 100) / diff_total))
      cpu_iowait_pct=$((((cpu_iowait[0] - prev_cpu_iowait[0]) * 100) / diff_total))
      cpu_steal_pct=$((((cpu_steal[0] - prev_cpu_steal[0]) * 100) / diff_total))
    fi
  fi

  IFS='|' read -r ram_used ram_total ram_available ram_free ram_cache ram_buffers ram_reclaimable ram_shared < <(read_ram_stats)
  IFS='|' read -r cpu_freq_mhz cpu_max_freq_mhz < <(read_cpu_freq_stats)
  cpu_temp_c="$(read_cpu_temp_c)"
  IFS='|' read -r cpu_pressure_some cpu_pressure_full < <(read_pressure_avg10 /proc/pressure/cpu)
  IFS='|' read -r mem_pressure_some mem_pressure_full < <(read_pressure_avg10 /proc/pressure/memory)
  wsl_host_metrics_json="$(read_wsl_windows_host_metrics_json)"
  apply_wsl_host_metrics
  IFS='|' read -r swap_free_kib swap_total_kib < <(read_swap_stats)
  IFS='|' read -r swap_in_pages swap_out_pages < <(read_swap_io_sample)
  IFS='|' read -r tcp_retrans_counter tcp_resets_counter < <(read_tcp_counter_sample)
  read -r load1 load5 load15 _ < /proc/loadavg
  uptime_s="$(awk '{printf "%d\n", $1}' /proc/uptime)"
  epoch_now="$(date +%s)"
  stamp_now="$(date '+%F %T')"

  if [ "${swap_in_pages}" -ge 0 ] && [ "${prev_swap_in_pages}" -ge 0 ]; then
    swap_in_bps=$((((swap_in_pages - prev_swap_in_pages) * page_size_bytes * 1000) / sample_elapsed_ms))
    if [ "${swap_in_bps}" -lt 0 ]; then
      swap_in_bps=0
    fi
  fi

  if [ "${swap_out_pages}" -ge 0 ] && [ "${prev_swap_out_pages}" -ge 0 ]; then
    swap_out_bps=$((((swap_out_pages - prev_swap_out_pages) * page_size_bytes * 1000) / sample_elapsed_ms))
    if [ "${swap_out_bps}" -lt 0 ]; then
      swap_out_bps=0
    fi
  fi

  if [ "${tcp_retrans_counter}" -ge 0 ] && [ "${prev_tcp_retrans}" -ge 0 ]; then
    tcp_retrans_per_sec=$((((tcp_retrans_counter - prev_tcp_retrans) * 1000) / sample_elapsed_ms))
    if [ "${tcp_retrans_per_sec}" -lt 0 ]; then
      tcp_retrans_per_sec=0
    fi
  fi

  if [ "${tcp_resets_counter}" -ge 0 ] && [ "${prev_tcp_resets}" -ge 0 ]; then
    tcp_resets_per_sec=$((((tcp_resets_counter - prev_tcp_resets) * 1000) / sample_elapsed_ms))
    if [ "${tcp_resets_per_sec}" -lt 0 ]; then
      tcp_resets_per_sec=0
    fi
  fi

  disk_json="$(build_disk_json)"
  net_json="$(build_net_json)"
  filesystems_json="$(build_filesystems_json)"
  cpu_core_json="$(build_cpu_core_json)"
  top_process_json="$(build_top_process_json)"
  gpu_process_json="$(build_gpu_process_json)"
  gpu_json="$(build_gpu_json)"

  printf '{"version":1,"epoch":%s,"timestamp":"%s","remote":"%s","uptime_seconds":%s,"load1":%s,"load5":%s,"load15":%s,"cpu_cores":%s,"cpu_name":"%s","cpu_percent":%s,"cpu_user_percent":%s,"cpu_system_percent":%s,"cpu_iowait_percent":%s,"cpu_steal_percent":%s,"ram_used_mib":%s,"ram_total_mib":%s,"ram_available_mib":%s,"ram_free_mib":%s,"ram_cache_mib":%s,"ram_buffers_mib":%s,"ram_reclaimable_mib":%s,"ram_shared_mib":%s,"cpu_freq_mhz":%s,"cpu_max_freq_mhz":%s,"cpu_temp_c":%s,"cpu_pressure_some_avg10":%s,"cpu_pressure_full_avg10":%s,"mem_pressure_some_avg10":%s,"mem_pressure_full_avg10":%s,"swap":{"free_kib":%s,"total_kib":%s,"in_bps":%s,"out_bps":%s},"disk":%s,"net":%s,"filesystems":%s,"tcp_retrans_segs_per_sec":%s,"tcp_resets_per_sec":%s,"cpu_core_usage":%s,"top_processes":%s,"gpu_processes":%s,"gpus":%s}\n' \
    "${epoch_now}" \
    "$(json_escape "${stamp_now}")" \
    "$(json_escape "${remote_name}")" \
    "${uptime_s}" \
    "${load1}" \
    "${load5}" \
    "${load15}" \
    "${remote_cpu_cores}" \
    "$(json_escape "${remote_cpu_name}")" \
    "${cpu_pct}" \
    "${cpu_user_pct}" \
    "${cpu_system_pct}" \
    "${cpu_iowait_pct}" \
    "${cpu_steal_pct}" \
    "${ram_used}" \
    "${ram_total}" \
    "${ram_available}" \
    "${ram_free}" \
    "${ram_cache}" \
    "${ram_buffers}" \
    "${ram_reclaimable}" \
    "${ram_shared}" \
    "${cpu_freq_mhz}" \
    "${cpu_max_freq_mhz}" \
    "${cpu_temp_c}" \
    "${cpu_pressure_some}" \
    "${cpu_pressure_full}" \
    "${mem_pressure_some}" \
    "${mem_pressure_full}" \
    "${swap_free_kib}" \
    "${swap_total_kib}" \
    "${swap_in_bps}" \
    "${swap_out_bps}" \
    "${disk_json}" \
    "${net_json}" \
    "${filesystems_json}" \
    "${tcp_retrans_per_sec}" \
    "${tcp_resets_per_sec}" \
    "${cpu_core_json}" \
    "${top_process_json}" \
    "${gpu_process_json}" \
    "${gpu_json}"

  prev_cpu_labels=("${cpu_labels[@]}")
  prev_cpu_idle=("${cpu_idle[@]}")
  prev_cpu_total=("${cpu_total[@]}")
  prev_cpu_user=("${cpu_user[@]}")
  prev_cpu_system=("${cpu_system[@]}")
  prev_cpu_iowait=("${cpu_iowait[@]}")
  prev_cpu_steal=("${cpu_steal[@]}")
  prev_swap_in_pages="${swap_in_pages}"
  prev_swap_out_pages="${swap_out_pages}"
  prev_tcp_retrans="${tcp_retrans_counter}"
  prev_tcp_resets="${tcp_resets_counter}"
  prev_sample_ns="${sample_now_ns}"

  current_ns="$(now_ns)"
  if [ "${next_tick_ns}" -lt "${current_ns}" ]; then
    next_tick_ns="${current_ns}"
  fi
done
