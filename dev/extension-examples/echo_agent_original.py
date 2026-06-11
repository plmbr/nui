#!/usr/bin/env python3
"""
Minimal Loop extension harness — echo agent example.

Protocol: JSON-RPC 2.0 over TCP. On startup, binds a random local port and
writes ~/.loop/extensions/echo-agent.json so Loop can find (or reconnect to) it.

Methods Loop calls:
  harness.info()            -> {name, version, capabilities}
  harness.run({message, runId})  -> streams harness.event notifications, then result
  harness.cancel({runId})   -> {ok: true}

No third-party dependencies — stdlib only.
"""

import json
import os
import socket
import sys
import threading
import uuid
from pathlib import Path

NAME = "echo-agent"
VERSION = "0.1.0"


def _connection_file() -> Path:
    path = Path.home() / ".loop" / "extensions" / f"{NAME}.json"
    path.parent.mkdir(parents=True, exist_ok=True)
    return path


def _send(conn: socket.socket, msg: dict):
    conn.sendall((json.dumps(msg) + "\n").encode())


def _handle(conn: socket.socket, req: dict):
    method = req.get("method", "")
    params = req.get("params") or {}
    rid = req.get("id")

    if method == "harness.info":
        _send(conn, {
            "jsonrpc": "2.0", "id": rid,
            "result": {"name": NAME, "version": VERSION, "capabilities": ["run", "cancel"]},
        })

    elif method == "harness.run":
        message = params.get("message", "")
        run_id = params.get("runId") or str(uuid.uuid4())

        # Stream text back as a JSON-RPC notification (no "id")
        _send(conn, {
            "jsonrpc": "2.0", "method": "harness.event",
            "params": {"runId": run_id, "type": "text", "content": f"Echo: {message}\n"},
        })
        _send(conn, {
            "jsonrpc": "2.0", "method": "harness.event",
            "params": {"runId": run_id, "type": "done"},
        })
        _send(conn, {"jsonrpc": "2.0", "id": rid, "result": {"runId": run_id}})

    elif method == "harness.cancel":
        _send(conn, {"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})

    else:
        _send(conn, {
            "jsonrpc": "2.0", "id": rid,
            "error": {"code": -32601, "message": f"method not found: {method}"},
        })


def _client_loop(conn: socket.socket, addr):
    log(f"client connected: {addr}")
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
                    _handle(conn, json.loads(line))
                except Exception as e:
                    log(f"error handling request: {e}")
    finally:
        conn.close()
        log(f"client disconnected: {addr}")


def log(msg: str):
    print(f"[{NAME}] {msg}", file=sys.stderr, flush=True)


def main():
    session_id = str(uuid.uuid4())

    srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    srv.bind(("127.0.0.1", 0))
    srv.listen(5)
    port = srv.getsockname()[1]

    conn_file = _connection_file()
    conn_file.write_text(json.dumps({
        "host": "127.0.0.1",
        "port": port,
        "session_id": session_id,
        "pid": os.getpid(),
    }))

    log(f"listening on 127.0.0.1:{port} (session {session_id})")
    log(f"connection file: {conn_file}")

    try:
        while True:
            conn, addr = srv.accept()
            threading.Thread(target=_client_loop, args=(conn, addr), daemon=True).start()
    except KeyboardInterrupt:
        log("shutting down")
    finally:
        conn_file.unlink(missing_ok=True)
        srv.close()


if __name__ == "__main__":
    main()
