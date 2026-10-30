#!/bin/sh
# Install the latest lazydash release binary for this platform.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/0xheartcode/lazydash/main/scripts/install.sh | sh
#   BIN_DIR=/usr/local/bin sh install.sh          # choose the install dir
#   VERSION=v0.2.1 sh install.sh                  # pin a specific version
set -eu

REPO="0xheartcode/lazydash"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"

say() { printf '%s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

# Detect OS.
os=$(uname -s)
case "$os" in
	Linux) os="linux" ;;
	Darwin) os="darwin" ;;
	*) die "unsupported OS: $os (use the release page or 'go install')" ;;
esac

# Detect architecture, mapped to the release naming.
arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	arm64 | aarch64) arch="arm64" ;;
	*) die "unsupported architecture: $arch" ;;
esac

# Resolve the version tag.
if [ "$VERSION" = "latest" ]; then
	VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
		| grep '"tag_name"' | head -1 | cut -d '"' -f 4)
	[ -n "$VERSION" ] || die "could not resolve the latest release tag"
fi

# goreleaser names archives with the version minus the leading v.
ver_no_v="${VERSION#v}"
asset="lazydash_${ver_no_v}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$VERSION/$asset"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

say "Downloading $asset ($VERSION)..."
curl -fsSL "$url" -o "$tmp/$asset" || die "download failed: $url"
tar -xzf "$tmp/$asset" -C "$tmp"

mkdir -p "$BIN_DIR"
install -m 0755 "$tmp/lazydash" "$BIN_DIR/lazydash"
say "Installed lazydash to $BIN_DIR/lazydash"

case ":$PATH:" in
	*":$BIN_DIR:"*) ;;
	*) say "Note: $BIN_DIR is not on your PATH. Add it, for example:"
	   say "  echo 'export PATH=\"\$PATH:$BIN_DIR\"' >> ~/.bashrc" ;;
esac
