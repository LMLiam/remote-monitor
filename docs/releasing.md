# Releasing

GitHub Releases are the source of truth for binary archives, checksums, and release notes. Homebrew packaging is tracked separately in issue #12.

## Versioning

Use manual SemVer tags. Start with `v0.1.0` unless maintainers choose a different initial version.

- Patch release: `v0.1.1`
- Compatible feature release: `v0.2.0`
- Breaking pre-1.0 change: bump the minor version
- Release candidate: `v0.2.0-rc.1`

The release workflow embeds the tag, commit, and build date into `remote-monitor --version`.

## Dry Run

Before the first public release, run the `Release` workflow manually from GitHub Actions. The manual path builds snapshot artifacts, skips publishing, and verifies the Linux `amd64` archive.

You can also run the same check locally:

```sh
go install github.com/goreleaser/goreleaser/v2@v2.16.0
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

## Publish

Release from a clean `main` branch.

```sh
git fetch --tags
git switch main
git pull --ff-only
git tag -a v0.1.0 -m "remote-monitor v0.1.0"
git push origin v0.1.0
```

The tag push starts `.github/workflows/release.yml`, which creates the GitHub
Release. GoReleaser publishes the archives and checksums first; the workflow
then replaces the release body with GitHub-generated notes so merged pull
requests are listed with their authors and full changelog links.

The release job also generates the `remote-monitor` formula from GoReleaser's
checksum file and pushes it to the `LMLiam/homebrew-tap` repository. Configure
a `TAP_GITHUB_TOKEN` repository secret before publishing the first tag. Use a
fine-grained token scoped to `LMLiam/homebrew-tap` with contents read/write
access; the workflow's normal `GITHUB_TOKEN` keeps using the release
repository's `contents: write` permission.

After the tap update lands, users can install with:

```sh
brew install LMLiam/tap/remote-monitor
```

## Verify

After the workflow finishes:

- Confirm the release contains Linux and macOS archives for `amd64` and `arm64`.
- Confirm the release notes use GitHub's generated "What's Changed" format with
  pull request author attribution.
- Confirm the checksum file lists every archive.
- Confirm `LMLiam/homebrew-tap` contains `Formula/remote-monitor.rb` for the
  release tag.
- Run `brew install LMLiam/tap/remote-monitor`.
- Run `brew test LMLiam/tap/remote-monitor`.
- Download one Linux archive and one macOS archive.
- Run `remote-monitor --help`.
- Run `remote-monitor --version` and confirm it matches the tag.

Supply-chain provenance, SBOMs, signing, and attestations belong in issue #5.
