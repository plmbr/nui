---
layout: docs
title: Storage handlers
subtitle: Custom persistence for session history and agent/user memory.
permalink: /docs/extensions/storage/
---

Storage handlers let extensions own persistence for three data domains. When a matching handler is installed, nui **skips built-in storage** for that scope.

| Kind | Built-in fallback | Match rule |
|------|-------------------|------------|
| `sessionHistory` | `data.json` (`sessionMessages`, `agentSessions`) | `agentTypes` on handler |
| `agentMemory` | `~/.nui/memory/agents/<id>.md` | `agentTypes` on handler |
| `userMemory` | `~/.nui/memory/user.md` | global (no `agentTypes`) |

With no handler, behavior is unchanged.

## Read / write semantics

- **Session history:** first successful handler wins on read
- **Agent / user memory:** merge non-empty content from all handlers (`\n\n` between blocks)
- **Writes / deletes:** nui updates in-memory state immediately, then **async fan-out** to all matching handlers. Extension errors are logged to stderr only — API callers still succeed.

## Manifest

```yaml
contributions:
  storage:
    source:
      file: storage-handlers.yaml
    runtime:
      command: ["python3", "storage_host.py"]
```

`storage-handlers.yaml`:

```yaml
storageHandlers:
  - id: postgres-sessions
    kind: sessionHistory
    agentTypes: ["ext:corp/reviewer", "claude-code"]
  - id: agent-notes
    kind: agentMemory
    agentTypes: ["ext:corp/reviewer"]
  - id: user-cloud
    kind: userMemory
```

Handler id in wire RPC: the `id` field from the list file.

## Wire protocol (`storage.*`)

| Method | Params | Result |
|--------|--------|--------|
| `storage.info` | `{}` | `{id}` |
| `storage.session.read` | `{handlerId, sessionId, agentType, workingDir}` | `{messages, agentSessionId}` |
| `storage.session.write` | `{handlerId, sessionId, agentType, messages, agentSessionId, workingDir}` | `{ok}` |
| `storage.session.delete` | `{handlerId, sessionId, agentType, agentSessionId, workingDir}` | `{ok}` |
| `storage.agentMemory.read` | `{handlerId, agentId}` | `{content}` |
| `storage.agentMemory.write` | `{handlerId, agentId, content, writeMode}` | `{ok}` |
| `storage.agentMemory.delete` | `{handlerId, agentId}` | `{ok}` |
| `storage.userMemory.read` | `{handlerId}` | `{content}` |
| `storage.userMemory.write` | `{handlerId, content, writeMode}` | `{ok}` |
| `storage.userMemory.delete` | `{handlerId}` | `{ok}` |
| `storage.shutdown` | `{}` | `{ok: true}` |

- `messages` — same shape as nui chat messages (role, content, tool calls, etc.)
- `writeMode` — `replace` or `append` (distinct from nui memory mode settings; modes gate **when** reads/writes happen)

## Python SDK

Framework: [`harness-sdk/nui_storage.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_storage.py)

### In-memory example (storage-demo)

```python
#!/usr/bin/env python3
from typing import Any
from nui_storage import NuiStorageHandler

_SESSIONS: dict[str, dict[str, Any]] = {}
_AGENT_MEMORY: dict[str, str] = {}
_USER_MEMORY = ""


def _session_key(session_id: str, agent_type: str) -> str:
    return f"{session_id}:{agent_type}"


class InMemoryStorage(NuiStorageHandler):
    def read_session(self, handler_id, session_id="", agent_type="", working_dir="", **kwargs):
        row = _SESSIONS.get(_session_key(session_id, agent_type), {})
        return {
            "messages": row.get("messages", []),
            "agentSessionId": row.get("agentSessionId", ""),
        }

    def write_session(
        self, handler_id, session_id="", agent_type="",
        agent_session_id="", working_dir="", messages=None, **kwargs
    ):
        _SESSIONS[_session_key(session_id, agent_type)] = {
            "messages": messages or [],
            "agentSessionId": agent_session_id,
        }
        return {"ok": True}

    def delete_session(self, handler_id, session_id="", agent_type="", **kwargs):
        _SESSIONS.pop(_session_key(session_id, agent_type), None)
        return {"ok": True}

    def read_agent_memory(self, handler_id, agent_id="", **kwargs):
        return {"content": _AGENT_MEMORY.get(agent_id, "")}

    def write_agent_memory(self, handler_id, agent_id="", content="", write_mode="replace", **kwargs):
        if write_mode == "append" and _AGENT_MEMORY.get(agent_id):
            _AGENT_MEMORY[agent_id] = _AGENT_MEMORY[agent_id].rstrip() + "\n\n" + content.strip()
        else:
            _AGENT_MEMORY[agent_id] = content
        return {"ok": True}

    def read_user_memory(self, handler_id, **kwargs):
        return {"content": _USER_MEMORY}

    def write_user_memory(self, handler_id, content="", write_mode="replace", **kwargs):
        global _USER_MEMORY
        if write_mode == "append" and _USER_MEMORY:
            _USER_MEMORY = _USER_MEMORY.rstrip() + "\n\n" + content.strip()
        else:
            _USER_MEMORY = content
        return {"ok": True}


if __name__ == "__main__":
    InMemoryStorage().serve()
```

### Postgres sketch

```python
class PostgresSessionStorage(NuiStorageHandler):
    def read_session(self, handler_id, session_id, agent_type, working_dir, **kwargs):
        with self.pool.connection() as conn:
            row = conn.execute(
                "SELECT messages, agent_session_id FROM sessions WHERE id = %s",
                (session_id,),
            ).fetchone()
        if not row:
            return {"messages": [], "agentSessionId": ""}
        return {"messages": row[0], "agentSessionId": row[1]}

    def write_session(self, handler_id, session_id, agent_type, messages, agent_session_id, **kwargs):
        with self.pool.connection() as conn:
            conn.execute(
                """
                INSERT INTO sessions (id, agent_type, messages, agent_session_id)
                VALUES (%s, %s, %s, %s)
                ON CONFLICT (id) DO UPDATE SET messages = EXCLUDED.messages,
                  agent_session_id = EXCLUDED.agent_session_id
                """,
                (session_id, agent_type, json.dumps(messages), agent_session_id),
            )
        return {"ok": True}
```

## Programmatic extensions

Override on `NuiExtension` ([`sdk/python/nui_extension/extension.py`](https://github.com/plmbr/nui/blob/main/sdk/python/nui_extension/extension.py)):

```python
def get_storage_handlers(self):
    return [
        {"id": "agent-notes", "kind": "agentMemory", "agentTypes": ["ext:myext/reviewer"]},
    ]

def read_session(self, handler_id, session_id, agent_type, working_dir, ctx=None):
    ...

def write_session(self, handler_id, session_id, agent_type, messages, agent_session_id, working_dir, ctx=None):
    ...
```

Go equivalent: [`sdk/go/nuiextension/extension.go`](https://github.com/plmbr/nui/blob/main/sdk/go/nuiextension/extension.go)

## Try the demo

```bash
nui extension add dev/extension-examples/storage-demo
```

Example: [`dev/extension-examples/storage-demo/`](https://github.com/plmbr/nui/tree/main/dev/extension-examples/storage-demo/)
