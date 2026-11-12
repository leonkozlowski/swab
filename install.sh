#!/bin/sh
# Installs swab from GitHub Releases: curl -fsSL https://raw.githubusercontent.com/leonkozlowski/swab/main/install.sh | sh
set -eu

REPO="leonkozlowski/swab"
VERSION="${SWAB_VERSION:-latest}"
INSTALL_DIR="${SWAB_INSTALL_DIR:-}"

err() {
	echo "swab install: $*" >&2
	exit 1
}

if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -qO "$2" "$1"; }
else
	err "curl or wget is required"
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
linux | darwin) ;;
*) err "unsupported OS: $os (use go install or download from https://github.com/$REPO/releases)" ;;
esac

arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) err "unsupported architecture: $arch" ;;
esac

if [ "$VERSION" = "latest" ]; then
	base="https://github.com/$REPO/releases/latest/download"
else
	case "$VERSION" in v*) ;; *) VERSION="v$VERSION" ;; esac
	base="https://github.com/$REPO/releases/download/$VERSION"
fi

archive="swab_${os}_${arch}.tar.gz"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $archive ($VERSION)..."
fetch "$base/$archive" "$tmp/$archive" || err "download failed: $base/$archive"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || err "download failed: $base/checksums.txt"

expected=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)
[ -n "$expected" ] || err "no checksum found for $archive"
if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp/$archive" | cut -d' ' -f1)
elif command -v shasum >/dev/null 2>&1; then
	actual=$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)
else
	err "sha256sum or shasum is required to verify the download"
fi
[ "$expected" = "$actual" ] || err "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" swab

if [ -z "$INSTALL_DIR" ]; then
	if [ -w /usr/local/bin ]; then
		INSTALL_DIR=/usr/local/bin
	else
		INSTALL_DIR="$HOME/.local/bin"
	fi
fi
mkdir -p "$INSTALL_DIR"
cp "$tmp/swab" "$INSTALL_DIR/swab"
chmod 0755 "$INSTALL_DIR/swab"

echo "Installed $("$INSTALL_DIR/swab" version) to $INSTALL_DIR/swab"
case ":$PATH:" in
*":$INSTALL_DIR:"*) ;;
*) echo "Note: $INSTALL_DIR is not on your PATH; add it to use swab directly." ;;
esac
