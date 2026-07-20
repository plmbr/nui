#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Go tests"
# Avoid scanning ui/node_modules (contains nested Go packages) and Playwright artifacts.
"$ROOT/scripts/ensure-ui-dist.sh"
(cd "$ROOT" && go test -coverprofile=coverage.out . ./cmd/... ./internal/...)

echo "==> harness-sdk pytest"
HARNESS_VENV="$ROOT/.venv-harness"
if [ ! -x "$HARNESS_VENV/bin/pytest" ]; then
  python3 -m venv "$HARNESS_VENV"
  "$HARNESS_VENV/bin/pip" install pytest >/dev/null
fi
"$HARNESS_VENV/bin/pytest" "$ROOT/harness-sdk"

echo "==> Binary size check"
"$ROOT/scripts/check-binary-size.sh"

echo "==> UI lint + build + unit tests"
(cd "$ROOT/ui" && npm run lint && npm run build && npm run test:coverage)

echo "==> Playwright E2E (echo agent by default; set ANTHROPIC_API_KEY for real claude-code chat)"
export E2E_AGENT="${E2E_AGENT:-echo}"
(cd "$ROOT/ui" && npx playwright install chromium && npm run test:e2e -- --grep-invert "ollama api")

echo "==> Playwright E2E (ollama api mock)"
export E2E_AGENT=ollama
(cd "$ROOT/ui" && npm run test:e2e -- e2e/ollama-api.spec.ts)

echo "All tests passed."
