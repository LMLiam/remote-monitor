#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
script="${script_dir}/verify-main-checks.sh"
sha="abc123"

success_runs='[
  {"headSha":"abc123","workflowName":"Build","status":"completed","conclusion":"success"},
  {"headSha":"abc123","workflowName":"CodeQL","status":"completed","conclusion":"success"},
  {"headSha":"abc123","workflowName":"Conventional Titles","status":"completed","conclusion":"success"}
]'

missing_runs='[
  {"headSha":"abc123","workflowName":"Build","status":"completed","conclusion":"success"},
  {"headSha":"abc123","workflowName":"CodeQL","status":"completed","conclusion":"success"}
]'

failed_runs='[
  {"headSha":"abc123","workflowName":"Build","status":"completed","conclusion":"success"},
  {"headSha":"abc123","workflowName":"CodeQL","status":"completed","conclusion":"failure"},
  {"headSha":"abc123","workflowName":"Conventional Titles","status":"completed","conclusion":"success"}
]'

pending_runs='[
  {"headSha":"abc123","workflowName":"Build","status":"completed","conclusion":"success"},
  {"headSha":"abc123","workflowName":"CodeQL","status":"in_progress","conclusion":null},
  {"headSha":"abc123","workflowName":"Conventional Titles","status":"completed","conclusion":"success"}
]'

wrong_sha_runs='[
  {"headSha":"def456","workflowName":"Build","status":"completed","conclusion":"success"},
  {"headSha":"def456","workflowName":"CodeQL","status":"completed","conclusion":"success"},
  {"headSha":"def456","workflowName":"Conventional Titles","status":"completed","conclusion":"success"}
]'

RELEASE_RUNS_JSON="${success_runs}" "${script}" "${sha}" Build CodeQL "Conventional Titles"

if RELEASE_RUNS_JSON="${missing_runs}" "${script}" "${sha}" Build CodeQL "Conventional Titles" >/tmp/verify-main-checks-missing.out 2>&1; then
  echo "missing workflow unexpectedly succeeded" >&2
  exit 1
fi
if ! grep -q "required workflow did not complete successfully: Conventional Titles" /tmp/verify-main-checks-missing.out; then
  echo "missing workflow error was not helpful" >&2
  cat /tmp/verify-main-checks-missing.out >&2
  exit 1
fi

if RELEASE_RUNS_JSON="${failed_runs}" "${script}" "${sha}" Build CodeQL "Conventional Titles" >/tmp/verify-main-checks-failed.out 2>&1; then
  echo "failed workflow unexpectedly succeeded" >&2
  exit 1
fi
if ! grep -q "required workflow did not complete successfully: CodeQL" /tmp/verify-main-checks-failed.out; then
  echo "failed workflow error was not helpful" >&2
  cat /tmp/verify-main-checks-failed.out >&2
  exit 1
fi

if RELEASE_RUNS_JSON="${pending_runs}" "${script}" "${sha}" Build CodeQL "Conventional Titles" >/tmp/verify-main-checks-pending.out 2>&1; then
  echo "pending workflow unexpectedly succeeded" >&2
  exit 1
fi
if ! grep -q "required workflow did not complete successfully: CodeQL" /tmp/verify-main-checks-pending.out; then
  echo "pending workflow error was not helpful" >&2
  cat /tmp/verify-main-checks-pending.out >&2
  exit 1
fi

if RELEASE_RUNS_JSON="${wrong_sha_runs}" "${script}" "${sha}" Build CodeQL "Conventional Titles" >/tmp/verify-main-checks-sha.out 2>&1; then
  echo "wrong sha workflow unexpectedly succeeded" >&2
  exit 1
fi
if ! grep -q "required workflow did not complete successfully: Build" /tmp/verify-main-checks-sha.out; then
  echo "wrong sha error was not helpful" >&2
  cat /tmp/verify-main-checks-sha.out >&2
  exit 1
fi
