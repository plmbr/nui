#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${LOOP_E2E_PORT:-18080}"
export LOOP_E2E_PORT="$PORT"

TEST_HOME="${LOOP_TEST_HOME:-$(mktemp -d)}"
export HOME="$TEST_HOME"
mkdir -p "$TEST_HOME/.loop"

if [[ "${LOOP_E2E_SKIP_BUILD:-}" != "1" ]]; then
  (cd "$ROOT/ui" && npm run build)
fi

if [[ "${E2E_AGENT:-}" == "echo" ]]; then
  python3 "$ROOT/dev/harness-examples/remote/echo_agent.py" --port 9090 &
  ECHO_PID=$!
  trap 'kill $ECHO_PID 2>/dev/null || true' EXIT
  mkdir -p "$TEST_HOME/.loop/agents"
  cat > "$TEST_HOME/.loop/agents/e2e-echo.yaml" <<'EOF'
adl: "1.0"
id: e2e-echo
name: E2E Echo
harness:
  type: remote
  host: 127.0.0.1
  port: 9090
EOF
fi

cd "$ROOT"
exec go run . ui --port "$PORT" --no-browser
