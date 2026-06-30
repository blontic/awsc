#!/bin/sh
# Install script for awsc (https://github.com/blontic/awsc)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/blontic/awsc/main/install.sh | sh
#
# Options (environment variables):
#   AWSC_VERSION       Install a specific version, e.g. v0.4.1 (default: latest release)
#   AWSC_INSTALL_DIR   Install location (default: /usr/local/bin, or /opt/homebrew/bin
#                      on Apple Silicon; falls back to sudo if not writable)
#
# Updating: re-run this script; it always installs the requested (or latest) version.

set -eu

REPO="blontic/awsc"
BINARY="awsc"

err() { printf 'error: %s\n' "$1" >&2; exit 1; }
info() { printf '%s\n' "$1" >&2; }

need() { command -v "$1" >/dev/null 2>&1 || err "required command not found: $1"; }

# --- prerequisites -----------------------------------------------------------
need uname
need tar
need mktemp
# need a downloader
if command -v curl >/dev/null 2>&1; then
  DL="curl -fsSL"
  DL_O="curl -fsSL -o"
elif command -v wget >/dev/null 2>&1; then
  DL="wget -qO-"
  DL_O="wget -qO"
else
  err "need curl or wget to download"
fi

# --- detect OS / arch --------------------------------------------------------
os=$(uname -s)
case "$os" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *) err "unsupported OS: $os (only macOS and Linux are supported)" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

# --- resolve version ---------------------------------------------------------
VERSION="${AWSC_VERSION:-}"
if [ -z "$VERSION" ]; then
  info "Resolving latest release..."
  VERSION=$(
    $DL "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
  )
  [ -n "$VERSION" ] || err "could not determine latest version (set AWSC_VERSION to override)"
fi
# GoReleaser archive names use the version without a leading 'v'.
VER_NUM=$(printf '%s' "$VERSION" | sed 's/^v//')

ARCHIVE="${BINARY}_${VER_NUM}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

# --- download + verify checksum ---------------------------------------------
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT INT TERM

info "Downloading ${ARCHIVE} (${VERSION})..."
$DL_O "$TMP/$ARCHIVE" "${BASE_URL}/${ARCHIVE}" || err "download failed: ${BASE_URL}/${ARCHIVE}"

info "Verifying checksum..."
if $DL_O "$TMP/checksums.txt" "${BASE_URL}/checksums.txt" 2>/dev/null; then
  expected=$(grep " ${ARCHIVE}\$" "$TMP/checksums.txt" | awk '{print $1}')
  if [ -n "$expected" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      actual=$(sha256sum "$TMP/$ARCHIVE" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      actual=$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')
    else
      info "warning: no sha256 tool found, skipping checksum verification"
      actual="$expected"
    fi
    [ "$actual" = "$expected" ] || err "checksum mismatch (expected $expected, got $actual)"
  else
    info "warning: ${ARCHIVE} not found in checksums.txt, skipping verification"
  fi
else
  info "warning: could not download checksums.txt, skipping verification"
fi

# --- extract -----------------------------------------------------------------
tar -xzf "$TMP/$ARCHIVE" -C "$TMP" || err "failed to extract archive"
[ -f "$TMP/$BINARY" ] || err "binary '$BINARY' not found in archive"
chmod +x "$TMP/$BINARY"

# macOS: strip the quarantine attribute so Gatekeeper doesn't block the binary.
if [ "$OS" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
  xattr -d com.apple.quarantine "$TMP/$BINARY" 2>/dev/null || true
fi

# --- choose install dir ------------------------------------------------------
if [ -n "${AWSC_INSTALL_DIR:-}" ]; then
  DEST="$AWSC_INSTALL_DIR"
elif [ "$OS" = "darwin" ] && [ "$ARCH" = "arm64" ] && [ -d /opt/homebrew/bin ]; then
  DEST="/opt/homebrew/bin"
else
  DEST="/usr/local/bin"
fi

mkdir -p "$DEST" 2>/dev/null || true

# --- install (with sudo fallback) -------------------------------------------
if [ -w "$DEST" ]; then
  mv "$TMP/$BINARY" "$DEST/$BINARY"
elif command -v sudo >/dev/null 2>&1; then
  info "Elevating with sudo to write to $DEST..."
  sudo mv "$TMP/$BINARY" "$DEST/$BINARY"
else
  err "cannot write to $DEST and sudo is unavailable; set AWSC_INSTALL_DIR to a writable directory"
fi

info "Installed ${BINARY} ${VERSION} to ${DEST}/${BINARY}"

# --- PATH hint + verify ------------------------------------------------------
case ":$PATH:" in
  *":$DEST:"*) ;;
  *) info "note: $DEST is not on your PATH; add it, e.g.:  export PATH=\"$DEST:\$PATH\"" ;;
esac

if command -v "$BINARY" >/dev/null 2>&1; then
  "$BINARY" version 2>/dev/null || true
fi
