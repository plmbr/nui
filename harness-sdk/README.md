# Loop Harness SDK (Python)

Python helpers for **extension authors**. Loop copies selected modules to `~/.loop/harness-sdk/` on first use; the repo copy is the source of truth during development.

See also: [`dev/extension-api.md`](../dev/extension-api.md), [`dev/extension-sdk.md`](../dev/extension-sdk.md), [`dev/harness-design.md`](../dev/harness-design.md).

## Author-facing modules

| Module | Purpose |
|---|---|
| [`loop_extension.py`](loop_extension.py) | **Programmatic extension SDK** — subclass `LoopExtension`, override `get_harnesses()` / `run_harness()`, call `serve()` |
| [`loop_agent_stdio.py`](loop_agent_stdio.py) | **Declarative harness framework** — subclass `LoopAgent`, implement `run()`, call `serve_stdio()` |
| [`loop_agent.py`](loop_agent.py) | TCP JSON-RPC harness framework (reference / advanced). Used by standalone examples in `dev/harness-examples/py/`. |
| [`loop_catalog.py`](loop_catalog.py) | Multiplexed catalog RPC — list harnesses, MCP servers, skills, and agents from one stdio process. |
| [`loop_hitl.py`](loop_hitl.py) | REST client for HITL — `ask_user()`, `request_approval()`, `create_request()`, `wait_response()`, `respond()`, `list_pending()`, `api_url()`. |
| [`loop_hitl_channel.py`](loop_hitl_channel.py) | Stdio host for extension HITL delivery channels (`hitl.info`, `hitl.deliver`, `hitl.shutdown`). |
| [`loop_mention.py`](loop_mention.py) | Stdio host for `@`-mention providers (`mention.info`, `mention.list`, `mention.resolve`). |
| [`loop_mcp_tools.py`](loop_mcp_tools.py) | Stdio MCP proxy for custom extension tools (Loop-managed; tool scripts read JSON from stdin). |

### `LoopAgent` (stdio)

```python
from loop_agent_stdio import LoopAgent

class EchoAgent(LoopAgent):
    def run(self, message, **ctx):
        yield message

    def on_cancel(self):
        pass

if __name__ == "__main__":
    EchoAgent("echo").serve_stdio()
```

Built-in helpers on `LoopAgent`: `get_session_id()`, `ask_user()`, `request_approval()`, `on_shutdown()`.

Wire methods: `harness.info`, `harness.run`, `harness.cancel`, `harness.shutdown`.

### `loop_hitl.py`

Used from extension harnesses and custom MCP tool scripts:

```python
from loop_hitl import ask_user, request_approval

answer = ask_user(questions=[{"question": "Proceed?", "options": ["Yes", "No"]}])
approved = request_approval(message="Deploy to production?")
```

Loop sets `LOOP_API_URL`, `LOOP_SESSION_ID`, and `LOOP_RUN_ID` during runs.

## Internal modules (not for extension authors)

These wrap builtin CLI harnesses inside Docker sandbox images and are **not** part of the extension author API:

- `claude_code.py`, `codex.py`, `opencode.py`, `pi.py` — harness entrypoints
- `*_session.py`, `*_stream.py` — persistent session and stream parsers

## Auto-install behavior

Loop copies SDK files to `~/.loop/harness-sdk/` when an extension feature first needs them:

| Trigger | Files installed |
|---|---|
| HITL (`hitl_sdk.go`) | `loop_hitl.py`, `loop_hitl_channel.py`, `loop_agent_stdio.py` |
| Mention providers (`mention_sdk.go`) | `loop_mention.py` |
| Custom MCP tools (`custom_mcp.go`) | `loop_mcp_tools.py` |

Environment overrides:

| Variable | Effect |
|---|---|
| `LOOP_HITL_SDK_DIR` | Directory containing HITL SDK files |
| `LOOP_MENTION_SDK_DIR` | Directory containing `loop_mention.py` |
| `LOOP_MCP_TOOLS_PATH` | Path to `loop_mcp_tools.py` |

During extension harness runs, Loop also sets `LOOP_HITL_SDK_DIR` on the host process environment when HITL is enabled.

## TypeScript reference

Partial TS stubs live in [`dev/extension-examples/ts/`](../dev/extension-examples/ts/). Python is the canonical SDK for extension harnesses.
