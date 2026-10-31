#!/usr/bin/env bash
# Point the Homebrew formula at a tagged release.
#
#   1. git tag v0.1.0 && git push origin v0.1.0
#   2. scripts/release.sh 0.1.0
#   3. commit and push Formula/flightcheck.rb
set -euo pipefail

version="${1:?usage: scripts/release.sh VERSION (e.g. 0.1.0)}"
version="${version#v}"
repo="weaversam8/flightcheck"
url="https://github.com/${repo}/archive/refs/tags/v${version}.tar.gz"
formula="$(cd "$(dirname "$0")/.." && pwd)/Formula/flightcheck.rb"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
echo "fetching ${url}"
curl -fsSL "$url" -o "$tmp"

if command -v sha256sum >/dev/null; then
  sha="$(sha256sum "$tmp" | cut -d' ' -f1)"
else
  sha="$(shasum -a 256 "$tmp" | cut -d' ' -f1)"
fi

URL="$url" SHA="$sha" perl -0pi -e '
  s/^  url "[^"]*"\n//m;
  s/^  sha256 "[^"]*"\n//m;
  s/^(  homepage "[^"]*"\n)/$1  url "$ENV{URL}"\n  sha256 "$ENV{SHA}"\n/m;
' "$formula"

echo "updated ${formula}"
echo "  url    ${url}"
echo "  sha256 ${sha}"
