"""
Loop extension framework.

Subclass LoopAgent, override `run()`, call `serve()`. Everything else
(TCP socket, connection file, JSON-RPC dispatch, streaming) is handled here.

Example:

    from loop_agent import LoopAgent

    class MyAgent(LoopAgent):
        name = "my-agent"
        version = "0.1.0"

        def run(self, message: str, run_id: str, **kwargs):
            yield "Thinking..."
            yield f"\\n\\nYou said: {message}\\n"

    if __name__ == "__main__":
        MyAgent().serve()
"""

import json
import os
import signal
import socket
import sys
import threading
import uuid
from pathlib import Path
from typing import Generator


class LoopAgent:
    name: str = "loop-agent"
    version: str = "0.1.0"

    # ── Override this ────────────────────────────────────────────────────────

    def run(self, message: str, run_id: str, **kwargs) -> Generator[str, None, None]:
        """Yield text chunks to stream back to Loop. Override in subclass."""
        raise NotImplementedError

    # ── Optional hooks ───────────────────────────────────────────────────────

    def on_start(self, port: int) -> None:
        """Called once after the server is bound and the connection file written."""
        _ = port

    def on_cancel(self, run_id: str) -> None:
        """Called when Loop sends harness.cancel for a running run_id."""
        _ = run_id

    def on_shutdown(self) -> None:
        """Called before the extension process exits (RPC, signal, or interrupt)."""
        pass

    def get_session_id(self, run_id: str) -> str:
        """Return session ID to include in the done event. Override if needed."""
        _ = run_id
        return ""

    # ── Internals ────────────────────────────────────────────────────────────

    def serve(self):
        """Start the TCP server. Blocks until shutdown."""
        # --project-id <id> is passed by the Go manager to scope this process to one project.
        self._project_id = self._parse_project_id()
        self._shutdown_lock = threading.Lock()
        self._shutdown_done = False
        session_id = str(uuid.uuid4())

        srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        srv.bind(("127.0.0.1", 0))
        srv.listen(5)
        port = srv.getsockname()[1]

        conn_file = self._write_connection_file(port, session_id)
        self._log(f"listening on 127.0.0.1:{port}")
        self._log(f"connection file: {conn_file}")
        self.on_start(port)

        def request_shutdown(reason: str) -> None:
            self._log(reason)
            try:
                srv.close()
            except OSError:
                pass

        signal.signal(signal.SIGTERM, lambda _s, _f: request_shutdown("received SIGTERM"))
        signal.signal(signal.SIGINT, lambda _s, _f: request_shutdown("received SIGINT"))

        try:
            while True:
                try:
                    conn, addr = srv.accept()
                except OSError:
                    break
                threading.Thread(
                    target=self._client_loop, args=(conn, addr), daemon=True
                ).start()
        finally:
            self._shutdown_once()
            conn_file.unlink(missing_ok=True)
            try:
                srv.close()
            except OSError:
                pass

    def _shutdown_once(self) -> None:
        with self._shutdown_lock:
            if self._shutdown_done:
                return
            self._shutdown_done = True
        self.on_shutdown()

    # ── Private ──────────────────────────────────────────────────────────────

    def _parse_project_id(self) -> str:
        args = sys.argv[1:]
        for i, a in enumerate(args):
            if a == "--project-id" and i + 1 < len(args):
                return args[i + 1]
        return self.name

    def _write_connection_file(self, port: int, session_id: str) -> Path:
        path = Path.home() / ".loop" / "extensions" / f"{self._project_id}.json"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps({
            "host": "127.0.0.1",
            "port": port,
            "session_id": session_id,
            "pid": os.getpid(),
        }))
        return path

    def _send(self, conn: socket.socket, msg: dict):
        conn.sendall((json.dumps(msg) + "\n").encode())

    def _dispatch(self, conn: socket.socket, req: dict):
        method = req.get("method", "")
        params = req.get("params") or {}
        rid = req.get("id")

        if method == "harness.info":
            self._send(conn, {
                "jsonrpc": "2.0", "id": rid,
                "result": {
                    "name": self.name,
                    "version": self.version,
                    "capabilities": ["run", "cancel", "shutdown"],
                },
            })

        elif method == "harness.run":
            message = params.get("message", "")
            run_id = params.get("runId") or str(uuid.uuid4())
            extra = {k: v for k, v in params.items() if k not in ("message", "runId")}
            try:
                for chunk in self.run(message, run_id, **extra):
                    if isinstance(chunk, dict):
                        event_params = {"runId": run_id, **chunk}
                    else:
                        event_params = {"runId": run_id, "type": "text", "content": chunk}
                    self._send(conn, {
                        "jsonrpc": "2.0", "method": "harness.event",
                        "params": event_params,
                    })
            except Exception as e:
                self._send(conn, {
                    "jsonrpc": "2.0", "method": "harness.event",
                    "params": {"runId": run_id, "type": "error", "error": str(e)},
                })
            done_params: dict = {"runId": run_id, "type": "done"}
            sid = self.get_session_id(run_id)
            if sid:
                done_params["sessionId"] = sid
            self._send(conn, {"jsonrpc": "2.0", "method": "harness.event", "params": done_params})
            self._send(conn, {"jsonrpc": "2.0", "id": rid, "result": {"runId": run_id}})

        elif method == "harness.cancel":
            run_id = params.get("runId", "")
            self.on_cancel(run_id)
            self._send(conn, {"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})

        elif method == "harness.shutdown":
            self._shutdown_once()
            self._send(conn, {"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})

        else:
            self._send(conn, {
                "jsonrpc": "2.0", "id": rid,
                "error": {"code": -32601, "message": f"method not found: {method}"},
            })

    def _client_loop(self, conn: socket.socket, addr):
        self._log(f"client connected: {addr}")
        buf = ""
        try:
            while True:
                chunk = conn.recv(4096).decode()
                if not chunk:
                    break
                buf += chunk
                while "\n" in buf:
                    line, buf = buf.split("\n", 1)
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        self._dispatch(conn, json.loads(line))
                    except Exception as e:
                        self._log(f"dispatch error: {e}")
        finally:
            conn.close()
            self._log(f"client disconnected: {addr}")

    def _log(self, msg: str):
        print(f"[{self.name}] {msg}", file=sys.stderr, flush=True)
