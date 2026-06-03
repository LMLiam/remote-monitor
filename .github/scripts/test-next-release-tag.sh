#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
script="${script_dir}/next-release-tag.sh"

assert_next_tag() {
  local bump="$1"
  local want="$2"
  shift 2

  local got
  got="$("${script}" "${bump}" "$@")"
  if [ "${got}" != "${want}" ]; then
    echo "next tag for ${bump} with tags [$*] = ${got}, want ${want}" >&2
    exit 1
  fi
}

assert_next_tag patch v0.1.6 v0.1.5 v0.1.4
assert_next_tag minor v0.2.0 v0.1.5 v0.1.4
assert_next_tag major v1.0.0 v0.1.5 v0.1.4
assert_next_tag patch v0.0.1
assert_next_tag minor v0.1.0
assert_next_tag major v1.0.0
assert_next_tag patch v2.0.1 v1.99.99 v2.0.0 v1.100.0
assert_next_tag minor v2.1.0 v2.0.9 v1.99.99
assert_next_tag patch v1.10.1 v1.9.99 v1.10.0 not-a-version v1.2

if "${script}" feature v0.1.5 >/tmp/next-release-tag-invalid.out 2>&1; then
  echo "invalid bump unexpectedly succeeded" >&2
  exit 1
fi

if ! grep -q "bump must be one of: patch, minor, major" /tmp/next-release-tag-invalid.out; then
  echo "invalid bump error message was not helpful" >&2
  cat /tmp/next-release-tag-invalid.out >&2
  exit 1
fi
