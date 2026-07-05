#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Go tests"
# Avoid scanning ui/node_modules (contains nested Go packages) and Playwright artifacts.
(cd "$ROOT" && go test . ./cmd/... ./internal/...)

echo "==> UI lint + build + unit tests"
(cd "$ROOT/ui" && npm run lint && npm run build && npm run test)

echo "==> Playwright E2E (echo agent by default; set ANTHROPIC_API_KEY for real claude-code chat)"
export E2E_AGENT="${E2E_AGENT:-echo}"
(cd "$ROOT/ui" && npx playwright install chromium && npm run test:e2e)

echo "All tests passed."
