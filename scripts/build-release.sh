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

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

LDFLAGS="-s -w"
TARGETS=(
  "linux:amd64:nui"
  "linux:arm64:nui"
  "darwin:amd64:nui"
  "darwin:arm64:nui"
  "windows:amd64:nui.exe"
  "windows:arm64:nui.exe"
)

echo "==> Cross-compiling nui ${TAG}"
for target in "${TARGETS[@]}"; do
  IFS=: read -r goos goarch binary_name <<<"$target"
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

  if [[ "$goos" == "windows" ]]; then
    (cd "$binary_dir" && zip -q "$OUT_DIR/$archive" "$binary_name")
  else
    tar -czf "$OUT_DIR/$archive" -C "$binary_dir" "$binary_name"
  fi
done

echo "==> Generating checksums.txt"
(
  cd "$OUT_DIR"
  shopt -s nullglob
  files=(nui_"${TAG}"_*.tar.gz nui_"${TAG}"_*.zip)
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${files[@]}" >checksums.txt
  else
    shasum -a 256 "${files[@]}" >checksums.txt
  fi
)

rm -rf "$OUT_DIR/.build"
echo "Release artifacts in $OUT_DIR:"
ls -1 "$OUT_DIR"
