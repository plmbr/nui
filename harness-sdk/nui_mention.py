"""
nui extension mention provider — stdio JSON-RPC.

Subclass NuiMentionProvider and override list_items / resolve_value, then call serve().

Example:

    from nui_mention import NuiMentionProvider

    class MyMentions(NuiMentionProvider):
        def list_items(self, parent="", query="", limit=20, **kwargs):
            if not parent:
                return {"items": [{"label": "Runbooks", "value": "runbooks", "hasChildren": True}]}
            return {"items": [{"label": "Deploy", "value": "ext:corp-pack:corp-refs:deploy", "hasChildren": False}]}

        def resolve_value(self, value, **kwargs):
            return {"text": f"See runbook: {value}"}

    if __name__ == "__main__":
        MyMentions().serve()
"""

import json
import os
import sys
from typing import Any


class NuiMentionProvider:
    api_version = "nui.plmbr.dev/extension/v1"
    name = "mention-provider"
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

    def list_items(
        self,
        parent: str = "",
        query: str = "",
        limit: int = 20,
        **kwargs: Any,
    ) -> dict[str, Any]:
        return {"items": [], "breadcrumb": []}

    def resolve_value(self, value: str, **kwargs: Any) -> dict[str, Any]:
        return {"text": value}

    def _write(self, msg: dict) -> None:
        sys.stdout.write(json.dumps(msg) + "\n")
        sys.stdout.flush()

    def _dispatch(self, req: dict) -> None:
        method = req.get("method", "")
        rid = req.get("id")
        params = req.get("params") or {}

        if method == "mention.info":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {
                    "id": os.environ.get("NUI_MENTION_PROVIDER_ID", self.name),
                    "name": self.name,
                    "version": self.version,
                    "capabilities": ["list", "resolve"],
                },
            })
            return

        if method == "mention.list":
            result = self.list_items(
                parent=params.get("parent") or "",
                query=params.get("query") or "",
                limit=int(params.get("limit") or 20),
                provider_id=params.get("providerId") or "",
                working_dir=params.get("workingDir") or "",
                session_id=params.get("sessionId") or "",
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "mention.resolve":
            result = self.resolve_value(
                value=params.get("value") or "",
                provider_id=params.get("providerId") or "",
                working_dir=params.get("workingDir") or "",
                session_id=params.get("sessionId") or "",
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "mention.shutdown":
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})
            sys.exit(0)

        if rid is not None:
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "error": {"code": -32601, "message": f"Method not found: {method}"},
            })
