#!/bin/sh
# es-tui installer
# Usage: curl -fsSL https://raw.githubusercontent.com/davidbudnick/es-tui/main/install.sh | bash
#
# Environment variables:
#   INSTALL_DIR  — directory to install the binary (default: ~/.local/bin)

set -e

REPO="davidbudnick/es-tui"
BINARY="es-tui"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

info() { printf '\033[1;34m%s\033[0m\n' "$1"; }
error() { printf '\033[1;31merror: %s\033[0m\n' "$1" >&2; exit 1; }

OS="$(uname -s)"
case "$OS" in
  Darwin) OS="Darwin" ;;
  Linux)  OS="Linux" ;;
  *)      error "Unsupported operating system: $OS" ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="x86_64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)             error "Unsupported architecture: $ARCH" ;;
esac

info "Detected platform: ${OS}/${ARCH}"

info "Fetching latest release..."
LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
if command -v curl >/dev/null 2>&1; then
  RELEASE_JSON="$(curl -fsSL "$LATEST_URL")"
elif command -v wget >/dev/null 2>&1; then
  RELEASE_JSON="$(wget -qO- "$LATEST_URL")"
else
  error "curl or wget is required"
fi

VERSION="$(printf '%s' "$RELEASE_JSON" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')"
[ -z "$VERSION" ] && error "Could not determine latest version"

info "Latest version: ${VERSION}"

VER="${VERSION#v}"
ARCHIVE="${BINARY}_${VER}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
ARCHIVE_URL="${BASE_URL}/${ARCHIVE}"
CHECKSUMS_URL="${BASE_URL}/checksums.txt"

TMPDIR="$(mktemp -d)"
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

info "Downloading ${ARCHIVE}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$ARCHIVE_URL"
  curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL"
else
  wget -qO "${TMPDIR}/${ARCHIVE}" "$ARCHIVE_URL"
  wget -qO "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL"
fi

info "Verifying checksum..."
EXPECTED="$(grep "${ARCHIVE}" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
[ -z "$EXPECTED" ] && error "Checksum not found for ${ARCHIVE}"

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')"
else
  error "sha256sum or shasum is required"
fi

[ "$ACTUAL" = "$EXPECTED" ] || error "Checksum mismatch"

info "Extracting..."
tar -xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}"

mkdir -p "${INSTALL_DIR}"
install -m 755 "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

info "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
info "Ensure ${INSTALL_DIR} is on your PATH"
