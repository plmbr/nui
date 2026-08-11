#!/usr/bin/env bash
# Package desktop/build/bin output into dist/nui-desktop_<tag>_<os>_<arch>.{zip,tar.gz}
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/desktop/build/bin"
DIST="$ROOT/dist"

usage() {
  cat <<'EOF'
Usage: package-desktop.sh <version-tag> <os> <arch>

Packages Wails build output from desktop/build/bin into dist/.

Examples:
  package-desktop.sh v0.4.0 darwin arm64
  package-desktop.sh v0.4.0 windows amd64
  package-desktop.sh v0.4.0 linux amd64
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || $# -lt 3 ]]; then
  usage
  exit "$([[ $# -lt 3 ]] && echo 1 || echo 0)"
fi

TAG="$1"
OS="$2"
ARCH="$3"
NAME="nui-desktop_${TAG}_${OS}_${ARCH}"

mkdir -p "$DIST"
rm -f "$DIST/${NAME}.zip" "$DIST/${NAME}.tar.gz"

zip_paths() {
  local out="$1"
  shift
  python3 - "$out" "$@" <<'PY'
import os
import stat
import sys
import zipfile
from pathlib import Path

out = Path(sys.argv[1])
paths = [Path(p) for p in sys.argv[2:]]
with zipfile.ZipFile(out, "w", compression=zipfile.ZIP_DEFLATED) as zf:
    for path in paths:
        if path.is_dir():
            for f in sorted(path.rglob("*")):
                if not f.is_file():
                    continue
                arcname = f.relative_to(path.parent).as_posix()
                info = zipfile.ZipInfo.from_file(f, arcname)
                mode = f.stat().st_mode
                # Ensure Mach-O binaries stay executable after unzip on macOS.
                if mode & (stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH):
                    info.external_attr = (stat.S_IFREG | 0o755) << 16
                zf.writestr(info, f.read_bytes(), compress_type=zipfile.ZIP_DEFLATED)
        elif path.is_file():
            info = zipfile.ZipInfo.from_file(path, path.name)
            mode = path.stat().st_mode
            if mode & (stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH):
                info.external_attr = (stat.S_IFREG | 0o755) << 16
            zf.writestr(info, path.read_bytes(), compress_type=zipfile.ZIP_DEFLATED)
        else:
            raise SystemExit(f"missing path: {path}")
print(f"wrote {out}")
PY
}

case "$OS" in
  darwin)
    APP="$BIN/nui.app"
    if [[ ! -d "$APP" ]]; then
      echo "error: missing $APP" >&2
      ls -la "$BIN" >&2 || true
      exit 1
    fi
    if [[ ! -f "$APP/Contents/Resources/nui" ]]; then
      echo "error: missing bundled CLI $APP/Contents/Resources/nui (run build-desktop.sh)" >&2
      exit 1
    fi
    # Prefer ditto on macOS: preserves exec bits + resource forks for .app zips.
    if command -v ditto >/dev/null 2>&1; then
      xattr -cr "$APP" 2>/dev/null || true
      ditto -c -k --keepParent "$APP" "$DIST/${NAME}.zip"
      echo "wrote $DIST/${NAME}.zip"
    else
      zip_paths "$DIST/${NAME}.zip" "$APP"
    fi
    ;;
  windows)
    EXE=""
    for candidate in "$BIN/nui-desktop.exe" "$BIN/nui.exe"; do
      if [[ -f "$candidate" ]]; then
        EXE="$candidate"
        break
      fi
    done
    if [[ -z "$EXE" ]]; then
      echo "error: missing Windows exe in $BIN" >&2
      ls -la "$BIN" >&2 || true
      exit 1
    fi
    CLI="$BIN/nui.exe"
    # Prefer the dedicated CLI sidecar; never zip the desktop exe twice as "nui.exe".
    if [[ ! -f "$CLI" || "$CLI" -ef "$EXE" ]]; then
      echo "error: missing bundled CLI $BIN/nui.exe (run build-desktop.sh)" >&2
      ls -la "$BIN" >&2 || true
      exit 1
    fi
    zip_paths "$DIST/${NAME}.zip" "$EXE" "$CLI"
    ;;
  linux)
    BINFILE=""
    for candidate in "$BIN/nui-desktop" "$BIN/nui"; do
      if [[ -f "$candidate" ]]; then
        BINFILE="$candidate"
        break
      fi
    done
    if [[ -z "$BINFILE" ]]; then
      echo "error: missing Linux binary in $BIN" >&2
      ls -la "$BIN" >&2 || true
      exit 1
    fi
    CLI="$BIN/nui"
    if [[ ! -f "$CLI" || "$CLI" -ef "$BINFILE" ]]; then
      echo "error: missing bundled CLI $BIN/nui (run build-desktop.sh)" >&2
      ls -la "$BIN" >&2 || true
      exit 1
    fi
    (
      cd "$BIN"
      tar -czf "$DIST/${NAME}.tar.gz" "$(basename "$BINFILE")" "$(basename "$CLI")"
    )
    echo "wrote $DIST/${NAME}.tar.gz"
    ;;
  *)
    echo "error: unsupported os: $OS" >&2
    exit 1
    ;;
esac
