#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: next-release-tag.sh <patch|minor|major> [vX.Y.Z ...]" >&2
  exit 64
fi

bump="$1"
shift

case "${bump}" in
  patch | minor | major) ;;
  *)
    echo "bump must be one of: patch, minor, major" >&2
    exit 64
    ;;
esac

found=0
best_major=0
best_minor=0
best_patch=0

for tag in "$@"; do
  if [[ ! "${tag}" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    continue
  fi

  major=$((10#${BASH_REMATCH[1]}))
  minor=$((10#${BASH_REMATCH[2]}))
  patch=$((10#${BASH_REMATCH[3]}))

  if (( found == 0 ||
    major > best_major ||
    (major == best_major && minor > best_minor) ||
    (major == best_major && minor == best_minor && patch > best_patch) )); then
    found=1
    best_major="${major}"
    best_minor="${minor}"
    best_patch="${patch}"
  fi
done

case "${bump}" in
  patch)
    best_patch=$((best_patch + 1))
    ;;
  minor)
    best_minor=$((best_minor + 1))
    best_patch=0
    ;;
  major)
    best_major=$((best_major + 1))
    best_minor=0
    best_patch=0
    ;;
esac

printf 'v%d.%d.%d\n' "${best_major}" "${best_minor}" "${best_patch}"
