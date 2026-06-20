#!/usr/bin/env python3
"""Built-in opencode agent extension for Loop."""

import os
import subprocess
import sys
import threading
from subprocess import DEVNULL, PIPE

sys.path.insert(0, os.path.dirname(__file__))
from loop_agent import LoopAgent
from opencode_stream import parse_opencode_stream


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

        def stdout_lines():
            for raw in proc.stdout:
                yield raw.decode(errors="replace")

        for event in parse_opencode_stream(stdout_lines()):
            if event.get("type") == "session_id":
                with self._lock:
                    self._sessions[run_id] = event.get("sessionId") or ""
                continue
            yield event

        proc.wait()

    def get_session_id(self, run_id: str) -> str:
        with self._lock:
            return self._sessions.pop(run_id, "")


if __name__ == "__main__":
    OpenCodeAgent().serve()
