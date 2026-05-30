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
bash .github/scripts/install-git-hooks.sh
```

## Checks

Run these before opening a pull request:

```sh
unformatted="$(gofmt -l ./cmd ./internal)"
test -z "$unformatted" || { echo "$unformatted"; exit 1; }
go vet ./...
go test ./...
golangci-lint run
go build -o remote-monitor ./cmd/remote-monitor
```

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
