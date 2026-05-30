#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

git config core.hooksPath .github/hooks

printf 'Configured Git hooks path: %s\n' "$(git config core.hooksPath)"
printf 'Commit messages are validated with .github/hooks/commit-msg\n'
