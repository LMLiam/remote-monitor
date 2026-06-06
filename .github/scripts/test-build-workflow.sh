#!/usr/bin/env bash
set -euo pipefail

workflow="${1:-.github/workflows/build.yml}"

extract_job() {
  local job="$1"

  awk -v job="${job}" '
    $0 == "  " job ":" {
      in_job = 1
      print
      next
    }
    in_job && /^  [[:alnum:]_-]+:/ {
      exit
    }
    in_job {
      print
    }
  ' "${workflow}"
}

require_line() {
  local pattern="$1"
  local description="$2"

  if ! grep -Fq -- "${pattern}" "${workflow}"; then
    echo "build workflow missing ${description}: ${pattern}" >&2
    exit 1
  fi
}

require_text() {
  local text="$1"
  local pattern="$2"
  local description="$3"

  if ! grep -Fq -- "${pattern}" <<<"${text}"; then
    echo "build workflow missing ${description}: ${pattern}" >&2
    exit 1
  fi
}

reject_text() {
  local text="$1"
  local pattern="$2"
  local description="$3"

  if grep -Fq -- "${pattern}" <<<"${text}"; then
    echo "build workflow still has ${description}: ${pattern}" >&2
    exit 1
  fi
}

go_job="$(extract_job "go")"
if [ -z "${go_job}" ]; then
  echo "build workflow missing go job" >&2
  exit 1
fi

require_line "bash .github/scripts/test-build-workflow.sh" "workflow verifier invocation"

require_text "${go_job}" "strategy:" "go job matrix strategy"
require_text "${go_job}" "matrix:" "go job matrix definition"
require_text "${go_job}" "ubuntu-latest" "Linux runner in go job matrix"
require_text "${go_job}" "macos-latest" "macOS runner in go job matrix"
require_text "${go_job}" 'runs-on: ${{ matrix.os }}' "matrix runner selection"
require_text "${go_job}" "go vet -tags=integration ./..." "cross-platform vet step"
require_text "${go_job}" "go test -tags=integration ./cmd/... ./internal/..." "cross-platform Go test step"
require_text "${go_job}" "go test -tags=integration ./tests/e2e" "SSH e2e test step"
require_text "${go_job}" "if: \${{ matrix.os == 'ubuntu-latest' }}" "Linux-only SSH e2e gate"
require_text "${go_job}" "go build -o remote-monitor ./cmd/remote-monitor" "cross-platform build step"

reject_text "${go_job}" "go test -tags=integration ./..." "ungated all-package integration test"
