# nui Harness SDK (Python)

Python helpers for **extension authors**. nui copies selected modules to `~/.nui/harness-sdk/` on first use; the repo copy is the source of truth during development.

See also: [`dev/extension-api.md`](../dev/extension-api.md), [`dev/extension-sdk.md`](../dev/extension-sdk.md), [`dev/harness-design.md`](../dev/harness-design.md).

## Author-facing modules

| Module | Purpose |
|---|---|
| [`nui_extension.py`](nui_extension.py) | **Programmatic extension SDK** — subclass `NuiExtension`, override `get_harnesses()` / `run_harness()`, call `serve()` |
| [`nui_agent_stdio.py`](nui_agent_stdio.py) | **Declarative harness framework** — subclass `NuiAgent`, implement `run()`, call `serve_stdio()` |
| [`nui_agent.py`](nui_agent.py) | TCP JSON-RPC harness framework (reference / advanced). Used by standalone examples in `dev/harness-examples/py/`. |
| [`nui_catalog.py`](nui_catalog.py) | Multiplexed catalog RPC — list harnesses, MCP servers, skills, and agents from one stdio process. |
| [`nui_hitl.py`](nui_hitl.py) | REST client for HITL — `ask_user()`, `request_approval()`, `create_request()`, `wait_response()`, `respond()`, `list_pending()`, `api_url()`. |
| [`nui_hitl_channel.py`](nui_hitl_channel.py) | Stdio host for extension HITL delivery channels (`hitl.info`, `hitl.deliver`, `hitl.shutdown`). |
| [`nui_mention.py`](nui_mention.py) | Stdio host for `@`-mention providers (`mention.info`, `mention.list`, `mention.resolve`). |
| [`nui_mcp_tools.py`](nui_mcp_tools.py) | Stdio MCP proxy for custom extension tools (nui-managed; tool scripts read JSON from stdin). |

### `NuiAgent` (stdio)

```python
from nui_agent_stdio import NuiAgent

class EchoAgent(NuiAgent):
    def run(self, message, **ctx):
        yield message

    def on_cancel(self):
        pass

if __name__ == "__main__":
    EchoAgent("echo").serve_stdio()
```

Built-in helpers on `NuiAgent`: `get_session_id()`, `ask_user()`, `request_approval()`, `on_shutdown()`.

Wire methods: `harness.info`, `harness.run`, `harness.cancel`, `harness.shutdown`.

### `nui_hitl.py`

Used from extension harnesses and custom MCP tool scripts:

```python
from nui_hitl import ask_user, request_approval

answer = ask_user(questions=[{"question": "Proceed?", "options": ["Yes", "No"]}])
approved = request_approval(message="Deploy to production?")
```

nui sets `NUI_API_URL`, `NUI_SESSION_ID`, and `NUI_RUN_ID` during runs.

## Internal modules (not for extension authors)

These wrap builtin CLI harnesses inside Docker sandbox images and are **not** part of the extension author API:

- `claude_code.py`, `codex.py`, `opencode.py`, `pi.py` — harness entrypoints
- `*_session.py`, `*_stream.py` — persistent session and stream parsers

## Auto-install behavior

nui copies SDK files to `~/.nui/harness-sdk/` when an extension feature first needs them:

| Trigger | Files installed |
|---|---|
| HITL (`hitl_sdk.go`) | `nui_hitl.py`, `nui_hitl_channel.py`, `nui_agent_stdio.py` |
| Mention providers (`mention_sdk.go`) | `nui_mention.py` |
| Custom MCP tools (`custom_mcp.go`) | `nui_mcp_tools.py` |

Environment overrides:

| Variable | Effect |
|---|---|
| `NUI_HITL_SDK_DIR` | Directory containing HITL SDK files |
| `NUI_MENTION_SDK_DIR` | Directory containing `nui_mention.py` |
| `NUI_MCP_TOOLS_PATH` | Path to `nui_mcp_tools.py` |

During extension harness runs, nui also sets `NUI_HITL_SDK_DIR` on the host process environment when HITL is enabled.

## TypeScript reference

Partial TS stubs live in [`dev/extension-examples/ts/`](../dev/extension-examples/ts/). Python is the canonical SDK for extension harnesses.
