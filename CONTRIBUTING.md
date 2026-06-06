# Contributing

Thanks for helping improve `remote-monitor`.

## Before You Start

- Open an issue first for larger features, sampler changes, or behavior that affects SSH command execution.
- Check existing issues and pull requests to avoid duplicate work.
- Keep changes focused; small pull requests are easier to review and safer to merge.
- Do not include generated binaries, local IDE files, SSH config, logs with hostnames, or terminal recordings with secrets.

## Development Setup

Use Go 1.26 or newer. The project intentionally uses native Go tooling and `golangci-lint`.

```sh
go mod download
make setup
```

## Checks

Run the full local check gate before opening a pull request:

```sh
make check
```

`make check` mirrors the CI checks in `.github/workflows/build.yml`: Go formatting,
shell formatting, ShellCheck, workflow helper script tests, `go vet`,
integration-tagged tests, `golangci-lint`, and build. The ShellCheck target uses
Docker to run CI's pinned image. The integration-tagged test run includes the SSH
end-to-end test, which self-skips when Docker is unavailable.

Individual targets such as `make fmt`, `make shfmt`, `make shellcheck`, `make scripts`,
`make test`, and `make lint` are available for targeted checks.

## Code Style

- Prefer clear package boundaries over large catch-all files.
- Keep sampler changes portable across ordinary Linux hosts.
- Preserve both supported themes: `aurora` and `basic`.
- Fix lint findings in code instead of weakening lint rules.
- Keep generated binaries, IDE state, and machine-local files out of commits.

## Pull Requests

- Use the pull request template.
- Use `verb(area): something` for pull request titles and commit subjects, for example `feat(metrics): add disk pressure summary`.
- The `main` branch ruleset requires pull requests, required checks, and squash merges. The required `Validate Commit Subjects` check enforces the same commit subject format before merge.
- Explain user-visible behavior changes and any portability assumptions.
- Add or update tests for parser, renderer, transport, config, and monitor behavior when practical.
- Include manual testing notes for terminal rendering or SSH behavior that is hard to cover with unit tests.

## Security

Please follow [SECURITY.md](SECURITY.md) for sensitive reports. Do not file public issues for vulnerabilities, leaked host details, or exploitable command construction problems.

## GitHub Actions Policy

The repository uses a selected-actions allow-list for GitHub Actions. GitHub-owned actions are allowed, and third-party actions are limited to the sources this project intentionally uses.

Approved GitHub-owned action sources:

- `actions/checkout`
- `actions/dependency-review-action`
- `actions/setup-go`
- `actions/upload-artifact`
- `github/codeql-action`

Approved third-party action sources:

- `golangci/golangci-lint-action`
- `ossf/scorecard-action`

Before adding a new action source, maintainers should open a pull request or issue that explains why the action is needed, who maintains it, what permissions it needs, and whether a shell command or existing GitHub-owned action would be simpler. New `uses:` entries must be pinned to a full commit SHA with a nearby version comment. The repository Actions allow-list must be updated before the workflow change merges.

Keep the default workflow token permission at read-only. Workflows should declare `permissions: contents: read` or `permissions: {}` by default, and grant write permissions only at the job level when that job needs them.
