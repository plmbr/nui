#!/usr/bin/env python3
"""In-memory storage backend for the storage-demo extension."""

from __future__ import annotations

from typing import Any

from nui_storage import NuiStorageHandler

_SESSIONS: dict[str, dict[str, Any]] = {}
_AGENT_MEMORY: dict[str, str] = {}
_USER_MEMORY = ""


def _session_key(session_id: str, agent_type: str) -> str:
    return f"{session_id}:{agent_type}"


class InMemoryStorage(NuiStorageHandler):
    def read_session(
        self,
        handler_id: str,
        session_id: str = "",
        agent_type: str = "",
        working_dir: str = "",
        **kwargs: Any,
    ) -> dict[str, Any]:
        _ = handler_id, working_dir, kwargs
        row = _SESSIONS.get(_session_key(session_id, agent_type), {})
        return {
            "messages": row.get("messages", []),
            "agentSessionId": row.get("agentSessionId", ""),
        }

    def write_session(
        self,
        handler_id: str,
        session_id: str = "",
        agent_type: str = "",
        agent_session_id: str = "",
        working_dir: str = "",
        messages: list[dict[str, Any]] | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        _ = handler_id, working_dir, kwargs
        _SESSIONS[_session_key(session_id, agent_type)] = {
            "messages": messages or [],
            "agentSessionId": agent_session_id,
        }
        return {"ok": True}

    def delete_session(
        self,
        handler_id: str,
        session_id: str = "",
        agent_type: str = "",
        agent_session_id: str = "",
        working_dir: str = "",
        **kwargs: Any,
    ) -> dict[str, Any]:
        _ = handler_id, agent_session_id, working_dir, kwargs
        _SESSIONS.pop(_session_key(session_id, agent_type), None)
        return {"ok": True}

    def read_agent_memory(self, handler_id: str, agent_id: str = "", **kwargs: Any) -> dict[str, Any]:
        _ = handler_id, kwargs
        return {"content": _AGENT_MEMORY.get(agent_id, "")}

    def write_agent_memory(
        self,
        handler_id: str,
        agent_id: str = "",
        content: str = "",
        write_mode: str = "replace",
        **kwargs: Any,
    ) -> dict[str, Any]:
        _ = handler_id, kwargs
        if write_mode == "append" and _AGENT_MEMORY.get(agent_id):
            _AGENT_MEMORY[agent_id] = _AGENT_MEMORY[agent_id].rstrip() + "\n\n" + content.strip()
        else:
            _AGENT_MEMORY[agent_id] = content
        return {"ok": True}

    def delete_agent_memory(self, handler_id: str, agent_id: str = "", **kwargs: Any) -> dict[str, Any]:
        _ = handler_id, kwargs
        _AGENT_MEMORY.pop(agent_id, None)
        return {"ok": True}

    def read_user_memory(self, handler_id: str, **kwargs: Any) -> dict[str, Any]:
        _ = handler_id, kwargs
        return {"content": _USER_MEMORY}

    def write_user_memory(
        self,
        handler_id: str,
        content: str = "",
        write_mode: str = "replace",
        **kwargs: Any,
    ) -> dict[str, Any]:
        _ = handler_id, kwargs
        global _USER_MEMORY
        if write_mode == "append" and _USER_MEMORY:
            _USER_MEMORY = _USER_MEMORY.rstrip() + "\n\n" + content.strip()
        else:
            _USER_MEMORY = content
        return {"ok": True}

    def delete_user_memory(self, handler_id: str, **kwargs: Any) -> dict[str, Any]:
        _ = handler_id, kwargs
        global _USER_MEMORY
        _USER_MEMORY = ""
        return {"ok": True}


if __name__ == "__main__":
    InMemoryStorage().serve()
