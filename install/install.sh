#!/bin/sh
# nui installer — https://nui.plmbr.dev/install.sh
# Usage: curl -fsSL https://nui.plmbr.dev/install.sh | sh
#
# Environment:
#   NUI_VERSION   Release tag (default: latest), e.g. v0.1.0
#   NUI_INSTALL_DIR  Install directory (default: $HOME/.local/bin)
#   GITHUB_REPO   GitHub owner/repo (default: plmbr/nui)

set -e

GITHUB_REPO="${GITHUB_REPO:-plmbr/nui}"
INSTALL_DIR="${NUI_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${NUI_VERSION:-latest}"
BINARY_NAME="nui"

say() { printf '%s\n' "$*"; }
err() { printf 'error: %s\n' "$*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || err "required command not found: $1"
}

detect_platform() {
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
    Linux)  platform_os="linux" ;;
    Darwin) platform_os="darwin" ;;
    *)
      err "unsupported operating system: $os

On Windows, use PowerShell:
  irm https://nui.plmbr.dev/install.ps1 | iex

Or download a release manually:
  https://github.com/${GITHUB_REPO}/releases"
      ;;
  esac

  case "$arch" in
    x86_64|amd64)   platform_arch="amd64" ;;
    aarch64|arm64) platform_arch="arm64" ;;
    *)
      err "unsupported architecture: $arch

Download a release manually:
  https://github.com/${GITHUB_REPO}/releases"
      ;;
  esac
}

resolve_version() {
  if [ "$VERSION" = "latest" ]; then
    need_cmd curl
    VERSION="$(
      curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" \
        | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' \
        | head -n 1
    )"
    [ -n "$VERSION" ] || err "could not resolve latest release from GitHub"
  fi
}

download_file() {
  url="$1"
  dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
  else
    err "curl or wget is required to download nui"
  fi
}

verify_checksum() {
  archive_name="$1"
  archive_path="$2"
  checksums_path="$3"

  if ! grep -q " ${archive_name}$" "$checksums_path" 2>/dev/null; then
    err "checksums.txt does not list ${archive_name}"
  fi

  expected="$(grep " ${archive_name}$" "$checksums_path" | awk '{print $1}')"

  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$archive_path" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
  else
    say "warning: sha256sum/shasum not found; skipping checksum verification"
    return 0
  fi

  if [ "$expected" != "$actual" ]; then
    err "checksum mismatch for ${archive_name}"
  fi
}

path_contains_dir() {
  dir="$1"
  case ":${PATH}:" in
    *:"$dir":*) return 0 ;;
    *) return 1 ;;
  esac
}

# macOS marks curl downloads with com.apple.quarantine; extracted binaries inherit it.
remove_macos_quarantine() {
  [ "$platform_os" = "darwin" ] || return 0
  command -v xattr >/dev/null 2>&1 || return 0
  for path in "$@"; do
    [ -e "$path" ] || continue
    xattr -d com.apple.quarantine "$path" 2>/dev/null || true
  done
}

install_binary() {
  src="$1"
  dest="$2"
  if [ "$platform_os" = "darwin" ]; then
    remove_macos_quarantine "$src"
    cp -X "$src" "$dest"
    chmod 755 "$dest"
    remove_macos_quarantine "$dest"
  else
    install -m 755 "$src" "$dest"
  fi
}

main() {
  need_cmd tar
  need_cmd mkdir
  detect_platform
  resolve_version

  archive_name="${BINARY_NAME}_${VERSION}_${platform_os}_${platform_arch}.tar.gz"
  base_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}"
  archive_url="${base_url}/${archive_name}"
  checksums_url="${base_url}/checksums.txt"

  tmpdir="$(mktemp -d 2>/dev/null || mktemp -d -t nui-install)"
  trap 'rm -rf "$tmpdir"' EXIT INT HUP TERM

  say "Installing nui ${VERSION} (${platform_os}/${platform_arch})"

  download_file "$checksums_url" "$tmpdir/checksums.txt"
  download_file "$archive_url" "$tmpdir/${archive_name}"
  verify_checksum "$archive_name" "$tmpdir/${archive_name}" "$tmpdir/checksums.txt"
  remove_macos_quarantine "$tmpdir/${archive_name}"

  tar -xzf "$tmpdir/${archive_name}" -C "$tmpdir"
  [ -f "$tmpdir/${BINARY_NAME}" ] || err "archive did not contain ${BINARY_NAME} binary"

  mkdir -p "$INSTALL_DIR"
  install_binary "$tmpdir/${BINARY_NAME}" "$INSTALL_DIR/${BINARY_NAME}"

  say "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"

  if ! path_contains_dir "$INSTALL_DIR"; then
    say ""
    say "Add ${INSTALL_DIR} to your PATH, for example:"
    say "  export PATH=\"${INSTALL_DIR}:\$PATH\""
  fi

  if command -v "$INSTALL_DIR/${BINARY_NAME}" >/dev/null 2>&1; then
    installed_version="$("$INSTALL_DIR/${BINARY_NAME}" --version 2>/dev/null || true)"
    if [ -n "$installed_version" ]; then
      say "nui version: ${installed_version}"
    fi
  fi
}

main "$@"
