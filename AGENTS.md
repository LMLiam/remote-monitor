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

## Setup

```sh
go mod download
bash .github/scripts/install-git-hooks.sh   # commit-msg hook validates commit subjects locally
```

## Local check gate (must be green before pushing — mirrors CI)

```sh
gofmt -l ./cmd ./internal ./tests           # must print nothing
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
6. **Self-review loop** — repeat until clean:
   - `gh pr checks <pr> --watch`; treat any red check as a blocking issue.
   - Leave **inline, line-anchored** review comments for every real issue via
     `gh api repos/:owner/:repo/pulls/<pr>/comments` (`commit_id` + `path` + `line`). No nitpicks:
     correctness, missing tests, edge cases, error handling, security, convention violations.
   - Fix each, push, reply on the thread, and resolve it
     (`gh api graphql` `resolveReviewThread`).
   - Re-review the new diff and re-check CI.
   - Loop guard: if a failure persists after a few honest fix attempts, stop and report it with
     evidence rather than thrashing.
7. **Definition of Done** (all true): every acceptance criterion met; local gate and remote CI fully
   green; PR open with correct title/body/labels/milestone/assignee and `Closes #<N>`; your review
   finds no substantive issues and every thread is resolved. **Do not merge** — `main` is protected
   (PR + 1 approval + status checks + thread resolution required); leave merging to a human.
8. Report: the PR URL, a change summary, an acceptance-criteria checklist mapping each criterion to
   where it is satisfied, the final CI status, and anything deferred.

## GitHub access

Use the `gh` CLI for all GitHub interaction. Native issue dependencies: add a blocker with
`gh api --method POST repos/:owner/:repo/issues/<N>/dependencies/blocked_by -F issue_id=<DB_ID>`,
where `<DB_ID>` is the blocker's numeric database id from
`gh api repos/:owner/:repo/issues/<M> --jq .id` (use `-F`, not `-f`, so it is sent as an integer).
