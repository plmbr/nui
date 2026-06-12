#!/usr/bin/env python3
"""Built-in Pi agent extension for Loop.

Wraps the `pi` CLI (AI coding assistant) exactly the way claude_code.py
wraps the `claude` CLI — subprocess with --mode json, streaming text deltas,
session resume via --session <uuid>.
"""

import json
import os
import subprocess
import sys
import threading
from subprocess import PIPE

sys.path.insert(0, os.path.dirname(__file__))
from loop_agent import LoopAgent


class PiAgent(LoopAgent):
    name = "pi"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()

        args = ["pi", "-p", message, "--mode", "json"]
        if session_id:
            args += ["--session", session_id]

        proc = subprocess.Popen(args, stdout=PIPE, stderr=PIPE, cwd=working_dir)

        for raw in proc.stdout:
            line = raw.decode(errors="replace").strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue

            t = obj.get("type", "")

            if t == "session":
                sid = obj.get("id", "")
                if sid:
                    with self._lock:
                        self._sessions[run_id] = sid

            elif t == "message_update":
                ev = obj.get("assistantMessageEvent", {})
                if ev.get("type") == "text_delta":
                    delta = ev.get("delta", "")
                    if delta:
                        yield delta

        proc.wait()

    def get_session_id(self, run_id: str) -> str:
        with self._lock:
            return self._sessions.pop(run_id, "")


if __name__ == "__main__":
    PiAgent().serve()
