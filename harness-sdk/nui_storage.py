"""
nui extension storage handler — stdio JSON-RPC.

Subclass NuiStorageHandler and override read/write/delete methods, then call serve().

Example:

    from nui_storage import NuiStorageHandler

    class MyStorage(NuiStorageHandler):
        def read_user_memory(self, handler_id: str, **kwargs) -> dict:
            return {"content": "notes"}

    if __name__ == "__main__":
        MyStorage().serve()
"""

import json
import os
import sys
from typing import Any


class NuiStorageHandler:
    api_version = "nui.dev/extension/v1"
    name = "storage-handler"
    version = "1.0.0"

    def serve(self) -> None:
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            try:
                req = json.loads(line)
            except json.JSONDecodeError:
                continue
            self._dispatch(req)

    # --- session history ---

    def read_session(
        self,
        handler_id: str,
        session_id: str = "",
        agent_type: str = "",
        working_dir: str = "",
        **kwargs: Any,
    ) -> dict[str, Any]:
        _ = handler_id, session_id, agent_type, working_dir, kwargs
        return {"messages": [], "agentSessionId": ""}

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
        _ = handler_id, session_id, agent_type, agent_session_id, working_dir, messages, kwargs
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
        _ = handler_id, session_id, agent_type, agent_session_id, working_dir, kwargs
        return {"ok": True}

    # --- agent memory ---

    def read_agent_memory(self, handler_id: str, agent_id: str = "", **kwargs: Any) -> dict[str, Any]:
        _ = handler_id, agent_id, kwargs
        return {"content": ""}

    def write_agent_memory(
        self,
        handler_id: str,
        agent_id: str = "",
        content: str = "",
        write_mode: str = "replace",
        **kwargs: Any,
    ) -> dict[str, Any]:
        _ = handler_id, agent_id, content, write_mode, kwargs
        return {"ok": True}

    def delete_agent_memory(self, handler_id: str, agent_id: str = "", **kwargs: Any) -> dict[str, Any]:
        _ = handler_id, agent_id, kwargs
        return {"ok": True}

    # --- user memory ---

    def read_user_memory(self, handler_id: str, **kwargs: Any) -> dict[str, Any]:
        _ = handler_id, kwargs
        return {"content": ""}

    def write_user_memory(
        self,
        handler_id: str,
        content: str = "",
        write_mode: str = "replace",
        **kwargs: Any,
    ) -> dict[str, Any]:
        _ = handler_id, content, write_mode, kwargs
        return {"ok": True}

    def delete_user_memory(self, handler_id: str, **kwargs: Any) -> dict[str, Any]:
        _ = handler_id, kwargs
        return {"ok": True}

    def _write(self, msg: dict[str, Any]) -> None:
        sys.stdout.write(json.dumps(msg) + "\n")
        sys.stdout.flush()

    def _dispatch(self, req: dict[str, Any]) -> None:
        method = req.get("method", "")
        rid = req.get("id")
        params = req.get("params") or {}

        if method == "storage.info":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {
                    "id": os.environ.get("NUI_STORAGE_HANDLER_ID", self.name),
                    "name": self.name,
                    "version": self.version,
                },
            })
            return

        if method == "storage.session.read":
            result = self.read_session(
                handler_id=params.get("handlerId") or "",
                session_id=params.get("sessionId") or "",
                agent_type=params.get("agentType") or "",
                working_dir=params.get("workingDir") or "",
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "storage.session.write":
            result = self.write_session(
                handler_id=params.get("handlerId") or "",
                session_id=params.get("sessionId") or "",
                agent_type=params.get("agentType") or "",
                agent_session_id=params.get("agentSessionId") or "",
                working_dir=params.get("workingDir") or "",
                messages=params.get("messages") or [],
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "storage.session.delete":
            result = self.delete_session(
                handler_id=params.get("handlerId") or "",
                session_id=params.get("sessionId") or "",
                agent_type=params.get("agentType") or "",
                agent_session_id=params.get("agentSessionId") or "",
                working_dir=params.get("workingDir") or "",
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "storage.agentMemory.read":
            result = self.read_agent_memory(
                handler_id=params.get("handlerId") or "",
                agent_id=params.get("agentId") or "",
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "storage.agentMemory.write":
            result = self.write_agent_memory(
                handler_id=params.get("handlerId") or "",
                agent_id=params.get("agentId") or "",
                content=params.get("content") or "",
                write_mode=params.get("writeMode") or "replace",
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "storage.agentMemory.delete":
            result = self.delete_agent_memory(
                handler_id=params.get("handlerId") or "",
                agent_id=params.get("agentId") or "",
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "storage.userMemory.read":
            result = self.read_user_memory(handler_id=params.get("handlerId") or "")
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "storage.userMemory.write":
            result = self.write_user_memory(
                handler_id=params.get("handlerId") or "",
                content=params.get("content") or "",
                write_mode=params.get("writeMode") or "replace",
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "storage.userMemory.delete":
            result = self.delete_user_memory(handler_id=params.get("handlerId") or "")
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "storage.shutdown":
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})
            sys.exit(0)

        if rid is not None:
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "error": {"code": -32601, "message": f"Method not found: {method}"},
            })
