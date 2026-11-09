#!/bin/sh
# Install flightcheck from a GitHub release, without Homebrew.
#
#   curl -fsSL https://raw.githubusercontent.com/weaversam8/flightcheck/main/install.sh | sh
#
# Environment:
#   FLIGHTCHECK_VERSION      release tag to install, e.g. v0.1.0 (default: latest)
#   FLIGHTCHECK_INSTALL_DIR  where to put the binaries (default: ~/.local/bin)
#   FLIGHTCHECK_SETUP=1      also run `flightcheck install` to add the Claude Code hooks
#   FLIGHTCHECK_ARCHIVE      install from a local tarball instead of downloading
#   FLIGHTCHECK_BASE_URL     download from a mirror instead of GitHub releases
set -eu

repo="weaversam8/flightcheck"
version="${FLIGHTCHECK_VERSION:-latest}"
dir="${FLIGHTCHECK_INSTALL_DIR:-$HOME/.local/bin}"

say() { printf '%s\n' "$*"; }
die() { printf 'flightcheck install: %s\n' "$*" >&2; exit 1; }

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) die "unsupported OS $(uname -s) (Linux and macOS only)" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) die "unsupported architecture $(uname -m)" ;;
esac
asset="flightcheck_${os}_${arch}.tar.gz"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

fetch() { # url dest
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$1" -O "$2"
  else
    die "need curl or wget"
  fi
}

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | cut -d' ' -f1
  else
    die "need sha256sum or shasum to verify the download"
  fi
}

if [ -n "${FLIGHTCHECK_ARCHIVE:-}" ]; then
  cp "$FLIGHTCHECK_ARCHIVE" "$tmp/$asset"
  say "using local archive $FLIGHTCHECK_ARCHIVE"
else
  if [ -n "${FLIGHTCHECK_BASE_URL:-}" ]; then
    base="$FLIGHTCHECK_BASE_URL"
  elif [ "$version" = latest ]; then
    base="https://github.com/$repo/releases/latest/download"
  else
    base="https://github.com/$repo/releases/download/$version"
  fi
  say "downloading $asset ($version)"
  fetch "$base/$asset" "$tmp/$asset" || die "download failed: $base/$asset"
  fetch "$base/checksums.txt" "$tmp/checksums.txt" || die "couldn't fetch checksums.txt"
  want="$(grep " $asset\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
  [ -n "$want" ] || die "no checksum listed for $asset"
  got="$(sha256 "$tmp/$asset")"
  [ "$want" = "$got" ] || die "checksum mismatch for $asset"
  say "checksum ok"
fi

tar -xzf "$tmp/$asset" -C "$tmp"
mkdir -p "$dir"
# Replace atomically so a running hook never sees a half-written binary.
cp "$tmp/flightcheck_${os}_${arch}/flightcheck" "$dir/.flightcheck.new"
chmod 755 "$dir/.flightcheck.new"
mv -f "$dir/.flightcheck.new" "$dir/flightcheck"
ln -sf flightcheck "$dir/flchk"
say "installed $("$dir/flightcheck" version) to $dir (also as flchk)"

case ":$PATH:" in
  *":$dir:"*) ;;
  *) say ""
     say "note: $dir isn't on your PATH. Add this to your shell profile:"
     say "  export PATH=\"$dir:\$PATH\"" ;;
esac

if [ "${FLIGHTCHECK_SETUP:-}" = 1 ]; then
  say ""
  # Under `curl | sh`, stdin is this script, so ask on the terminal instead.
  if [ ! -t 0 ] && (: </dev/tty) 2>/dev/null; then
    "$dir/flightcheck" install </dev/tty
  else
    "$dir/flightcheck" install
  fi
else
  say ""
  say "next: run '$dir/flightcheck install' to add the Claude Code hooks"
fi
