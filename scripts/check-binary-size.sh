#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MAX_BYTES="${NUI_MAX_BINARY_BYTES:-25000000}"
OUT="${NUI_BINARY_CHECK_OUT:-/tmp/nui_bin_size_check}"

if [[ ! -d "$ROOT/ui/dist" ]]; then
  echo "binary size check: building ui/dist first"
  (cd "$ROOT/ui" && npm run build)
fi

(cd "$ROOT" && go build -o "$OUT" .)

if [[ "$(uname -s)" == "Darwin" ]]; then
  SIZE=$(stat -f%z "$OUT")
else
  SIZE=$(stat -c%s "$OUT")
fi

echo "nui binary size: $SIZE bytes (max $MAX_BYTES)"

if (( SIZE > MAX_BYTES )); then
  echo "ERROR: nui binary exceeds size budget ($SIZE > $MAX_BYTES)" >&2
  exit 1
fi
