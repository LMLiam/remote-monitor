#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
manifest="${script_dir}/manifest.txt"
output="${script_dir}/../sampler.sh"
tmp="${output}.tmp"
first_module=1

cleanup_tmp() {
  rm -f "${tmp}"
}
trap cleanup_tmp EXIT

: >"${tmp}"
while IFS= read -r module || [ -n "${module}" ]; do
  case "${module}" in
    '' | '#'*)
      continue
      ;;
  esac

  module_path="${script_dir}/${module}"
  if [ ! -f "${module_path}" ]; then
    printf 'missing sampler module: %s\n' "${module}" >&2
    exit 1
  fi

  if [ "${first_module}" -eq 1 ]; then
    first_module=0
  else
    printf '\n' >>"${tmp}"
  fi

  cat "${module_path}" >>"${tmp}"
  if [ -s "${tmp}" ] && [ "$(tail -c 1 "${tmp}")" != "" ]; then
    printf '\n' >>"${tmp}"
  fi
done <"${manifest}"

mv "${tmp}" "${output}"
trap - EXIT
