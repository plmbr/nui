#!/usr/bin/env python3
"""
Sample Loop extension client.

Reads ~/.loop/extensions/<name>.json, connects over TCP, and calls
harness.info then harness.run, printing streamed events as they arrive.

Usage:
  # terminal 1 — start the agent
  python echo_agent.py

  # terminal 2 — run this client
  python client.py [extension-name]   # default: echo-agent
"""

import json
import socket
import sys
import uuid
from pathlib import Path


def load_connection(name: str) -> dict:
    path = Path.home() / ".loop" / "extensions" / f"{name}.json"
    if not path.exists():
        sys.exit(f"connection file not found: {path}\nIs the extension running?")
    return json.loads(path.read_text())


def connect(info: dict) -> socket.socket:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((info["host"], info["port"]))
    return sock


class Client:
    def __init__(self, sock: socket.socket):
        self._sock = sock
        self._buf = ""
        self._next_id = 1

    def _send(self, method: str, params: dict) -> int:
        rid = self._next_id
        self._next_id += 1
        msg = {"jsonrpc": "2.0", "id": rid, "method": method, "params": params}
        self._sock.sendall((json.dumps(msg) + "\n").encode())
        return rid

    def _read_message(self) -> dict:
        while "\n" not in self._buf:
            chunk = self._sock.recv(4096).decode()
            if not chunk:
                raise ConnectionError("server closed connection")
            self._buf += chunk
        line, self._buf = self._buf.split("\n", 1)
        return json.loads(line.strip())

    def call(self, method: str, params: dict = {}) -> dict:
        """Send a request and return the result (skips notifications)."""
        rid = self._send(method, params)
        while True:
            msg = self._read_message()
            if msg.get("id") == rid:
                if "error" in msg:
                    raise RuntimeError(msg["error"])
                return msg["result"]
            # notification — yield back to caller via on_event if set
            if "method" in msg:
                self._on_notification(msg)

    def run(self, message: str, on_event=None):
        """Call harness.run and yield streaming events until done."""
        self._pending_events = []
        self._on_event_cb = on_event
        run_id = str(uuid.uuid4())
        result = self.call("harness.run", {"message": message, "runId": run_id})
        return result

    def _on_notification(self, msg: dict):
        if self._on_event_cb:
            self._on_event_cb(msg.get("params", {}))


def main():
    name = sys.argv[1] if len(sys.argv) > 1 else "echo-agent"

    print(f"connecting to extension: {name}")
    info = load_connection(name)
    print(f"  host={info['host']} port={info['port']} pid={info['pid']}")

    sock = connect(info)
    client = Client(sock)

    # 1. Get extension info
    ext_info = client.call("harness.info")
    print(f"\nharness.info → {json.dumps(ext_info, indent=2)}")

    # 2. Run with streaming events
    print("\nharness.run →")

    def on_event(event: dict):
        t = event.get("type")
        if t == "text":
            print(event.get("content", ""), end="", flush=True)
        elif t == "done":
            print()  # newline after stream ends

    result = client.run("Hello from the Loop client!", on_event=on_event)
    print(f"result: {result}")

    sock.close()


if __name__ == "__main__":
    main()
