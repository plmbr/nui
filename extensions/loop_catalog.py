"""
Loop extension catalog provider — stdio JSON-RPC.

Subclass LoopCatalog and override list methods, then call serve().

Example:

    from loop_catalog import LoopCatalog

    class MyCatalog(LoopCatalog):
        def list_harnesses(self):
            return [{"id": "echo", "displayName": "Echo"}]

    if __name__ == "__main__":
        MyCatalog().serve()
"""

import json
import os
import sys
from typing import Any


class LoopCatalog:
    api_version = "loop.dev/extension/v1"

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

    def list_harnesses(self) -> list[dict[str, Any]]:
        return []

    def list_mcp_servers(self) -> list[dict[str, Any]]:
        return []

    def list_skills(self) -> list[dict[str, Any]]:
        return []

    def list_agents(self) -> list[dict[str, Any]]:
        return []

    def _write(self, msg: dict) -> None:
        sys.stdout.write(json.dumps(msg) + "\n")
        sys.stdout.flush()

    def _dispatch(self, req: dict) -> None:
        method = req.get("method", "")
        rid = req.get("id")

        if method == "extension.initialize":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {
                    "apiVersion": self.api_version,
                    "extensionName": os.environ.get("LOOP_EXTENSION_NAME", ""),
                    "capabilities": ["harnesses", "mcpServers", "skills", "agents"],
                },
            })
            return

        if method == "extension.listHarnesses":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {"harnesses": self.list_harnesses()},
            })
            return

        if method == "extension.listMCPServers":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {"mcpServers": self.list_mcp_servers()},
            })
            return

        if method == "extension.listSkills":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {"skills": self.list_skills()},
            })
            return

        if method == "extension.listAgents":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {"agents": self.list_agents()},
            })
            return

        if method == "extension.shutdown":
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})
            sys.exit(0)

        self._write({
            "jsonrpc": "2.0",
            "id": rid,
            "error": {"code": -32601, "message": f"method not found: {method}"},
        })
