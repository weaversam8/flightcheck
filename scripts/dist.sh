#!/usr/bin/env bash
# Build release tarballs for every supported platform into dist/.
#
#   scripts/dist.sh 0.1.0
#
# Produces dist/flightcheck_<os>_<arch>.tar.gz (each containing flightcheck
# and the flchk symlink) plus dist/checksums.txt. Asset names carry no
# version so install.sh can always fetch .../releases/latest/download/<name>.
set -euo pipefail

version="${1:?usage: scripts/dist.sh VERSION}"
version="${version#v}"
root="$(cd "$(dirname "$0")/.." && pwd)"
dist="$root/dist"
platforms=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64)

rm -rf "$dist"
mkdir -p "$dist"
cd "$root"

for p in "${platforms[@]}"; do
  os="${p%/*}" arch="${p#*/}"
  name="flightcheck_${os}_${arch}"
  stage="$dist/$name"
  mkdir -p "$stage"
  echo "building $name"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w -X main.version=${version}" \
    -o "$stage/flightcheck" ./cmd/flchk
  ln -s flightcheck "$stage/flchk"
  cp README.md LICENSE "$stage/"
  tar -C "$dist" -czf "$dist/$name.tar.gz" "$name"
  rm -rf "$stage"
done

cd "$dist"
if command -v sha256sum >/dev/null; then
  sha256sum ./*.tar.gz | sed 's#\./##' > checksums.txt
else
  shasum -a 256 ./*.tar.gz | sed 's#\./##' > checksums.txt
fi
cat checksums.txt
