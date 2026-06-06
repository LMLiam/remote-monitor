# AGENTS.md

Guidance for AI agents and contributors working in this repository.

## Project

`remote-monitor` is a Go + Bubble Tea TUI that streams a small Bash sampler over SSH to a remote
Linux host and renders CPU / memory / disk / network / process / GPU metrics locally. There is no
remote agent or daemon. Module path: `github.com/lmliam/remote-monitor`.

## Layout

- `cmd/remote-monitor` — entrypoint.
- `internal/config` — CLI flags, environment, TOML profiles.
- `internal/transport` — SSH execution, reconnect/stream loop, embedded sampler.
- `internal/transport/sampler/` — Bash sampler **source modules** assembled into `internal/transport/sampler.sh`.
- `internal/parser` — sampler JSON → `core.Sample`.
- `internal/core` — shared model (`Sample`, `Config`, `AppState`).
- `internal/metrics` — derived metrics, rolling history, aggregation.
- `internal/render`, `internal/render/ansi`, `internal/render/banner` — TUI / text / layout rendering.
- `internal/output` — JSONL encoder.
- `internal/monitor` — app wiring, output-mode selection, Bubble Tea program.

## Documentation maintenance

Before opening a PR, check whether the change affects an architecture-documented component:
transport, sampler, parser, core model, output modes, renderer, or test strategy. If it changes
architectural boundaries, data flow, ownership, or testing strategy, update `docs/architecture.md`
in the same PR. Do not edit the architecture guide for unrelated changes.

## Setup

```sh
go mod download
bash .github/scripts/install-git-hooks.sh   # commit-msg hook validates commit subjects locally
```

## Local check gate (must be green before pushing — mirrors CI)

`make check` runs the full local gate. The raw commands are:

```sh
gofmt -l ./cmd ./internal ./tests           # must print nothing
go run mvdan.cc/sh/v3/cmd/shfmt@v3.13.1 -i 2 -ci -d .github/scripts internal/transport/sampler internal/transport/sampler.sh tests/e2e/ssh-target
docker run --rm -v "$PWD:/mnt" -w /mnt docker.io/koalaman/shellcheck:v0.11.0 -S warning -s bash .github/scripts/*.sh tests/e2e/ssh-target/*.sh internal/transport/sampler/assemble.sh internal/transport/sampler.sh
awk 'NF && $1 !~ /^#/ { print "internal/transport/sampler/" $0 }' internal/transport/sampler/manifest.txt | xargs docker run --rm -v "$PWD:/mnt" -w /mnt docker.io/koalaman/shellcheck:v0.11.0 -S warning -s bash -e SC2034,SC2154
bash .github/scripts/test-conventional-title.sh
bash .github/scripts/test-next-release-tag.sh
bash .github/scripts/test-verify-main-checks.sh
bash .github/scripts/test-build-workflow.sh
go vet -tags=integration ./...
go test -tags=integration ./...             # the SSH e2e test self-skips if Docker is absent
golangci-lint run --build-tags=integration  # use the version pinned in .github/workflows/build.yml
go build -o /tmp/remote-monitor ./cmd/remote-monitor
```

Never make checks pass by weakening them: no skipped or deleted tests, no removed assertions, no
blanket `//nolint`, no disabled linters, no relaxed thresholds. Use narrowly scoped, justified
suppressions only, and explain them in the PR.

## Sampler changes

The remote sampler is assembled from `internal/transport/sampler/*.sh` (order in `manifest.txt`).
After editing any module, regenerate and commit the embedded script:

```sh
bash internal/transport/sampler/assemble.sh   # or: go generate ./internal/transport
```

`go test ./internal/transport` verifies the embedded `sampler.sh` matches the modules and passes
`bash -n`.

## Conventions

- **Commits and PR titles** are conventional: `verb(scope): summary`, lowercase verb (e.g.
  `ci(sampler): ...`). Both are CI-validated (`conventional-titles.yml`); the commit-msg hook
  enforces it locally.
- **Lint is strict**: ~40 golangci-lint linters including `exhaustruct`, `funlen`, `cyclop`, `gosec`,
  `paralleltest`, `testpackage`. Match existing style; new tests call `t.Parallel()`.
- **Tests-first**: add or extend tests for any behavior change; pure helpers get table-driven tests.
- **Issues / labels / milestones**: reuse the existing taxonomy; do not invent new labels or
  milestones. Issue titles follow the same conventional scheme.
- Keep generated binaries and machine-local files out of commits (see `.gitignore`).

## Working an issue (PR workflow)

1. `gh issue view <N> --comments`. Capture **every** acceptance criterion. If the issue is
   `status:blocked` or has an open "blocked by" dependency, stop and report instead of proceeding.
2. `git fetch origin`, then branch a dedicated worktree off fresh `origin/main`:
   `git worktree add -b <type>/<N>-<slug> ../rm-<N> origin/main`. Install the hooks.
