#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: verify-main-checks.sh <commit-sha> [workflow ...]" >&2
  exit 64
fi

commit_sha="$1"
shift

if [ "$#" -eq 0 ]; then
  set -- Build CodeQL "Conventional Titles"
fi

workflows=("$@")
fixture_mode=0
if [ -n "${RELEASE_RUNS_JSON:-}" ]; then
  fixture_mode=1
fi

fetch_runs_json() {
  if [ -n "${RELEASE_RUNS_JSON:-}" ]; then
    printf '%s\n' "${RELEASE_RUNS_JSON}"
    return
  fi

  if [ -z "${GITHUB_REPOSITORY:-}" ]; then
    echo "GITHUB_REPOSITORY is required when RELEASE_RUNS_JSON is not set" >&2
    exit 64
  fi

  gh run list \
    --repo "${GITHUB_REPOSITORY}" \
    --branch main \
    --commit "${commit_sha}" \
    --event push \
    --json headSha,workflowName,status,conclusion \
    --limit 100
}

workflow_succeeded() {
  local workflow="$1"
  jq -e --arg sha "${commit_sha}" --arg workflow "${workflow}" '
    any(.[]; .headSha == $sha and .workflowName == $workflow and .status == "completed" and .conclusion == "success")
  ' <<<"${runs_json}" >/dev/null
}

workflow_failed() {
  local workflow="$1"
  jq -e --arg sha "${commit_sha}" --arg workflow "${workflow}" '
    any(.[]; .headSha == $sha and .workflowName == $workflow and .status == "completed" and .conclusion != "success")
  ' <<<"${runs_json}" >/dev/null
}

print_workflow_status() {
  local workflow="$1"
  jq -r --arg sha "${commit_sha}" --arg workflow "${workflow}" '
    .[]
    | select(.headSha == $sha and .workflowName == $workflow)
    | "  found \(.workflowName): status=\(.status) conclusion=\(.conclusion)"
  ' <<<"${runs_json}" >&2
}

timeout_seconds="${CHECK_WAIT_TIMEOUT_SECONDS:-600}"
interval_seconds="${CHECK_WAIT_INTERVAL_SECONDS:-15}"
deadline=$(( $(date +%s) + timeout_seconds ))
runs_json=""

while true; do
  runs_json="$(fetch_runs_json)"

  incomplete=()
  failed=()

  for workflow in "${workflows[@]}"; do
    if workflow_succeeded "${workflow}"; then
      continue
    fi

    if workflow_failed "${workflow}"; then
      failed+=("${workflow}")
    else
      incomplete+=("${workflow}")
    fi
  done

  if [ "${#failed[@]}" -eq 0 ] && [ "${#incomplete[@]}" -eq 0 ]; then
    echo "required main checks succeeded for ${commit_sha}"
    exit 0
  fi

  if [ "${#failed[@]}" -gt 0 ]; then
    for workflow in "${failed[@]}"; do
      echo "required workflow did not complete successfully: ${workflow}" >&2
      print_workflow_status "${workflow}"
    done
    exit 1
  fi

  if [ "${fixture_mode}" -eq 1 ] || [ "$(date +%s)" -ge "${deadline}" ]; then
    for workflow in "${incomplete[@]}"; do
      echo "required workflow did not complete successfully: ${workflow}" >&2
      print_workflow_status "${workflow}"
    done
    exit 1
  fi

  echo "waiting for required main workflows for ${commit_sha}: ${incomplete[*]}"
  sleep "${interval_seconds}"
done
