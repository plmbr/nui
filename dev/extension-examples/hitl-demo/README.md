# HITL Demo Extension

Example extension showing **harness-agnostic** human-in-the-loop (HITL) in Loop.

Install:

```sh
loop extension add dev/extension-examples/hitl-demo
```

Restart the UI or `POST /api/extensions/reload`.

## Three ways to ask a human

### 1. Built-in `loop-hitl` MCP (Claude / Codex / Pi / OpenCode)

Loop injects a `loop-hitl` MCP server when `hitl.mode` is `interactive`. Harnesses call:

- `ask_user` — structured questions (AskUserQuestion-style)
- `request_approval` — approve/reject gates

The **HITL Claude Demo** agent (`ext:hitl-demo/hitl-claude`) enables this. Start a session and ask the agent to use `ask_user` or the `confirm_action` tool.

### 2. REST from any harness (`ask_user()` in Python SDK)

Extension harnesses can call `self.ask_user()` from `loop_agent_stdio.LoopAgent`. The SDK POSTs to `LOOP_API_URL/api/hitl/requests` and blocks on `GET .../wait`.

Try the **HITL Harness Demo** agent (`ext:hitl-demo/hitl-harness`).

Environment (set automatically by Loop during runs):

| Variable | Purpose |
|----------|---------|
| `LOOP_API_URL` | Loop server base URL |
| `LOOP_SESSION_ID` | Session scope for HITL policy |
| `LOOP_RUN_ID` | Active run id |

SDK: [`harness-sdk/loop_hitl.py`](../../../harness-sdk/loop_hitl.py)

### 3. Custom MCP tool via REST

The `hitl-tools/confirm_action` tool uses `loop_hitl.request_approval()` directly — useful when your harness supports custom MCP servers but not the built-in `loop-hitl` injection.

## HITL channels

Agents declare delivery channels under `hitl.channels`:

```yaml
hitl:
  mode: interactive
  channels:
    - loop-ui
    - ext:hitl-demo/demo-slack
```

| Channel | Delivery |
|---------|----------|
| `loop-ui` | Built-in chat prompt cards |
| `ext:hitl-demo/demo-slack` | Stdio JSON-RPC stub (`hitl_channel_host.py`) |
| `ext:hitl-demo/demo-webhook` | REST-only bridge (`origin_bridge.py`) |

### Stdio channel host

`hitl_channel_host.py` implements the `hitl.*` wire protocol (see [`dev/extension-api.md`](../../extension-api.md)). Loop spawns it when the extension is loaded; on delivery it logs the request to stderr.

### REST origin bridge

For Kafka, Slack webhooks, or other event buses, skip the stdio runtime and poll/respond over REST:

```sh
LOOP_API_URL=http://127.0.0.1:8080 \
HITL_CHANNEL_ID=ext:hitl-demo/demo-webhook \
python3 origin_bridge.py
```

The bridge lists `GET /api/hitl/requests?pending=true`, filters by `routing.channels`, and posts answers to `POST /api/hitl/requests/:id/respond`.

## Agent ADL reference

```yaml
hitl:
  mode: interactive          # interactive | off | auto
  channels:
    - loop-ui
    - ext:hitl-demo/demo-slack
  ttlSeconds: 3600           # optional
```

See [`dev/extension-api.md`](../../extension-api.md) for the full HITL extension API.
