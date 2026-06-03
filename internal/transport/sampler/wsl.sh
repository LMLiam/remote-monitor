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
