#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: $0 <release-tag> <checksums-file> <formula-output>" >&2
  exit 2
fi

release_tag="$1"
checksums_file="$2"
formula_output="$3"

if [ ! -f "$checksums_file" ]; then
  echo "checksums file not found: $checksums_file" >&2
  exit 1
fi

version="${release_tag#v}"
project="remote-monitor"
repo="LMLiam/remote-monitor"

sha_for() {
  local artifact="$1"
  local sha

  sha="$(awk -v artifact="$artifact" '$2 == artifact { print $1 }' "$checksums_file")"
  if [ -z "$sha" ]; then
    echo "checksum missing for $artifact" >&2
    exit 1
  fi

  printf '%s' "$sha"
}

artifact_for() {
  local os="$1"
  local arch="$2"

  printf '%s_%s_%s_%s.tar.gz' "$project" "$version" "$os" "$arch"
}

url_for() {
  local artifact="$1"

  printf 'https://github.com/%s/releases/download/%s/%s' "$repo" "$release_tag" "$artifact"
}

darwin_amd64="$(artifact_for darwin amd64)"
darwin_arm64="$(artifact_for darwin arm64)"
linux_amd64="$(artifact_for linux amd64)"
linux_arm64="$(artifact_for linux arm64)"

mkdir -p "$(dirname "$formula_output")"

cat >"$formula_output" <<FORMULA
class RemoteMonitor < Formula
  desc "Terminal UI for monitoring a remote Linux host over SSH"
  homepage "https://github.com/LMLiam/remote-monitor"
  version "$version"
  license "MIT"

  on_macos do
    on_arm do
      url "$(url_for "$darwin_arm64")"
      sha256 "$(sha_for "$darwin_arm64")"
    end

    on_intel do
      url "$(url_for "$darwin_amd64")"
      sha256 "$(sha_for "$darwin_amd64")"
    end
  end

  on_linux do
    on_arm do
      url "$(url_for "$linux_arm64")"
      sha256 "$(sha_for "$linux_arm64")"
    end

    on_intel do
      url "$(url_for "$linux_amd64")"
      sha256 "$(sha_for "$linux_amd64")"
    end
  end

  def install
    bin.install "remote-monitor"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/remote-monitor --version")
  end
end
FORMULA
