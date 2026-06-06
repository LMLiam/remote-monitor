#!/usr/bin/env bash
set -euo pipefail

workflow="${1:-.github/workflows/build.yml}"

require_line() {
  local pattern="$1"
  local description="$2"

  if ! grep -Fq -- "${pattern}" "${workflow}"; then
    echo "build workflow missing ${description}: ${pattern}" >&2
    exit 1
  fi
}

reject_line() {
  local pattern="$1"
  local description="$2"

  if grep -Fq -- "${pattern}" "${workflow}"; then
    echo "build workflow still has ${description}: ${pattern}" >&2
    exit 1
  fi
}

require_line "strategy:" "matrix strategy"
require_line "matrix:" "matrix definition"
require_line "os: [ubuntu-latest, macos-latest]" "Linux and macOS runners"
require_line 'runs-on: ${{ matrix.os }}' "matrix runner selection"
require_line "go vet -tags=integration ./..." "cross-platform vet step"
require_line "go test -tags=integration ./cmd/... ./internal/..." "cross-platform Go test step"
require_line "go test -tags=integration ./tests/e2e" "SSH e2e test step"
require_line "if: \${{ matrix.os == 'ubuntu-latest' }}" "Linux-only SSH e2e gate"
require_line "go build -o remote-monitor ./cmd/remote-monitor" "cross-platform build step"

reject_line "go test -tags=integration ./..." "ungated all-package integration test"
