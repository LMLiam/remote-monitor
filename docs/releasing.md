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
go install github.com/goreleaser/goreleaser/v2@v2.20.2
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
Release. GoReleaser publishes the archives, checksums, and SPDX JSON SBOMs
first. The workflow then publishes GitHub build-provenance attestations for the
checksum-listed release artifacts and replaces the release body with
GitHub-generated notes so merged pull requests are listed with their authors
and full changelog links.

The release job also generates the `remote-monitor` formula from GoReleaser's
checksum file and opens a formula PR in the `LMLiam/homebrew-tap` repository.
The PR must be merged before the new version is available to Homebrew. The
workflow authenticates as a GitHub App with a short-lived token scoped to that
tap. Create a private GitHub App with Contents and Pull requests read/write
permissions, and install it only on `LMLiam/homebrew-tap`. Then configure these
in this repository before publishing the first tag:

- the repository variable `TAP_APP_CLIENT_ID`, set to the App's Client ID;
- the repository secret `TAP_APP_PRIVATE_KEY`, set to the contents of the App's
  `.pem` private key.

The workflow's normal `GITHUB_TOKEN` keeps using the release repository's
`contents: write` permission.

After the formula PR is merged, users can install with:

```sh
brew install LMLiam/tap/remote-monitor
```

## Verify Downloaded Artifacts

Download one archive, its checksum file, and its SBOM from the release page:

```sh
tag=v0.1.2
version="${tag#v}"
repo=LMLiam/remote-monitor
archive="remote-monitor_${version}_linux_amd64.tar.gz"
sbom="${archive}.sbom.spdx.json"

gh release download "$tag" --repo "$repo" \
  --pattern "$archive" \
  --pattern "remote-monitor_${version}_checksums.txt" \
  --pattern "$sbom"
```

Verify the archive checksum:

```sh
grep "  ${archive}$" "remote-monitor_${version}_checksums.txt" | shasum -a 256 -c -
```

Inspect the SBOM enough to confirm it is a non-empty SPDX JSON document:

```sh
test -s "$sbom"
grep -q '"spdxVersion"' "$sbom"
grep -q '"packages"' "$sbom"
```

Verify the GitHub build-provenance attestation for the downloaded archive:

```sh
gh attestation verify "$archive" --repo "$repo"
```

## Verify

After the workflow finishes:

- Confirm the release contains Linux and macOS archives for `amd64` and `arm64`.
- Confirm the release notes use GitHub's generated "What's Changed" format with
  pull request author attribution.
- Confirm the checksum file lists every archive.
- Confirm each archive has a matching `.sbom.spdx.json` release asset.
- Confirm GitHub build-provenance attestations verify for at least one
  downloaded archive.
- Confirm the formula PR in `LMLiam/homebrew-tap` is merged and that its
  `Formula/remote-monitor.rb` targets the release tag.
- Run `brew install LMLiam/tap/remote-monitor`.
- Run `brew test LMLiam/tap/remote-monitor`.
- Download one Linux archive and one macOS archive.
- Run the checksum, SBOM, and attestation commands above for at least one
  archive.
- Run `remote-monitor --help`.
- Run `remote-monitor --version` and confirm it matches the tag.
