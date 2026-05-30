# Contributing

Thanks for helping improve `remote-monitor`.

## Development Setup

Use Go 1.26 or newer. The project intentionally uses native Go tooling and `golangci-lint`.

```sh
go mod download
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
