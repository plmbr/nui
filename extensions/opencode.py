#!/usr/bin/env python3
"""Built-in opencode agent extension for Loop.

Wraps the `opencode` CLI exactly the way pi.py wraps the `pi` CLI —
subprocess with --format json, streaming text parts, session resume via
--session <id>.
"""

import json
import os
import subprocess
import sys
import threading
from subprocess import PIPE, DEVNULL

sys.path.insert(0, os.path.dirname(__file__))
from loop_agent import LoopAgent


class OpenCodeAgent(LoopAgent):
    name = "opencode"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()
        model = kwargs.get("model", "")

        args = ["opencode", "run", "--format", "json"]
        if session_id:
            args += ["--session", session_id]
        if working_dir:
            args += ["--dir", working_dir]
        if model:
            args += ["-m", model]
        args += [message]

        proc = subprocess.Popen(
            args,
            stdin=DEVNULL,
            stdout=PIPE,
            stderr=PIPE,
            start_new_session=True,
        )

        def drain_stderr():
            for line in proc.stderr:
                print(f"[opencode stderr] {line.decode().rstrip()}", file=sys.stderr, flush=True)
        threading.Thread(target=drain_stderr, daemon=True).start()

        for raw in proc.stdout:
            line = raw.decode(errors="replace").strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue

            t = obj.get("type", "")
            sid = obj.get("sessionID", "")

            # Capture session ID from the first event that carries it.
            if sid:
                with self._lock:
                    if run_id not in self._sessions:
                        self._sessions[run_id] = sid

            if t == "text":
                part = obj.get("part", {})
                if part.get("type") == "text":
                    text = part.get("text", "")
                    if text:
                        yield text

        proc.wait()

    def get_session_id(self, run_id: str) -> str:
        with self._lock:
            return self._sessions.pop(run_id, "")


if __name__ == "__main__":
    OpenCodeAgent().serve()
