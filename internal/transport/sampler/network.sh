declare -A prev_net_rx=()
declare -A prev_net_tx=()
declare -A prev_net_rx_packets=()
declare -A prev_net_tx_packets=()
declare -A prev_net_rx_drops=()
declare -A prev_net_rx_errors=()
declare -A prev_net_rx_overruns=()
declare -A prev_net_tx_drops=()
declare -A prev_net_tx_errors=()
declare -A prev_net_tx_overruns=()

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

reset_net_baseline_for_iface() {
  local iface="$1"

  prev_net_rx["${iface}"]='-1'
  prev_net_tx["${iface}"]='-1'
  prev_net_rx_packets["${iface}"]='-1'
  prev_net_tx_packets["${iface}"]='-1'
  prev_net_rx_drops["${iface}"]='-1'
  prev_net_rx_errors["${iface}"]='-1'
  prev_net_rx_overruns["${iface}"]='-1'
  prev_net_tx_drops["${iface}"]='-1'
  prev_net_tx_errors["${iface}"]='-1'
  prev_net_tx_overruns["${iface}"]='-1'
}

unset_net_baseline_for_iface() {
  local iface="$1"

  unset "prev_net_rx[${iface}]"
  unset "prev_net_tx[${iface}]"
  unset "prev_net_rx_packets[${iface}]"
  unset "prev_net_tx_packets[${iface}]"
  unset "prev_net_rx_drops[${iface}]"
  unset "prev_net_rx_errors[${iface}]"
  unset "prev_net_rx_overruns[${iface}]"
  unset "prev_net_tx_drops[${iface}]"
  unset "prev_net_tx_errors[${iface}]"
  unset "prev_net_tx_overruns[${iface}]"
}

prime_net_baselines() {
  local iface

  for iface in "${tracked_net_ifaces[@]}"; do
    IFS='|' read -r prev_net_rx["${iface}"] prev_net_tx["${iface}"] prev_net_rx_packets["${iface}"] prev_net_tx_packets["${iface}"] prev_net_rx_drops["${iface}"] prev_net_rx_errors["${iface}"] prev_net_rx_overruns["${iface}"] prev_net_tx_drops["${iface}"] prev_net_tx_errors["${iface}"] prev_net_tx_overruns["${iface}"] < <(read_net_sample "${iface}")
  done
}

refresh_tracked_net_ifaces() {
  local iface
  local old_ifaces=("${tracked_net_ifaces[@]}")
  declare -A was_tracked=()
  declare -A still_tracked=()

  for iface in "${old_ifaces[@]}"; do
    was_tracked["${iface}"]=1
  done

  discover_net_ifaces

  for iface in "${tracked_net_ifaces[@]}"; do
    still_tracked["${iface}"]=1
    if [ -z "${was_tracked[${iface}]:-}" ]; then
      reset_net_baseline_for_iface "${iface}"
    fi
  done

  for iface in "${!prev_net_rx[@]}"; do
    if [ -z "${still_tracked[${iface}]:-}" ]; then
      unset_net_baseline_for_iface "${iface}"
    fi
  done
}

network_refresh_sample_count() {
  local samples="${network_refresh_samples:-${filesystem_refresh_samples:-1}}"

  case "${samples}" in
    ''|*[!0-9]*|0)
      samples=1
      ;;
  esac

  printf '%s\n' "${samples}"
}

