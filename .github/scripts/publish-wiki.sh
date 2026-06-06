#!/usr/bin/env bash
set -euo pipefail

wiki_source_dir="${WIKI_SOURCE_DIR:-wiki}"
wiki_work_dir="${WIKI_WORK_DIR:-wiki-repo}"
wiki_remote_url="${WIKI_REMOTE_URL:-}"
wiki_commit_message="${WIKI_COMMIT_MESSAGE:-docs(wiki): sync repository wiki}"
askpass_file=""
clone_stderr=""

cleanup() {
  if [ -n "${askpass_file}" ]; then
    rm -f "${askpass_file}"
  fi
  if [ -n "${clone_stderr}" ]; then
    rm -f "${clone_stderr}"
  fi
}
trap cleanup EXIT

if [ ! -d "${wiki_source_dir}" ]; then
  echo "::error::Wiki source directory does not exist: ${wiki_source_dir}" >&2
  exit 1
fi

if [ -z "${wiki_remote_url}" ]; then
  if [ -z "${GITHUB_REPOSITORY:-}" ]; then
    echo "::error::Set GITHUB_REPOSITORY or WIKI_REMOTE_URL before publishing the wiki." >&2
    exit 1
  fi
  wiki_remote_url="https://github.com/${GITHUB_REPOSITORY}.wiki.git"
fi

if [ -e "${wiki_work_dir}" ]; then
  echo "::error::Wiki work directory already exists: ${wiki_work_dir}" >&2
  exit 1
fi

if [ -n "${WIKI_TOKEN:-}" ]; then
  askpass_file="$(mktemp "${TMPDIR:-/tmp}/remote-monitor-wiki-askpass.XXXXXX")"
  cat >"${askpass_file}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  *Username*) printf '%s\n' 'x-access-token' ;;
  *Password*) printf '%s\n' "${WIKI_TOKEN}" ;;
  *) printf '\n' ;;
esac
EOF
  chmod 0700 "${askpass_file}"
  export GIT_ASKPASS="${askpass_file}"
  export GIT_TERMINAL_PROMPT=0
fi

clone_stderr="$(mktemp "${TMPDIR:-/tmp}/remote-monitor-wiki-clone.XXXXXX")"
if ! git clone "${wiki_remote_url}" "${wiki_work_dir}" 2>"${clone_stderr}"; then
  {
    echo "::error::Unable to clone the GitHub wiki repository."
    echo "Bootstrap the wiki git backend before rerunning this workflow:"
    echo "  1. Open https://github.com/${GITHUB_REPOSITORY:-OWNER/REPO}/wiki"
    echo "  2. Create any starter page in the GitHub wiki UI."
    echo "  3. Rerun the Publish Wiki workflow."
    echo "If cloning still fails, verify WIKI_PUSH_TOKEN or GITHUB_TOKEN can push to the wiki repository."
    echo "git clone stderr:"
    sed 's/^/  /' "${clone_stderr}"
  } >&2
  exit 1
fi

# Replace published wiki files while preserving the clone metadata.
find "${wiki_work_dir}" -mindepth 1 -maxdepth 1 ! -name .git -exec rm -rf {} +
cp -a "${wiki_source_dir}/." "${wiki_work_dir}/"

cd "${wiki_work_dir}"
git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

# Avoid empty commits when repository content already matches the wiki.
if [ -z "$(git status --porcelain)" ]; then
  echo "Wiki is already up to date."
  exit 0
fi

git add -A
git commit -m "${wiki_commit_message}"
git push
