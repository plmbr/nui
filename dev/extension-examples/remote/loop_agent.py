"""
Loop extension framework — server-mode variant for Docker and remote agents.

Identical to the standard loop_agent.py except:
  - Accepts --port <n> to bind on a known port instead of a random one.
  - Binds to 0.0.0.0 when --port is given so Docker port-mapping and remote
    hosts can reach it.
  - Does NOT write a connection file when --port is given; Loop discovers the
    address itself (via `docker port` for Docker agents, or from stored config
    for remote agents).

Usage in a Dockerfile:
    CMD ["python3", "my_agent.py", "--port", "9090"]

Usage for remote:
    python3 my_agent.py --port 9000
"""

import json
import os
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
        """Called once after the server is bound."""
        _ = port

    def on_cancel(self, run_id: str) -> None:
        """Called when Loop sends harness.cancel for a running run_id."""
        _ = run_id

    def get_session_id(self, run_id: str) -> str:
        """Return session ID to include in the done event. Override if needed."""
        _ = run_id
        return ""

    # ── Internals ────────────────────────────────────────────────────────────

    def serve(self):
        """Start the TCP server. Blocks until KeyboardInterrupt or SIGTERM."""
        port = self._parse_port()
        explicit_port = port != 0
        bind_host = "0.0.0.0" if explicit_port else "127.0.0.1"

        srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        srv.bind((bind_host, port))
        srv.listen(5)
        bound_port = srv.getsockname()[1]

        if not explicit_port:
            # Standard in-process extension: write connection file so Loop can find it.
            project_id = self._parse_project_id()
            conn_file = self._write_connection_file(project_id, bound_port)
            self._log(f"connection file: {conn_file}")
        else:
            conn_file = None

        self._log(f"listening on {bind_host}:{bound_port}")
        self.on_start(bound_port)

        try:
            while True:
                conn, addr = srv.accept()
                threading.Thread(
                    target=self._client_loop, args=(conn, addr), daemon=True
                ).start()
        except KeyboardInterrupt:
            self._log("shutting down")
        finally:
            if conn_file:
                conn_file.unlink(missing_ok=True)
            srv.close()

    # ── Private ──────────────────────────────────────────────────────────────

    def _parse_port(self) -> int:
        args = sys.argv[1:]
        for i, a in enumerate(args):
            if a == "--port" and i + 1 < len(args):
                return int(args[i + 1])
        return 0

    def _parse_project_id(self) -> str:
        args = sys.argv[1:]
        for i, a in enumerate(args):
            if a == "--project-id" and i + 1 < len(args):
                return args[i + 1]
        return self.name

    def _write_connection_file(self, project_id: str, port: int) -> Path:
        path = Path.home() / ".loop" / "extensions" / f"{project_id}.json"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps({
            "host": "127.0.0.1",
            "port": port,
            "session_id": str(uuid.uuid4()),
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
                    "capabilities": ["run", "cancel"],
                },
            })

        elif method == "harness.run":
            message = params.get("message", "")
            run_id = params.get("runId") or str(uuid.uuid4())
            extra = {k: v for k, v in params.items() if k not in ("message", "runId")}
            try:
                for chunk in self.run(message, run_id, **extra):
                    self._send(conn, {
                        "jsonrpc": "2.0", "method": "harness.event",
                        "params": {"runId": run_id, "type": "text", "content": chunk},
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
