---
layout: docs
title: Extension harnesses
subtitle: stdio, TCP, and HTTP harness transports and the NuiAgent framework.
permalink: /docs/extensions/harnesses/
---

Extension harnesses let you run custom agent logic behind nui's chat UI, tool-call rendering, and HITL flows. They are declared in `contributions.harnesses` and referenced in ADL as `harness.type: ext:<extension>/<harness-id>`.

## Manifest

```yaml
contributions:
  harnesses:
    source:
      file: harnesses.yaml
    runtime:
      transport: stdio
      command: ["python3", "harness_host.py"]
```

```yaml
# harnesses.yaml
harnesses:
  - id: echo
    displayName: Echo
    description: Repeats your message
  - id: reverse
    displayName: Reverse
    description: Reverses your message
```

## Transports

| Transport | Go client | When to use |
|-----------|-----------|-------------|
| `stdio` (default) | `stdioHarnessAgent` | Simplest; nui spawns one process per extension runtime |
| `tcp` | `ExtensionAgent` | Long-lived host; JSON-RPC 2.0 over TCP |
| `http` | `HTTPExtensionAgent` | Same HTTP/SSE protocol as docker/remote harnesses |

### stdio (recommended)

nui spawns `runtime.command`, communicates via JSON-RPC on stdin/stdout. One host process can multiplex multiple harness ids (see multiplex example below).

### tcp

```yaml
    runtime:
      transport: tcp
      command: ["python3", "harness_tcp_host.py"]
      host: 127.0.0.1
      port: 0    # 0 = ephemeral; host writes connection file
```

The host writes `~/.nui/connections/<id>.json`:

```json
{
  "host": "127.0.0.1",
  "port": 52341,
  "session_id": "optional",
  "pid": 9876
}
```

Connection id: `NUI_CONNECTION_ID` or `ext:<extension>/<harness-id>`.

### http

```yaml
    runtime:
      transport: http
      command: ["python3", "harness_http.py"]
```

Implements `GET /info`, `POST /run` (SSE), `POST /cancel`, `POST /shutdown`. See [Harness protocols]({{ '/docs/harness-protocols/' | relative_url }}).

## Wire protocol (stdio / TCP)

| Method | Params | Result / events |
|--------|--------|-----------------|
| `harness.info` | `{}` | `{name, version, capabilities}` |
| `harness.run` | `{message, runId, sessionId?, workingDir?, systemPrompt?, model?, harnessId?}` | Streams `harness.event` notifications |
| `harness.cancel` | `{runId}` | Cancel in-flight run |
| `harness.shutdown` | `{}` | Release resources |

### Event types (`harness.event`)

```json
{"type": "text", "content": "streaming token"}
{"type": "tool_call", "toolCallId": "tc1", "name": "grep", "args": {"pattern": "foo"}}
{"type": "tool_result", "toolCallId": "tc1", "content": "match line"}
{"type": "image", "mimeType": "image/png", "data": "<base64>"}
{"type": "hitl_request", "requestId": "...", "kind": "question", "payload": {...}}
```

## Python — NuiAgent (stdio)

Framework: [`harness-sdk/nui_agent_stdio.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_agent_stdio.py)

### Minimal echo harness

```python
#!/usr/bin/env python3
from nui_agent_stdio import NuiAgent


class EchoAgent(NuiAgent):
    name = "echo"
    version = "1.0.0"

    def run(self, message: str, run_id: str, **kwargs):
        yield message

    def on_cancel(self, run_id: str):
        pass


if __name__ == "__main__":
    EchoAgent().serve_stdio()
```

### Multiplex host (multiple harness ids)

```python
#!/usr/bin/env python3
import os
from nui_agent_stdio import NuiAgent


class HarnessHost(NuiAgent):
    name = "corp-pack-host"
    version = "1.0.0"

    def run(self, message: str, run_id: str, **kwargs):
        harness_id = kwargs.get("harnessId") or os.environ.get("NUI_HARNESS_ID", "echo")
        if harness_id == "reverse":
            yield message[::-1]
        else:
            yield f"You said: {message}"


if __name__ == "__main__":
    HarnessHost().serve_stdio()
```

### Built-in helpers on NuiAgent

| Method | Description |
|--------|-------------|
| `get_session_id()` | Current nui session id |
| `ask_user(questions=[...])` | Blocking HITL question via REST |
| `request_approval(message=...)` | Blocking approval prompt |
| `on_cancel(run_id)` | Called when user cancels |
| `on_shutdown()` | Called on `harness.shutdown` |

### HITL from a harness

```python
from nui_agent_stdio import NuiAgent


class ConfirmAgent(NuiAgent):
    def run(self, message: str, run_id: str, **kwargs):
        answer = self.ask_user(questions=[{
            "question": "Proceed with this action?",
            "options": ["Yes", "No"],
        }])
        if answer.get("answers", [{}])[0].get("answer") == "Yes":
            yield "Done."
        else:
            yield "Cancelled."
```

`ask_user` uses `NUI_API_URL`, `NUI_SESSION_ID`, and `NUI_RUN_ID` set by nui. See [HITL]({{ '/docs/extensions/hitl/' | relative_url }}).

### Streaming multiple chunks

```python
def run(self, message: str, run_id: str, **kwargs):
    for word in message.split():
        yield word + " "
```

Each `yield` emits a `text` event.

### Emitting structured events

```python
def run(self, message: str, run_id: str, **kwargs):
    yield {"type": "tool_call", "toolCallId": "1", "name": "search", "args": {"q": message}}
    yield {"type": "tool_result", "toolCallId": "1", "content": "3 results"}
    yield "Summary of results."
```

## TypeScript reference

Partial stubs: [`dev/extension-examples/ts/nui_agent_stdio.ts`](https://github.com/plmbr/nui/blob/main/dev/extension-examples/ts/nui_agent_stdio.ts). Python is the canonical SDK.

## TCP JSON-RPC (advanced)

Standalone TCP framework: [`harness-sdk/nui_agent.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_agent.py). Used by reference examples in `dev/harness-examples/py/` — not registered as built-in agent types.

```python
from nui_agent import NuiAgent

class EchoAgent(NuiAgent):
    def run(self, message, **ctx):
        yield message

if __name__ == "__main__":
    EchoAgent("echo", port=7432).serve()
```

## ADL usage

```yaml
id: my-echo-agent
name: My Echo
harness:
  type: ext:corp-pack/echo
  # Optional per-harness overrides:
  # model: ...
  # systemPrompt: ...
```

Multi-step workflows can use extension harnesses as steps:

```yaml
steps:
  - id: preprocess
    harness:
      type: ext:corp-pack/reverse
    outputs:
      - name: reversed
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `NUI_HARNESS_ID` | Active harness id (multiplex hosts) |
| `NUI_EXTENSION_DIR` | Extension install path |
| `NUI_EXTENSION_NAME` | Manifest `name` |
| `NUI_API_URL` | REST base URL |
| `NUI_SESSION_ID` | Session id |
| `NUI_RUN_ID` | Current run id |
| `NUI_HITL_SDK_DIR` | Path to HITL SDK (when HITL enabled) |

## Further reading

- [Harness protocols]({{ '/docs/harness-protocols/' | relative_url }}) — HTTP/SSE details
- [Programmatic SDK]({{ '/docs/extensions/programmatic/' | relative_url }}) — `run_harness()` without static yaml
- [corp-pack example](https://github.com/plmbr/nui/tree/main/dev/extension-examples/corp-pack/)
