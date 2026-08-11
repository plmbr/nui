#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DESKTOP="$ROOT/desktop"
PLATFORM=""
TAGS=""
SKIP_UI="${NUI_SKIP_UI_BUILD:-0}"

usage() {
  cat <<'EOF'
Usage: build-desktop.sh [options]

Build the nui desktop app (Wails) for the current or specified platform.

Requires: wails CLI, CGO, platform webview libs.
Builds ui/dist first unless NUI_SKIP_UI_BUILD=1 or --skip-ui.

Options:
  --platform <os/arch>   Pass -platform to wails (e.g. darwin/arm64, windows/amd64, linux/amd64)
  --tags <go-tags>       Pass -tags to wails (e.g. webkit2_41 on Ubuntu 24.04+)
  --skip-ui              Skip ui/npm run build (ui/dist must exist)
  -h, --help             Show this help

Environment:
  NUI_SKIP_UI_BUILD=1    Same as --skip-ui
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --platform)
      PLATFORM="${2:-}"
      shift 2
      ;;
    --tags)
      TAGS="${2:-}"
      shift 2
      ;;
    --skip-ui)
      SKIP_UI=1
      shift
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if ! command -v wails >/dev/null 2>&1; then
  echo "error: wails CLI not found (install: go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2)" >&2
  exit 1
fi

if [[ "$SKIP_UI" != "1" ]]; then
  echo "==> Building UI"
  (cd "$ROOT/ui" && npm run build)
elif [[ ! -d "$ROOT/ui/dist" ]]; then
  echo "error: ui/dist missing and --skip-ui / NUI_SKIP_UI_BUILD=1" >&2
  exit 1
fi

echo "==> Generating app icon"
"$ROOT/scripts/generate-desktop-icon.sh"

echo "==> Building nui desktop (Wails)"
build_args=(-skipbindings)
if [[ -n "$PLATFORM" ]]; then
  build_args+=(-platform "$PLATFORM")
fi
if [[ -n "$TAGS" ]]; then
  build_args+=(-tags "$TAGS")
fi
(
  cd "$DESKTOP"
  # No Go bindings are used (UI talks to localhost HTTP); skip binding gen.
  wails build "${build_args[@]}"
)

echo "==> Done"
echo "    binary: $DESKTOP/build/bin/"
ls -la "$DESKTOP/build/bin/" 2>/dev/null || true
