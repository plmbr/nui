#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${NUI_E2E_PORT:-18080}"
export NUI_E2E_PORT="$PORT"
export NUI_URL="http://127.0.0.1:${PORT}"

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

if [[ "${E2E_AGENT:-}" == "integration" ]]; then
  OLLAMA_MOCK_PORT="${OLLAMA_MOCK_PORT:-11435}"
  python3 "$ROOT/dev/harness-examples/mock/ollama_e2e_server.py" --port "$OLLAMA_MOCK_PORT" &
  OLLAMA_PID=$!
  trap 'kill $OLLAMA_PID 2>/dev/null || true' EXIT
  export OLLAMA_HOST="http://127.0.0.1:${OLLAMA_MOCK_PORT}"

  E2E_MCP_TOOLS="$ROOT/harness-sdk/nui_mcp_tools.py"
  E2E_MCP_CONFIG="$ROOT/dev/harness-examples/mock/e2e_mcp_tools.json"
  cat > "$TEST_HOME/.nui/.mcp.json" <<EOF
{
  "mcpServers": {
    "e2e-mcp": {
      "command": "python3",
      "args": ["$E2E_MCP_TOOLS", "$E2E_MCP_CONFIG"],
      "type": "stdio"
    }
  }
}
EOF

  cat > "$TEST_HOME/.nui/mcp-servers.json" <<EOF
{
  "mcpServers": [
    {
      "name": "e2e-mcp",
      "type": "stdio",
      "command": "python3",
      "args": ["$E2E_MCP_TOOLS", "$E2E_MCP_CONFIG"]
    }
  ]
}
EOF

  mkdir -p "$TEST_HOME/.nui/skills/e2e-test-skill"
  cat > "$TEST_HOME/.nui/skills/e2e-test-skill/SKILL.md" <<'EOF'
---
name: e2e-test-skill
description: Seeded skill for local nui E2E tests.
---

When the user asks for the e2e skill check, reply exactly: E2E_SKILL_OK
EOF
fi

cd "$ROOT"
exec go run . server --port "$PORT" --no-browser