refresh_net_ifaces_if_needed() {
  local refresh_samples

  refresh_samples="$(network_refresh_sample_count)"
  if [ "${sample_index:-0}" -gt 0 ] && [ $((sample_index % refresh_samples)) -eq 0 ]; then
    refresh_tracked_net_ifaces
  fi
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
  local iface
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

  refresh_net_ifaces_if_needed
  elapsed_ms="$(elapsed_ms_or_default)"

  printf '['
  for iface in "${tracked_net_ifaces[@]}"; do
    IFS='|' read -r current_rx current_tx current_rx_packets current_tx_packets current_rx_drops current_rx_errors current_rx_overruns current_tx_drops current_tx_errors current_tx_overruns < <(read_net_sample "${iface}")
    speed_mbps="$(read_net_speed_mbps "${iface}")"

    if [ "${current_rx}" -ge 0 ] && [ "${prev_net_rx[${iface}]:--1}" -ge 0 ]; then
      rx_bps=$((((current_rx - prev_net_rx[${iface}]) * 1000) / elapsed_ms))
      if [ "${rx_bps}" -lt 0 ]; then
        rx_bps=0
      fi
    else
      rx_bps='-1'
    fi

    if [ "${current_tx}" -ge 0 ] && [ "${prev_net_tx[${iface}]:--1}" -ge 0 ]; then
      tx_bps=$((((current_tx - prev_net_tx[${iface}]) * 1000) / elapsed_ms))
      if [ "${tx_bps}" -lt 0 ]; then
        tx_bps=0
      fi
    else
      tx_bps='-1'
    fi

    if [ "${current_rx_packets}" -ge 0 ] && [ "${prev_net_rx_packets[${iface}]:--1}" -ge 0 ]; then
      rx_pps=$((((current_rx_packets - prev_net_rx_packets[${iface}]) * 1000) / elapsed_ms))
      if [ "${rx_pps}" -lt 0 ]; then
        rx_pps=0
      fi
    else
      rx_pps='-1'
    fi

    if [ "${current_tx_packets}" -ge 0 ] && [ "${prev_net_tx_packets[${iface}]:--1}" -ge 0 ]; then
      tx_pps=$((((current_tx_packets - prev_net_tx_packets[${iface}]) * 1000) / elapsed_ms))
      if [ "${tx_pps}" -lt 0 ]; then
        tx_pps=0
      fi
    else
      tx_pps='-1'
    fi

    if [ "${current_rx_drops}" -ge 0 ] && [ "${prev_net_rx_drops[${iface}]:--1}" -ge 0 ]; then
      rx_drops=$((current_rx_drops - prev_net_rx_drops[${iface}]))
      if [ "${rx_drops}" -lt 0 ]; then
        rx_drops=0
      fi
    else
      rx_drops='-1'
    fi

    if [ "${current_rx_errors}" -ge 0 ] && [ "${prev_net_rx_errors[${iface}]:--1}" -ge 0 ]; then
      rx_errors=$((current_rx_errors - prev_net_rx_errors[${iface}]))
      if [ "${rx_errors}" -lt 0 ]; then
        rx_errors=0
      fi
    else
      rx_errors='-1'
    fi

    if [ "${current_rx_overruns}" -ge 0 ] && [ "${prev_net_rx_overruns[${iface}]:--1}" -ge 0 ]; then
      rx_overruns=$((current_rx_overruns - prev_net_rx_overruns[${iface}]))
      if [ "${rx_overruns}" -lt 0 ]; then
        rx_overruns=0
      fi
    else
      rx_overruns='-1'
    fi

    if [ "${current_tx_drops}" -ge 0 ] && [ "${prev_net_tx_drops[${iface}]:--1}" -ge 0 ]; then
      tx_drops=$((current_tx_drops - prev_net_tx_drops[${iface}]))
      if [ "${tx_drops}" -lt 0 ]; then
        tx_drops=0
      fi
    else
      tx_drops='-1'
    fi

    if [ "${current_tx_errors}" -ge 0 ] && [ "${prev_net_tx_errors[${iface}]:--1}" -ge 0 ]; then
      tx_errors=$((current_tx_errors - prev_net_tx_errors[${iface}]))
      if [ "${tx_errors}" -lt 0 ]; then
        tx_errors=0
      fi
    else
      tx_errors='-1'
    fi

    if [ "${current_tx_overruns}" -ge 0 ] && [ "${prev_net_tx_overruns[${iface}]:--1}" -ge 0 ]; then
      tx_overruns=$((current_tx_overruns - prev_net_tx_overruns[${iface}]))
      if [ "${tx_overruns}" -lt 0 ]; then
        tx_overruns=0
      fi
    else
      tx_overruns='-1'
    fi

    printf '%s{"iface":"%s","rx_bps":%s,"tx_bps":%s,"rx_pps":%s,"tx_pps":%s,"speed_mbps":%s,"rx_drops":%s,"rx_errors":%s,"rx_overruns":%s,"tx_drops":%s,"tx_errors":%s,"tx_overruns":%s}' \
      "${comma}" \
      "$(json_escape "${iface}")" \
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
    prev_net_rx["${iface}"]="${current_rx}"
    prev_net_tx["${iface}"]="${current_tx}"
    prev_net_rx_packets["${iface}"]="${current_rx_packets}"
    prev_net_tx_packets["${iface}"]="${current_tx_packets}"
    prev_net_rx_drops["${iface}"]="${current_rx_drops}"
    prev_net_rx_errors["${iface}"]="${current_rx_errors}"
    prev_net_rx_overruns["${iface}"]="${current_rx_overruns}"
    prev_net_tx_drops["${iface}"]="${current_tx_drops}"
    prev_net_tx_errors["${iface}"]="${current_tx_errors}"
    prev_net_tx_overruns["${iface}"]="${current_tx_overruns}"
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
