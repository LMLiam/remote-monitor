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
