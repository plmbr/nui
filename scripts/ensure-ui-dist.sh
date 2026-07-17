#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/ui/dist"

if [[ -d "$DIST" ]] && [[ -n "$(find "$DIST" -mindepth 1 -maxdepth 1 2>/dev/null | head -n 1)" ]]; then
  exit 0
fi

echo "ensure-ui-dist: creating minimal ui/dist for Go compile"
mkdir -p "$DIST"
cat >"$DIST/index.html" <<'EOF'
<!DOCTYPE html>
<html><head><title>nui</title></head><body>nui</body></html>
EOF
