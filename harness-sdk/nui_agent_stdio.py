"""
nui harness framework — stdio JSON-RPC variant for ~/.nui/extensions/.

nui spawns this process and communicates over stdin/stdout (newline-delimited JSON-RPC).

Example:

    from nui_agent_stdio import NuiAgent

    class EchoAgent(NuiAgent):
        name = "echo"
        version = "0.1.0"

        def run(self, message: str, run_id: str, **kwargs):
            yield f"You said: {message}"

    if __name__ == "__main__":
        EchoAgent().serve_stdio()
"""

import json
import sys
import uuid
from typing import Generator


class NuiAgent:
    name: str = "nui-agent"
    version: str = "0.1.0"
    harness_id: str = ""

    def run(self, message: str, run_id: str, **kwargs) -> Generator[str, None, None]:
        raise NotImplementedError

    def on_cancel(self, run_id: str) -> None:
        _ = run_id

    def on_shutdown(self) -> None:
        pass

    def get_session_id(self, run_id: str) -> str:
        _ = run_id
        return ""

    def ask_user(
        self,
        *,
        questions=None,
        title: str = "",
        message: str = "",
        session_id: str = "",
        run_id: str = "",
        routing=None,
    ) -> dict:
        from nui_hitl import ask_user as hitl_ask_user

        return hitl_ask_user(
            questions=questions,
            title=title,
            message=message,
            session_id=session_id,
            run_id=run_id,
            routing=routing,
            emit_event=self._emit_hitl_request,
        )

    def request_approval(
        self,
        *,
        title: str = "",
        message: str = "",
        tool_name: str = "",
        tool_input=None,
        description: str = "",
        session_id: str = "",
        run_id: str = "",
        routing=None,
    ) -> dict:
        from nui_hitl import request_approval as hitl_request_approval

        return hitl_request_approval(
            title=title,
            message=message,
            tool_name=tool_name,
            tool_input=tool_input,
            description=description,
            session_id=session_id,
            run_id=run_id,
            routing=routing,
            emit_event=self._emit_hitl_request,
        )

    def _emit_hitl_request(self, req: dict) -> None:
        emit = getattr(self, "_emit_event", None)
        if not emit:
            return
        emit({
            "type": "hitl_request",
            "requestId": req.get("requestId", ""),
            "sessionId": req.get("sessionId", ""),
            "runId": req.get("runId", ""),
            "kind": req.get("kind", ""),
            "payload": req.get("payload") or {},
            "status": req.get("status", ""),
            "expiresAt": req.get("expiresAt", ""),
        })

    def serve_stdio(self) -> None:
        self.harness_id = __import__("os").environ.get("NUI_HARNESS_ID", self.name)
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            try:
                req = json.loads(line)
            except json.JSONDecodeError:
                continue
            self._dispatch(req)

    def _write(self, msg: dict) -> None:
        sys.stdout.write(json.dumps(msg) + "\n")
        sys.stdout.flush()

    def _dispatch(self, req: dict) -> None:
        method = req.get("method", "")
        params = req.get("params") or {}
        rid = req.get("id")

        if method == "harness.info":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {
                    "id": self.harness_id,
                    "name": self.name,
                    "version": self.version,
                    "capabilities": ["run", "cancel", "shutdown"],
                },
            })
            return

        if method == "harness.run":
            message = params.get("message", "")
            run_id = params.get("runId") or str(uuid.uuid4())
            extra = {k: v for k, v in params.items() if k not in ("message", "runId")}
            self._emit_event = lambda event: self._write({
                "jsonrpc": "2.0",
                "method": "harness.event",
                "params": {"runId": run_id, **event},
            })
            try:
                for chunk in self.run(message, run_id, **extra):
                    if isinstance(chunk, dict):
                        event_params = {"runId": run_id, **chunk}
                    else:
                        event_params = {"runId": run_id, "type": "text", "content": chunk}
                    self._write({
                        "jsonrpc": "2.0",
                        "method": "harness.event",
                        "params": event_params,
                    })
            except Exception as e:
                self._write({
                    "jsonrpc": "2.0",
                    "method": "harness.event",
                    "params": {"runId": run_id, "type": "error", "error": str(e)},
                })
            finally:
                self._emit_event = None
            done_params = {"runId": run_id, "type": "done"}
            sid = self.get_session_id(run_id)
            if sid:
                done_params["sessionId"] = sid
            self._write({"jsonrpc": "2.0", "method": "harness.event", "params": done_params})
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"runId": run_id}})
            return

        if method == "harness.cancel":
            self.on_cancel(params.get("runId", ""))
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})
            return

        if method == "harness.shutdown":
            self.on_shutdown()
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})
            sys.exit(0)

        self._write({
            "jsonrpc": "2.0",
            "id": rid,
            "error": {"code": -32601, "message": f"method not found: {method}"},
        })
