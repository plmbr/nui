"""
nui extension HITL channel — stdio JSON-RPC.

Subclass NuiHITLChannelProvider and override on_deliver, then call serve().

Example:

    from nui_hitl_channel import NuiHITLChannelProvider

    class DemoChannel(NuiHITLChannelProvider):
        def on_deliver(self, channel_id, request, **kwargs):
            print(f"deliver {request['requestId']} to {channel_id}", file=sys.stderr)
            return {"ok": True}

    if __name__ == "__main__":
        DemoChannel().serve()
"""

import json
import os
import sys
from typing import Any


class NuiHITLChannelProvider:
    api_version = "nui.plmbr.dev/extension/v1"
    name = "hitl-channel"
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

    def on_deliver(self, channel_id: str, request: dict[str, Any], **kwargs: Any) -> dict[str, Any]:
        return {"ok": True}

    def _write(self, msg: dict) -> None:
        sys.stdout.write(json.dumps(msg) + "\n")
        sys.stdout.flush()

    def _dispatch(self, req: dict) -> None:
        method = req.get("method", "")
        rid = req.get("id")
        params = req.get("params") or {}

        if method == "hitl.info":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {
                    "id": os.environ.get("NUI_HITL_CHANNEL_ID", self.name),
                    "name": self.name,
                    "version": self.version,
                    "capabilities": ["deliver"],
                },
            })
            return

        if method == "hitl.deliver":
            result = self.on_deliver(
                channel_id=params.get("channelId") or "",
                request=params.get("request") or {},
                working_dir=params.get("workingDir") or "",
                session_id=params.get("sessionId") or "",
            )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "hitl.shutdown":
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})
            sys.exit(0)

        if rid is not None:
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "error": {"code": -32601, "message": f"Method not found: {method}"},
            })
