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
CLI_GOOS=""
CLI_GOARCH=""
if [[ -n "$PLATFORM" ]]; then
  build_args+=(-platform "$PLATFORM")
  CLI_GOOS="${PLATFORM%%/*}"
  CLI_GOARCH="${PLATFORM##*/}"
fi
if [[ -n "$TAGS" ]]; then
  build_args+=(-tags "$TAGS")
fi
(
  cd "$DESKTOP"
  # No Go bindings are used (UI talks to localhost HTTP); skip binding gen.
  wails build "${build_args[@]}"
)

# Bundle a CGO-free CLI next to the desktop app for first-launch PATH install.
echo "==> Building bundled nui CLI"
BIN_DIR="$DESKTOP/build/bin"
CLI_NAME="nui"
CLI_GOOS_EFF="${CLI_GOOS:-}"
CLI_GOARCH_EFF="${CLI_GOARCH:-}"
if [[ -z "$CLI_GOOS_EFF" ]]; then
  case "$(uname -s)" in
    MINGW*|MSYS*|CYGWIN*) CLI_NAME="nui.exe" ;;
  esac
elif [[ "$CLI_GOOS_EFF" == "windows" ]]; then
  CLI_NAME="nui.exe"
fi
CLI_OUT="$BIN_DIR/$CLI_NAME"
mkdir -p "$BIN_DIR"
(
  cd "$ROOT"
  build_env=(CGO_ENABLED=0)
  if [[ -n "$CLI_GOOS_EFF" ]]; then
    build_env+=(GOOS="$CLI_GOOS_EFF" GOARCH="$CLI_GOARCH_EFF")
  fi
  env "${build_env[@]}" go build -trimpath -ldflags="-s -w" -o "$CLI_OUT" .
)
chmod +x "$CLI_OUT" 2>/dev/null || true

# macOS: ship CLI inside the .app so the archive stays a single bundle.
if [[ -d "$BIN_DIR/nui.app" ]]; then
  RESOURCES="$BIN_DIR/nui.app/Contents/Resources"
  mkdir -p "$RESOURCES"
  cp "$CLI_OUT" "$RESOURCES/nui"
  chmod +x "$RESOURCES/nui"
  echo "    staged: $RESOURCES/nui"

  # Copying into Resources invalidates any signature Wails applied. Re-sign
  # ad-hoc so Gatekeeper does not report the app as "damaged". Developer ID
  # notarization (when available) still required to skip the malware prompt
  # for downloads.
  if command -v codesign >/dev/null 2>&1 && [[ "$(uname -s)" == "Darwin" ]]; then
    echo "==> Ad-hoc codesigning nui.app"
    xattr -cr "$BIN_DIR/nui.app" 2>/dev/null || true
    codesign --force --sign - --timestamp=none "$RESOURCES/nui"
    codesign --force --deep --sign - --timestamp=none "$BIN_DIR/nui.app"
    codesign --verify --deep --strict "$BIN_DIR/nui.app" 2>&1 || true
  fi
fi

echo "==> Done"
echo "    binary: $BIN_DIR/"
ls -la "$BIN_DIR/" 2>/dev/null || true
