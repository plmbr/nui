---
layout: docs
title: Harness protocols
subtitle: HTTP/SSE, stdio JSON-RPC, and TCP harness wire formats.
permalink: /docs/harness-protocols/
---

nui supports multiple harness execution paths. Extension authors typically implement **stdio** or **TCP** JSON-RPC harnesses; **HTTP/SSE** is used for Docker, remote, and some extension hosts.

## Builtin harnesses (Go subprocess)

The four built-in CLI agents are managed directly by Go — not via the extension wire protocol:

| Harness | Implementation |
|---------|----------------|
| `claude-code` | `ClaudeCodeAgent` + persistent session |
| `pi` | `PiAgent` (`pi --mode rpc`) |
| `codex` | `CodexAgent` |
| `opencode` | `OpenCodeAgent` |

For `sandbox: docker`, builtins use HTTP/SSE inside nui-managed containers (`docker/` images, port **8090**).

## HTTP/SSE — docker and remote

Used by ADL `harness.type: docker`, `devcontainer`, or `remote`, and by extension harnesses with `transport: http`.

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /info` | Health check — `{name, version, capabilities}` |
| `POST /run` | Body: `{message, sessionId?, workingDir?, systemPrompt?, model?}` → `text/event-stream` |
| `POST /cancel` | Body: `{runId}` |
| `POST /shutdown` | Release resources; nui calls before `docker stop` |

### SSE events

```json
{"type": "text", "content": "..."}
{"type": "done", "sessionId": "..."}
{"type": "error", "error": "..."}
```

Extended types (tool calls, images, HITL) are supported — see `eventFromHarnessParams` in the Go extension client.

### Minimal Python HTTP harness

```python
#!/usr/bin/env python3
"""Minimal HTTP/SSE echo harness on port 9090."""

import json
from http.server import BaseHTTPRequestHandler, HTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/info":
            self._json(200, {"name": "echo", "version": "1.0.0", "capabilities": {}})
            return
        self.send_error(404)

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length)) if length else {}

        if self.path == "/run":
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            msg = body.get("message", "")
            self.wfile.write(f'data: {json.dumps({"type": "text", "content": msg})}\n\n'.encode())
            self.wfile.write(f'data: {json.dumps({"type": "done"})}\n\n'.encode())
            return

        if self.path == "/cancel":
            self._json(200, {"ok": True})
            return

        if self.path == "/shutdown":
            self._json(200, {"ok": True})
            return

        self.send_error(404)

    def _json(self, code, obj):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(obj).encode())

    def log_message(self, *_):
        pass


if __name__ == "__main__":
    HTTPServer(("127.0.0.1", 9090), Handler).serve_forever()
```

Reference implementations: [`docker/http_nui_agent.py`](https://github.com/plmbr/nui/blob/main/docker/http_nui_agent.py), [`dev/harness-examples/docker/`](https://github.com/plmbr/nui/tree/main/dev/harness-examples/docker/).

## Extension harnesses — stdio / TCP JSON-RPC

Installed extensions contribute harnesses under `contributions.harnesses`. ADL references them as `harness.type: ext:<extension>/<harness-id>`.

| Transport | Go client | Runtime |
|-----------|-----------|---------|
| `stdio` (default) | `stdioHarnessAgent` | nui spawns the extension host process |
| `tcp` | `ExtensionAgent` | Host writes `~/.nui/connections/<id>.json`; nui dials JSON-RPC |
| `http` | `HTTPExtensionAgent` | Same HTTP/SSE protocol as docker/remote |

### Wire methods (stdio and TCP)

| Method | Description |
|--------|-------------|
| `harness.info` | Returns `{name, version, capabilities}` |
| `harness.run` | Params: `{message, runId, sessionId?, workingDir?, systemPrompt?, model?, harnessId?}`; streams `harness.event` notifications |
| `harness.cancel` | Params: `{runId}` |
| `harness.shutdown` | Release resources |

### `harness.event` notification shapes

```json
{"type": "text", "content": "partial output"}
{"type": "tool_call", "toolCallId": "...", "name": "...", "args": {}}
{"type": "tool_result", "toolCallId": "...", "content": "..."}
{"type": "hitl_request", "requestId": "...", "kind": "question", "payload": {}}
{"type": "image", "mimeType": "image/png", "data": "<base64>"}
```

### Python stdio framework

Use [`harness-sdk/nui_agent_stdio.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_agent_stdio.py) — see [Extension harnesses]({{ '/docs/extensions/harnesses/' | relative_url }}) for a full example.

### TCP connection file

TCP and HTTP extension hosts write `~/.nui/connections/<id>.json`:

```json
{
  "host": "127.0.0.1",
  "port": 52341,
  "session_id": "abc-123",
  "pid": 9876
}
```

The connection id defaults to the harness name or `NUI_CONNECTION_ID` (derived from `ext:<extension>/<harness-id>`).

## API harness (in-process)

`harness.type: api` runs inside the nui binary. Providers: `anthropic`, `openai`, `gemini`, `openrouter`, `ollama`. Tool calling uses session-scoped MCP servers from ADL `aiAssets.mcpServers`.

## Devcontainer harness

`harness.type: devcontainer` runs a builtin CLI (`innerHarness`) inside a nui-provisioned dev container. nui generates `devcontainer.json` under `~/.nui/sessions/<session-id>/.devcontainer/`.

Requires `devcontainer` CLI on PATH and Docker.

## Environment variables (harness subprocesses)

| Variable | Description |
|----------|-------------|
| `NUI_API_URL` | nui REST base URL (e.g. `http://127.0.0.1:8080`) |
| `NUI_SESSION_ID` | Current nui session id |
| `NUI_RUN_ID` | Current run id |
| `NUI_EXTENSION_DIR` | Extension install directory |
| `NUI_EXTENSION_NAME` | Extension manifest `name` |
| `NUI_HARNESS_ID` | Active harness id (multiplexed hosts) |
| `NUI_CONNECTION_ID` | TCP/HTTP connection file id |

## Further reading

- [Extension harnesses]({{ '/docs/extensions/harnesses/' | relative_url }}) — manifest, multiplex hosts, HITL from harnesses
- [Harness examples](https://github.com/plmbr/nui/tree/main/dev/harness-examples/) — Python, TypeScript, Docker, remote
