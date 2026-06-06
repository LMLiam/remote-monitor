#!/usr/bin/env bash
set -euo pipefail

workflow="${1:-.github/workflows/build.yml}"

all_workflows=(
  .github/workflows/build.yml
  .github/workflows/codeql.yml
  .github/workflows/conventional-titles.yml
  .github/workflows/dependency-review.yml
  .github/workflows/release.yml
  .github/workflows/scorecard.yml
)

extract_job() {
  local job="$1"
  local source_workflow="${2:-${workflow}}"

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
  ' "${source_workflow}"
}

extract_job_header() {
  local job="$1"
  local source_workflow="${2:-${workflow}}"

  extract_job "${job}" "${source_workflow}" | awk '
    /^    steps:/ {
      exit
    }
    {
      print
    }
  '
}

extract_top_level_block() {
  local block="$1"
  local source_workflow="$2"

  awk -v block="${block}" '
    $0 == block ":" {
      in_block = 1
      print
      next
    }
    in_block && $0 ~ /^[A-Za-z0-9_-]+:/ {
      exit
    }
    in_block {
      print
    }
  ' "${source_workflow}"
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

require_workflow_concurrency() {
  local source_workflow="$1"
  local concurrency_block

  concurrency_block="$(extract_top_level_block concurrency "${source_workflow}")"
  if [ -z "${concurrency_block}" ]; then
    echo "${source_workflow} missing top-level concurrency block: concurrency:" >&2
    exit 1
  fi
  require_text "${concurrency_block}" 'group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}' "stable concurrency group"
  require_text "${concurrency_block}" "cancel-in-progress: true" "superseded-run cancellation"
}

require_job_timeout() {
  local source_workflow="$1"
  local job="$2"
  local job_header

  job_header="$(extract_job_header "${job}" "${source_workflow}")"
  if [ -z "${job_header}" ]; then
    echo "${source_workflow} missing ${job} job" >&2
    exit 1
  fi
  require_text "${job_header}" "timeout-minutes:" "${job} job timeout"
}

go_job="$(extract_job "go")"
if [ -z "${go_job}" ]; then
  echo "build workflow missing go job" >&2
  exit 1
fi

require_line "bash .github/scripts/test-build-workflow.sh" "workflow verifier invocation"

for source_workflow in "${all_workflows[@]}"; do
  require_workflow_concurrency "${source_workflow}"
done

require_job_timeout .github/workflows/build.yml tooling
require_job_timeout .github/workflows/build.yml go
require_job_timeout .github/workflows/codeql.yml analyze
require_job_timeout .github/workflows/conventional-titles.yml pr-title
require_job_timeout .github/workflows/conventional-titles.yml commit-subjects
require_job_timeout .github/workflows/dependency-review.yml dependency-review
require_job_timeout .github/workflows/release.yml prepare-release
require_job_timeout .github/workflows/release.yml release
require_job_timeout .github/workflows/scorecard.yml scorecard

require_text "${go_job}" "strategy:" "go job matrix strategy"
require_text "${go_job}" "matrix:" "go job matrix definition"
require_text "${go_job}" "ubuntu-latest" "Linux runner in go job matrix"
require_text "${go_job}" "macos-latest" "macOS runner in go job matrix"
require_text "${go_job}" 'runs-on: ${{ matrix.os }}' "matrix runner selection"
require_text "${go_job}" "brew install bash" "macOS Bash install step"
require_text "${go_job}" "brew --prefix bash" "macOS Bash path lookup"
require_text "${go_job}" '>> "$GITHUB_PATH"' "macOS Bash PATH export"
require_text "${go_job}" "go vet -tags=integration ./..." "cross-platform vet step"
require_text "${go_job}" "go test -race -covermode=atomic" "race-enabled coverage test"
require_text "${go_job}" 'coverage_file="coverage-${{ matrix.os }}.out"' "per-runner coverage profile path"
require_text "${go_job}" '-coverprofile="${coverage_file}"' "coverage profile output"
require_text "${go_job}" "-tags=integration ./cmd/... ./internal/..." "cross-platform Go package targets"
require_text "${go_job}" "go tool cover -func=\"\${coverage_file}\"" "coverage total calculation"
require_text "${go_job}" '>> "$GITHUB_STEP_SUMMARY"' "coverage summary output"
require_text "${go_job}" "go test -tags=integration ./tests/e2e" "SSH e2e test step"
require_text "${go_job}" "if: \${{ matrix.os == 'ubuntu-latest' }}" "Linux-only SSH e2e gate"
require_text "${go_job}" "go build -o remote-monitor ./cmd/remote-monitor" "cross-platform build step"

reject_text "${go_job}" "go test -tags=integration ./..." "ungated all-package integration test"
