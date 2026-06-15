#!/usr/bin/env python3
"""Claude Code agent for Loop Docker container.

Same logic as extensions/claude_code.py but speaks HTTP/SSE (HttpLoopAgent)
instead of TCP JSON-RPC, and skips bwrap — the container itself is the sandbox.
Auth credentials are provided via the ~/.claude volume mount.
"""

import json
import os
import subprocess
import sys
import threading

sys.path.insert(0, "/app")
from http_loop_agent import HttpLoopAgent


class ClaudeCodeAgent(HttpLoopAgent):
    name = "claude-code"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()

        args = [
            "claude", "-p", message,
            "--output-format", "stream-json",
            "--verbose",
            "--dangerously-skip-permissions",
            "--include-partial-messages",
        ]
        if session_id:
            args += ["--resume", session_id]
        if "systemPrompt" in kwargs:
            args += ["--system-prompt", kwargs["systemPrompt"]]

        proc = subprocess.Popen(
            args,
            stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            cwd=working_dir,
            start_new_session=True,
        )

        def drain_stderr():
            for line in proc.stderr:
                print(f"[claude stderr] {line.decode().rstrip()}", file=sys.stderr, flush=True)
        threading.Thread(target=drain_stderr, daemon=True).start()

        emitted_via_stream = False
        for raw in proc.stdout:
            line = raw.decode().strip()
            if not line:
                continue
            try:
                envelope = json.loads(line)
            except json.JSONDecodeError:
                continue

            t = envelope.get("type")
            if t == "stream_event":
                ev = envelope.get("event", {})
                if ev.get("type") == "content_block_delta":
                    delta = ev.get("delta", {})
                    if delta.get("type") == "text_delta" and delta.get("text"):
                        emitted_via_stream = True
                        yield delta["text"]
            elif t == "assistant":
                # Fallback: emit text blocks from the assistant message event.
                # This covers cases where stream_event deltas aren't produced
                # (e.g. auth errors, very short responses on older CLI versions).
                if not emitted_via_stream:
                    for block in envelope.get("message", {}).get("content", []):
                        if block.get("type") == "text" and block.get("text"):
                            yield block["text"]
            elif t == "result":
                if not envelope.get("is_error"):
                    with self._lock:
                        self._sessions[run_id] = envelope.get("session_id", "")

        proc.wait()

    def get_session_id(self, run_id: str) -> str:
        with self._lock:
            return self._sessions.pop(run_id, "")


if __name__ == "__main__":
    ClaudeCodeAgent().serve()
