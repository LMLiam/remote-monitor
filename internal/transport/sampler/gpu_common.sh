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

build_gpu_json() {
  local nvidia_json amd_json intel_json
  nvidia_json="$(build_nvidia_gpu_json)"
  amd_json="$(build_amd_gpu_json)"
  intel_json="$(build_intel_gpu_json)"

  combine_gpu_json_arrays "$(combine_gpu_json_arrays "${nvidia_json}" "${amd_json}")" "${intel_json}"
}
