"""
Loop extension framework — HTTP/SSE variant for Docker and remote agents.

Exposes three endpoints:
  GET  /info   — agent metadata (name, version, capabilities)
  POST /run    — run the agent; returns text/event-stream SSE
  POST /cancel — cancel a running run (best-effort)

Usage in a Dockerfile:
    CMD ["python3", "my_agent.py", "--port", "9090"]

Usage for a remote agent:
    python3 my_agent.py --port 9000
"""

import json
import sys
import threading
import uuid
from http.server import BaseHTTPRequestHandler, HTTPServer
from socketserver import ThreadingMixIn
from typing import Generator


class _CancelledError(Exception):
    pass


class ThreadingHTTPServer(ThreadingMixIn, HTTPServer):
    daemon_threads = True


class LoopAgent:
    name: str = "loop-agent"
    version: str = "0.1.0"

    # ── Override this ────────────────────────────────────────────────────────

    def run(self, message: str, **kwargs) -> Generator[str, None, None]:
        """Yield text chunks to stream back to Loop. Override in subclass."""
        raise NotImplementedError

    # ── Optional hooks ───────────────────────────────────────────────────────

    def on_start(self, port: int) -> None:
        """Called once after the server is bound."""

    def on_cancel(self, run_id: str) -> None:
        """Called when Loop sends POST /cancel."""

    def get_session_id(self, run_id: str) -> str:
        """Return session ID to include in the done event. Override if needed."""
        return ""

    # ── Internals ────────────────────────────────────────────────────────────

    def serve(self):
        """Start the HTTP server. Blocks until KeyboardInterrupt or SIGTERM."""
        port = self._parse_port()
        if port == 0:
            sys.exit("error: --port <n> is required for HTTP agents")

        agent = self
        cancel_events: dict[str, threading.Event] = {}
        cancel_lock = threading.Lock()

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, fmt, *args):  # silence default access log
                pass

            def _send_json(self, status: int, body: dict):
                data = json.dumps(body).encode()
                self.send_response(status)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

            def do_GET(self):
                if self.path == "/info":
                    self._send_json(200, {
                        "name": agent.name,
                        "version": agent.version,
                        "capabilities": ["run", "cancel"],
                    })
                else:
                    self._send_json(404, {"error": "not found"})

            def do_POST(self):
                length = int(self.headers.get("Content-Length", 0))
                body = json.loads(self.rfile.read(length) or b"{}")

                if self.path == "/run":
                    self._handle_run(body)
                elif self.path == "/cancel":
                    run_id = body.get("runId", "")
                    with cancel_lock:
                        ev = cancel_events.get(run_id)
                    if ev:
                        ev.set()
                    agent.on_cancel(run_id)
                    self._send_json(200, {"ok": True})
                else:
                    self._send_json(404, {"error": "not found"})

            def _sse(self, event: dict) -> bytes:
                return ("data: " + json.dumps(event) + "\n\n").encode()

            def _handle_run(self, body: dict):
                message = body.get("message", "")
                run_id = body.get("runId") or str(uuid.uuid4())
                extra = {k: v for k, v in body.items() if k not in ("message", "runId")}

                cancel_ev = threading.Event()
                with cancel_lock:
                    cancel_events[run_id] = cancel_ev

                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.send_header("Cache-Control", "no-cache")
                self.send_header("X-Accel-Buffering", "no")
                self.end_headers()

                try:
                    for chunk in agent.run(message, **extra):
                        if cancel_ev.is_set():
                            raise _CancelledError()
                        self.wfile.write(self._sse({"type": "text", "content": chunk}))
                        self.wfile.flush()
                except _CancelledError:
                    self.wfile.write(self._sse({"type": "error", "error": "cancelled"}))
                    self.wfile.flush()
                except Exception as exc:
                    self.wfile.write(self._sse({"type": "error", "error": str(exc)}))
                    self.wfile.flush()
                else:
                    done: dict = {"type": "done"}
                    sid = agent.get_session_id(run_id)
                    if sid:
                        done["sessionId"] = sid
                    self.wfile.write(self._sse(done))
                    self.wfile.flush()
                finally:
                    with cancel_lock:
                        cancel_events.pop(run_id, None)

        server = ThreadingHTTPServer(("0.0.0.0", port), Handler)
        self._log(f"listening on 0.0.0.0:{port}")
        self.on_start(port)
        try:
            server.serve_forever()
        except KeyboardInterrupt:
            self._log("shutting down")
        finally:
            server.server_close()

    # ── Private ──────────────────────────────────────────────────────────────

    def _parse_port(self) -> int:
        args = sys.argv[1:]
        for i, a in enumerate(args):
            if a == "--port" and i + 1 < len(args):
                return int(args[i + 1])
        return 0

    def _log(self, msg: str):
        print(f"[{self.name}] {msg}", file=sys.stderr, flush=True)
