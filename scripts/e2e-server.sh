#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${NUI_E2E_PORT:-18080}"
export NUI_E2E_PORT="$PORT"

TEST_HOME="${NUI_TEST_HOME:-$(mktemp -d)}"
export HOME="$TEST_HOME"
mkdir -p "$TEST_HOME/.nui"

if [[ "${NUI_E2E_SKIP_BUILD:-}" != "1" ]]; then
  (cd "$ROOT/ui" && npm run build)
fi

if [[ "${E2E_AGENT:-}" == "echo" ]]; then
  python3 "$ROOT/dev/harness-examples/remote/echo_agent.py" --port 9090 &
  ECHO_PID=$!
  trap 'kill $ECHO_PID 2>/dev/null || true' EXIT
  mkdir -p "$TEST_HOME/.nui/agents"
  cat > "$TEST_HOME/.nui/agents/e2e-echo.yaml" <<'EOF'
adl: "1.0"
id: e2e-echo
name: E2E Echo
harness:
  type: remote
  host: 127.0.0.1
  port: 9090
EOF
fi

if [[ "${E2E_AGENT:-}" == "ollama" ]]; then
  OLLAMA_MOCK_PORT="${OLLAMA_MOCK_PORT:-11435}"
  python3 "$ROOT/dev/harness-examples/mock/ollama_e2e_server.py" --port "$OLLAMA_MOCK_PORT" &
  OLLAMA_PID=$!
  trap 'kill $OLLAMA_PID 2>/dev/null || true' EXIT
  export OLLAMA_HOST="http://127.0.0.1:${OLLAMA_MOCK_PORT}"
fi

cd "$ROOT"
exec go run . ui --port "$PORT" --no-browser
