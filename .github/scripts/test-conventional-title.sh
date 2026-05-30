#!/usr/bin/env bash
set -euo pipefail

validator="${1:-.github/scripts/validate-conventional-title.sh}"
commit_msg_hook="${2:-.github/hooks/commit-msg}"

expect_pass() {
  local title="$1"

  bash "$validator" "$title" >/dev/null
}

expect_fail() {
  local title="$1"

  if bash "$validator" "$title" >/dev/null 2>&1; then
    printf 'expected failure for: %s\n' "$title" >&2
    exit 1
  fi
}

expect_pass 'feat(core): add remote monitor app'
expect_pass 'fix(ssh): reconnect after sampler exits'
expect_pass 'docs(open-source): document security policy'
expect_pass 'ci(github-actions): validate pull request titles'
expect_pass 'chore(deps): update Go modules'

expect_fail 'feat: missing area'
expect_fail 'Feat(core): uppercase verb'
expect_fail 'feat(Core): uppercase area'
expect_fail 'feat(core): '
expect_fail 'feat(core):  extra space'
expect_fail 'feat(core) missing colon'
expect_fail 'feat(core)!: breaking marker is not part of the format'
expect_fail '[Bug]: old issue template prefix'

printf '%s\n' \
  'feat(core): add remote monitor app' \
  'fix(ssh): reconnect after sampler exits' |
  bash "$validator" --stdin >/dev/null

if printf '%s\n' \
  'feat(core): add remote monitor app' \
  'bad title' |
  bash "$validator" --stdin >/dev/null 2>&1; then
  printf 'expected stdin validation to reject a bad title\n' >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

printf '%s\n\nbody text\n' 'feat(git): enforce commit subjects' > "$tmpdir/good-message"
bash "$commit_msg_hook" "$tmpdir/good-message" >/dev/null

printf '%s\n\nbody text\n' 'feat: missing area' > "$tmpdir/bad-message"
if bash "$commit_msg_hook" "$tmpdir/bad-message" >/dev/null 2>&1; then
  printf 'expected commit-msg hook to reject invalid subject\n' >&2
  exit 1
fi
