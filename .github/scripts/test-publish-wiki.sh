#!/usr/bin/env bash
set -euo pipefail

script=".github/scripts/publish-wiki.sh"

assert_file_exists() {
  local repo="$1"
  local path="$2"

  if ! git --git-dir="${repo}" cat-file -e "HEAD:${path}"; then
    echo "expected wiki repository to contain ${path}" >&2
    exit 1
  fi
}

assert_file_missing() {
  local repo="$1"
  local path="$2"

  if git --git-dir="${repo}" cat-file -e "HEAD:${path}" 2>/dev/null; then
    echo "expected wiki repository to remove ${path}" >&2
    exit 1
  fi
}

assert_contains() {
  local file="$1"
  local text="$2"

  if ! grep -Fq -- "${text}" "${file}"; then
    echo "expected ${file} to contain: ${text}" >&2
    echo "actual output:" >&2
    sed 's/^/  /' "${file}" >&2
    exit 1
  fi
}

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

wiki_source="${tmp_dir}/wiki"
wiki_seed="${tmp_dir}/seed"
wiki_repo="${tmp_dir}/remote-monitor.wiki.git"
wiki_work="${tmp_dir}/wiki-work"
noop_work="${tmp_dir}/wiki-work-noop"
failure_output="${tmp_dir}/failure-output.txt"
noop_output="${tmp_dir}/noop-output.txt"
sync_output="${tmp_dir}/sync-output.txt"

mkdir -p "${wiki_source}" "${wiki_seed}"
printf '# Remote Monitor Wiki\n' >"${wiki_source}/Home.md"
printf '# Configuration Reference\n' >"${wiki_source}/Configuration-Reference.md"

git init --bare "${wiki_repo}" >/dev/null
git -C "${wiki_seed}" init >/dev/null
git -C "${wiki_seed}" config user.name "test"
git -C "${wiki_seed}" config user.email "test@example.com"
printf '# Old Page\n' >"${wiki_seed}/Old.md"
git -C "${wiki_seed}" add Old.md
git -C "${wiki_seed}" commit -m "seed wiki" >/dev/null
git -C "${wiki_seed}" branch -M main
git -C "${wiki_seed}" remote add origin "${wiki_repo}"
git -C "${wiki_seed}" push origin main >/dev/null 2>&1
git --git-dir="${wiki_repo}" symbolic-ref HEAD refs/heads/main

if ! WIKI_REMOTE_URL="${wiki_repo}" \
  WIKI_SOURCE_DIR="${wiki_source}" \
  WIKI_WORK_DIR="${wiki_work}" \
  bash "${script}" >"${sync_output}" 2>&1; then
  sed 's/^/  /' "${sync_output}" >&2
  exit 1
fi

assert_file_exists "${wiki_repo}" "Home.md"
assert_file_exists "${wiki_repo}" "Configuration-Reference.md"
assert_file_missing "${wiki_repo}" "Old.md"

commit_count_before="$(git --git-dir="${wiki_repo}" rev-list --count HEAD)"
if ! WIKI_REMOTE_URL="${wiki_repo}" \
  WIKI_SOURCE_DIR="${wiki_source}" \
  WIKI_WORK_DIR="${noop_work}" \
  bash "${script}" >"${noop_output}" 2>&1; then
  sed 's/^/  /' "${noop_output}" >&2
  exit 1
fi
commit_count_after="$(git --git-dir="${wiki_repo}" rev-list --count HEAD)"

if [ "${commit_count_before}" != "${commit_count_after}" ]; then
  echo "expected no-op sync to avoid creating a commit" >&2
  exit 1
fi
assert_contains "${noop_output}" "Wiki is already up to date."

if GITHUB_REPOSITORY="LMLiam/remote-monitor" \
  WIKI_REMOTE_URL="${tmp_dir}/missing.wiki.git" \
  WIKI_SOURCE_DIR="${wiki_source}" \
  WIKI_WORK_DIR="${tmp_dir}/missing-work" \
  bash "${script}" >"${failure_output}" 2>&1; then
  echo "expected clone failure for missing wiki repository" >&2
  exit 1
fi

assert_contains "${failure_output}" "https://github.com/LMLiam/remote-monitor/wiki"
assert_contains "${failure_output}" "WIKI_PUSH_TOKEN or GITHUB_TOKEN"
