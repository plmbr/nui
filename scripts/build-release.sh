#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${NUI_RELEASE_DIR:-$ROOT/dist}"

usage() {
  cat <<'EOF'
Usage: build-release.sh <version-tag>

Build release archives for linux, darwin, and windows (amd64 + arm64).

  version-tag   Git tag with optional v prefix (e.g. v0.1.0)

Environment:
  NUI_SKIP_UI_BUILD=1   Skip ui/npm run build (ui/dist must exist)
  NUI_RELEASE_DIR       Output directory (default: ./dist)
  NUI_RELEASE_TARGETS   Space-separated goos/goarch list to build
                        (default: all). Example: "darwin/arm64 linux/amd64"
  NUI_KEEP_DIST=1       Do not wipe NUI_RELEASE_DIR before building
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

TAG="${1:-}"
if [[ -z "$TAG" ]]; then
  echo "error: version tag required" >&2
  usage >&2
  exit 1
fi

VERSION="${TAG#v}"
echo "$VERSION" >"$ROOT/VERSION"

if [[ "${NUI_SKIP_UI_BUILD:-}" != "1" ]]; then
  echo "==> Building UI"
  (cd "$ROOT/ui" && npm run build)
elif [[ ! -d "$ROOT/ui/dist" ]]; then
  echo "error: ui/dist missing and NUI_SKIP_UI_BUILD=1" >&2
  exit 1
fi

if [[ "${NUI_KEEP_DIST:-}" != "1" ]]; then
  rm -rf "$OUT_DIR"
fi
mkdir -p "$OUT_DIR"

LDFLAGS="-s -w"
ALL_TARGETS=(
  "linux:amd64:nui"
  "linux:arm64:nui"
  "darwin:amd64:nui"
  "darwin:arm64:nui"
  "windows:amd64:nui.exe"
  "windows:arm64:nui.exe"
)

want_target() {
  local goos="$1" goarch="$2"
  if [[ -z "${NUI_RELEASE_TARGETS:-}" ]]; then
    return 0
  fi
  local t
  for t in $NUI_RELEASE_TARGETS; do
    if [[ "$t" == "${goos}/${goarch}" ]]; then
      return 0
    fi
  done
  return 1
}

HOST_OS="$(uname -s)"

echo "==> Building nui ${TAG}"
built_any=0
for target in "${ALL_TARGETS[@]}"; do
  IFS=: read -r goos goarch binary_name <<<"$target"
  want_target "$goos" "$goarch" || continue
  built_any=1

  if [[ "$goos" == "windows" ]]; then
    archive="nui_${TAG}_${goos}_${goarch}.zip"
  else
    archive="nui_${TAG}_${goos}_${goarch}.tar.gz"
  fi
  binary_dir="$OUT_DIR/.build/${goos}_${goarch}"
  mkdir -p "$binary_dir"

  echo "  -> ${goos}/${goarch}"
  (
    cd "$ROOT"
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
      go build -trimpath -ldflags="$LDFLAGS" -o "$binary_dir/${binary_name}" .
  )

  # Linux-cross-compiled darwin binaries carry a linker-signed adhoc signature
  # that macOS rejects (SIGKILL / Code Signature Invalid). Sign on macOS hosts.
  if [[ "$goos" == "darwin" && "$HOST_OS" == "Darwin" ]] && command -v codesign >/dev/null 2>&1; then
    echo "     codesign (adhoc)"
    codesign --force --sign - --timestamp=none "$binary_dir/${binary_name}"
    codesign --verify --strict "$binary_dir/${binary_name}"
  elif [[ "$goos" == "darwin" && "$HOST_OS" != "Darwin" ]]; then
    echo "warning: building darwin/${goarch} on ${HOST_OS}; binary will not be macOS-codesigned" >&2
  fi

  if [[ "$goos" == "windows" ]]; then
    (cd "$binary_dir" && zip -q "$OUT_DIR/$archive" "$binary_name")
  else
    tar -czf "$OUT_DIR/$archive" -C "$binary_dir" "$binary_name"
  fi
done

if [[ "$built_any" -ne 1 ]]; then
  echo "error: no targets matched NUI_RELEASE_TARGETS=${NUI_RELEASE_TARGETS:-}" >&2
  exit 1
fi

echo "==> Generating checksums.txt"
(
  cd "$OUT_DIR"
  shopt -s nullglob
  files=(nui_"${TAG}"_*.tar.gz nui_"${TAG}"_*.zip)
  if [[ ${#files[@]} -eq 0 ]]; then
    echo "error: no release archives found in $OUT_DIR" >&2
    exit 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${files[@]}" >checksums.txt
  else
    shasum -a 256 "${files[@]}" >checksums.txt
  fi
)

rm -rf "$OUT_DIR/.build"
echo "Release artifacts in $OUT_DIR:"
ls -1 "$OUT_DIR"
