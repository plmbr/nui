"""HTTP/SSE base server for Loop Docker agents.

Subclass HttpLoopAgent, override run(), call serve(port).
Protocol matches HTTPExtensionAgent in Loop's Go backend:
  GET  /info  → {"name": "...", "version": "..."}
  POST /run   → text/event-stream
               data: {"type":"text","content":"..."}
               data: {"type":"done","sessionId":"..."}
               data: {"type":"error","error":"..."}
"""

import json
import sys
import uuid
from http.server import BaseHTTPRequestHandler, HTTPServer
from socketserver import ThreadingMixIn


class _ThreadingHTTPServer(ThreadingMixIn, HTTPServer):
    daemon_threads = True


class HttpLoopAgent:
    name: str = "loop-agent"
    version: str = "0.1.0"

    def run(self, message: str, run_id: str, **kwargs):
        """Yield text chunks to stream. Override in subclass."""
        raise NotImplementedError

    def get_session_id(self, run_id: str) -> str:
        return ""

    def serve(self, port: int = 8090):
        agent = self

        class _Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                if self.path == "/info":
                    body = json.dumps({"name": agent.name, "version": agent.version}).encode()
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.send_header("Content-Length", str(len(body)))
                    self.end_headers()
                    self.wfile.write(body)
                else:
                    self.send_response(404)
                    self.end_headers()

            def do_POST(self):
                if self.path != "/run":
                    self.send_response(404)
                    self.end_headers()
                    return

                length = int(self.headers.get("Content-Length", 0))
                params = json.loads(self.rfile.read(length)) if length else {}

                message = params.get("message", "")
                run_id = params.get("runId") or str(uuid.uuid4())
                kwargs = {k: v for k, v in params.items() if k not in ("message", "runId")}

                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.send_header("Cache-Control", "no-cache")
                self.end_headers()

                def sse(obj: dict):
                    self.wfile.write(f"data: {json.dumps(obj)}\n\n".encode())
                    self.wfile.flush()

                try:
                    for chunk in agent.run(message, run_id, **kwargs):
                        if isinstance(chunk, dict):
                            sse(chunk)
                        else:
                            sse({"type": "text", "content": chunk})
                except Exception as e:
                    sse({"type": "error", "error": str(e)})
                    return

                sse({"type": "done", "sessionId": agent.get_session_id(run_id)})

            def log_message(self, fmt, *args):
                print(f"[{agent.name}] {fmt % args}", file=sys.stderr, flush=True)

        server = _ThreadingHTTPServer(("0.0.0.0", port), _Handler)
        print(f"[{agent.name}] listening on 0.0.0.0:{port}", file=sys.stderr, flush=True)
        server.serve_forever()