3. Implement comprehensively and in scope. Capture adjacent findings as follow-up issues
   (`gh issue create`); do not scope-creep. Add tests. Regenerate the sampler if you touched it.
4. Pass the local check gate above.
5. `gh pr create` against `main`: conventional title; body following
   `.github/PULL_REQUEST_TEMPLATE.md` (what and why, how each acceptance criterion is met, the check
   commands you ran, follow-ups); include `Closes #<N>`; mirror the issue's `--label`s and
   `--milestone`; `--assignee @me`.
6. **Self-review — mandatory.** Run it as a separate phase against the
   **Self-review (required)** section below: review the full diff as a cold reviewer, post an inline
   comment for every finding, fix each, reply and resolve the thread, then re-review. Loop until a
   full rubric pass adds no new findings, all threads are resolved, and CI is green (or after 5
   rounds, reporting what remains). Finding nothing on round 1 means you reviewed too shallowly.
7. **Definition of Done** (all true): every acceptance criterion met; local gate and remote CI fully
   green; PR open with correct title/body/labels/milestone/assignee and `Closes #<N>`; you completed
   **at least one full Self-review pass over the final diff** and every thread you opened is resolved.
   Do not submit an approving review (you cannot approve your own PR). **Do not merge** — `main` is
   protected (PR + 1 approval + status checks + thread resolution required); leave merging to a human.
8. Report: the PR URL, a change summary, an acceptance-criteria checklist mapping each criterion to
   where it is satisfied, **every self-review finding and how it was resolved** (with comment/thread
   references), the final CI status, and anything deferred.

## Self-review (required)

Review as a **cold reviewer**: re-derive correctness from the diff alone and assume it is defective.
"No issues found" on the first pass means the review was too shallow — look again at tests and error
paths. Never submit an approving review on your own PR; use comments + thread resolution only.

**Review rubric** — walk the full diff (`git diff origin/main...HEAD`) file-by-file against each:

- Correctness: meets every acceptance criterion; off-by-one, nil/zero, ignored return values.
- Errors: every error checked and wrapped; none swallowed; context cancellation honored.
- Edge cases: empty / missing / unavailable inputs (sentinels are `-1` / `""`), large values, concurrency.
- Tests: a test exists for every new or changed branch; table-driven; `t.Parallel()`; clear messages.
- Security: no command/argument injection on anything reaching `ssh`/`exec` or the remote shell; no secrets in output or logs.
- Conventions: passes the full gate; no new `//nolint` without justification; `exhaustruct`/`funlen` respected; `gofmt` clean.
- Sampler: if a shell module changed, `sampler.sh` was regenerated; quoting is safe; `bash -n` is clean.
- Docs & scope: user-visible changes reflected in `README`; no debug leftovers or stray `TODO`s; no unrelated edits.

**Review mechanics (gh).** Inline comments need the head SHA + path + line; resolving a thread is a
GraphQL mutation:

```sh
PR=<n>
HEAD="$(gh pr view "$PR" --json headRefOid --jq .headRefOid)"

# 1) inline, line-anchored comment (add -F start_line=.. -f start_side=RIGHT for a range):
gh api repos/:owner/:repo/pulls/"$PR"/comments \
  -f commit_id="$HEAD" -f path='internal/foo.go' -F line=42 -f side=RIGHT -f body='finding ...'

# 2) list your review threads (ids + resolved state + location):
read OWNER REPO < <(gh repo view --json owner,name --jq '.owner.login+" "+.name')
gh api graphql -f owner="$OWNER" -f repo="$REPO" -F pr="$PR" -f query='
query($owner:String!,$repo:String!,$pr:Int!){repository(owner:$owner,name:$repo){
  pullRequest(number:$pr){reviewThreads(first:100){nodes{id isResolved
    comments(first:1){nodes{path line}}}}}}}'

# 3) reply with the fix, then resolve the thread:
gh api repos/:owner/:repo/pulls/"$PR"/comments/<comment_id>/replies -f body='fixed in <sha>'
gh api graphql -f tid='<threadId>' -f query='
mutation($tid:ID!){resolveReviewThread(input:{threadId:$tid}){thread{isResolved}}}'
```

**Exit criteria.** Stop only when a full rubric pass yields no new findings, every thread is
resolved, and all CI checks are green — or after 5 rounds, reporting what remains.

## GitHub access

Use the `gh` CLI for all GitHub interaction. Native issue dependencies: add a blocker with
`gh api --method POST repos/:owner/:repo/issues/<N>/dependencies/blocked_by -F issue_id=<DB_ID>`,
where `<DB_ID>` is the blocker's numeric database id from
`gh api repos/:owner/:repo/issues/<M> --jq .id` (use `-F`, not `-f`, so it is sent as an integer).
