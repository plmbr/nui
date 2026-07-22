---
layout: docs
title: Human-in-the-loop (HITL)
subtitle: Interactive prompts, delivery channels, and the REST API.
permalink: /docs/extensions/hitl/
---

HITL lets harnesses and tools pause for human input — questions, approvals, and multi-step forms. nui routes prompts to **delivery channels** (built-in UI and extension channels) and collects responses via a canonical request envelope.

## ADL configuration

```yaml
hitl:
  mode: interactive    # interactive | disabled
  channels:
    - nui-ui                              # built-in chat cards
    - ext:hitl-demo/demo-slack            # extension channel
```

Built-in channel: **`nui-ui`** — renders prompt cards in the chat UI.

## Builtin harness MCP server

When `hitl.mode` is `interactive`, builtin harnesses (Claude, Codex, Pi, OpenCode) receive an injected **`nui-hitl`** MCP server (`nui hitl-mcp`) with tools:

- `ask_user` — structured questions with options
- `request_approval` — yes/no approval flow

## Extension HITL channels

Declare channels for discovery and optional stdio delivery hosts.

### Manifest

```yaml
contributions:
  hitlChannels:
    source:
      file: hitl-channels.yaml
    runtime:
      transport: stdio
      command: ["python3", "hitl_channel_host.py"]
```

`hitl-channels.yaml`:

```yaml
hitlChannels:
  - id: demo-slack
    displayName: Demo Slack Channel
    description: Forwards HITL prompts to Slack (example)
```

Channel id in ADL: `ext:<extension>/demo-slack`

### Wire protocol (`hitl.*`)

| Method | Params | Result |
|--------|--------|--------|
| `hitl.info` | `{}` | `{id, name, version, capabilities}` |
| `hitl.deliver` | `{channelId, request, workingDir?, sessionId?}` | `{ok: true, ...}` |
| `hitl.shutdown` | `{}` | `{ok: true}` |

The `request` object is the canonical HITL envelope:

```json
{
  "requestId": "uuid",
  "kind": "question",
  "status": "pending",
  "payload": {
    "title": "Confirm deploy",
    "message": "Deploy to production?",
    "questions": [
      {"question": "Environment?", "options": ["staging", "production"]}
    ]
  },
  "routing": {
    "channels": ["nui-ui", "ext:hitl-demo/demo-slack"]
  }
}
```

### Python channel host

```python
#!/usr/bin/env python3
import os
import sys

sys.path.insert(0, os.path.expanduser("~/.nui/harness-sdk"))
from nui_hitl_channel import NuiHITLChannelProvider


class DemoHITLChannelHost(NuiHITLChannelProvider):
    name = "hitl-demo-channels"
    version = "1.0.0"

    def on_deliver(self, channel_id, request, **kwargs):
        payload = request.get("payload") or {}
        title = payload.get("title") or payload.get("message") or request.get("kind")
        # Forward to Slack, email, etc.
        print(f"[demo] channel={channel_id} title={title!r}", file=sys.stderr)
        return {"ok": True, "delivered": True}


if __name__ == "__main__":
    DemoHITLChannelHost().serve()
```

SDK: [`harness-sdk/nui_hitl_channel.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_hitl_channel.py)

## REST API (harness-agnostic)

Any harness, MCP tool script, or sidecar process can create and wait on HITL requests. nui sets `NUI_API_URL`, `NUI_SESSION_ID`, and `NUI_RUN_ID` on harness subprocesses.

| Step | HTTP |
|------|------|
| Create | `POST /api/hitl/requests` |
| Wait | `GET /api/hitl/requests/:id/wait` |
| Respond | `POST /api/hitl/requests/:id/respond` |
| List pending | `GET /api/hitl/requests?pending=true` |

### Create request (minimal)

```json
{
  "sessionId": "session-uuid",
  "runId": "run-uuid",
  "kind": "question",
  "payload": {
    "title": "Confirm",
    "message": "Proceed?",
    "questions": [
      {"question": "Continue?", "options": ["Yes", "No"]}
    ]
  },
  "routing": {
    "channels": ["nui-ui", "ext:hitl-demo/demo-slack"]
  }
}
```

### Respond

```json
{
  "status": "answered",
  "answers": [{"question": "Continue?", "answer": "Yes"}],
  "respondedBy": {"channel": "nui-ui"}
}
```

## Python SDK helpers

[`harness-sdk/nui_hitl.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_hitl.py):

```python
from nui_hitl import ask_user, request_approval, create_request, wait_response

# Blocking helpers (use env vars for session/run context)
answer = ask_user(questions=[{
    "question": "Which region?",
    "options": ["us-east", "eu-west"],
}])

approved = request_approval(message="Delete all staging data?")

# Lower-level
req = create_request(
    kind="question",
    payload={"title": "Pick", "questions": []},
    channels=["nui-ui"],
)
result = wait_response(req["requestId"])
```

### From extension harnesses

```python
from nui_agent_stdio import NuiAgent

class MyAgent(NuiAgent):
    def run(self, message, run_id, **kwargs):
        if not self.request_approval(message="Run expensive operation?"):
            yield "Cancelled."
            return
        yield "Running…"
```

`NuiAgent.ask_user()` emits `harness.event` with `type: hitl_request` for UI rendering.

### From MCP tool scripts

```python
#!/usr/bin/env python3
import json, sys, os
sys.path.insert(0, os.path.expanduser("~/.nui/harness-sdk"))
from nui_hitl import ask_user

args = json.load(sys.stdin)
answer = ask_user(questions=[{"question": "Confirm?", "options": ["Yes", "No"]}])
print(json.dumps(answer))
```

## REST-only origin bridge

For event-bus integrations (Kafka, webhooks, Slack), declare the channel in `hitl-channels.yaml` **without** a stdio runtime. Run a sidecar that polls and responds over REST:

```python
#!/usr/bin/env python3
"""Poll pending HITL requests and forward to an external system."""

import os
import requests

API = os.environ.get("NUI_API_URL", "http://127.0.0.1:8080")
CHANNEL = "ext:hitl-demo/demo-webhook"

pending = requests.get(f"{API}/api/hitl/requests", params={"pending": "true"}).json()
for req in pending:
    channels = (req.get("routing") or {}).get("channels") or []
    if CHANNEL not in channels:
        continue
    # deliver to external system …
    # when user answers:
    # requests.post(f"{API}/api/hitl/requests/{req['requestId']}/respond", json={...})
```

Example: [`dev/extension-examples/hitl-demo/origin_bridge.py`](https://github.com/plmbr/nui/blob/main/dev/extension-examples/hitl-demo/origin_bridge.py)

## Example extension

Install and try:

```bash
nui extension add dev/extension-examples/hitl-demo
```

Includes agents, a stdio channel host, custom MCP `confirm` tool, and `origin_bridge.py`.

## Endpoints summary

| Endpoint | Description |
|----------|-------------|
| `GET /api/hitl-channels` | Built-in + extension channel ids |
| `POST /api/hitl/requests` | Create request |
| `GET /api/hitl/requests/:id/wait` | Block until answered |
| `POST /api/hitl/requests/:id/respond` | Submit answer |
| `GET /api/hitl/requests?pending=true` | List pending |
