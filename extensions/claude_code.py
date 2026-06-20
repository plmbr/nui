#!/usr/bin/env python3
"""Built-in claude-code extension for Loop."""

import json
import os
import subprocess
import sys
import threading

sys.path.insert(0, os.path.dirname(__file__))
from claude_stream import parse_claude_stream
from loop_agent import LoopAgent


def _wrap_with_bwrap(claude_args: list[str], working_dir: str) -> list[str]:
    """Prepend bwrap wrapper args if LOOP_BWRAP_PATH is set, otherwise return as-is."""
    bwrap_path = os.environ.get("LOOP_BWRAP_PATH", "")
    if not bwrap_path:
        return claude_args

    home = os.path.expanduser("~")
    claude_dir = os.path.join(home, ".claude")
    os.makedirs(claude_dir, exist_ok=True)

    bwrap_args = [
        bwrap_path,
        "--unshare-user", "--unshare-ipc", "--unshare-pid", "--unshare-uts",
        "--ro-bind", "/", "/",
        "--proc", "/proc",
        "--dev", "/dev",
        "--tmpfs", "/tmp",
        "--die-with-parent",
        "--bind", claude_dir, claude_dir,
    ]
    if working_dir:
        bwrap_args += ["--bind", working_dir, working_dir, "--chdir", working_dir]
    bwrap_args += ["--"]
    return bwrap_args + claude_args


class ClaudeCodeAgent(LoopAgent):
    name = "claude-code"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()

        claude_args = [
            "claude", "-p", message,
            "--output-format", "stream-json",
            "--verbose",
            "--dangerously-skip-permissions",
            "--include-partial-messages",
        ]
        if session_id:
            claude_args += ["--resume", session_id]

        args = _wrap_with_bwrap(claude_args, working_dir)
        cwd = None if os.environ.get("LOOP_BWRAP_PATH") else working_dir

        proc = subprocess.Popen(
            args,
            stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            cwd=cwd,
            start_new_session=True,
        )

        def drain_stderr():
            for line in proc.stderr:
                print(f"[claude stderr] {line.decode().rstrip()}", file=sys.stderr, flush=True)
        threading.Thread(target=drain_stderr, daemon=True).start()

        def stdout_lines():
            for raw in proc.stdout:
                yield raw.decode()

        for event in parse_claude_stream(stdout_lines()):
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
    ClaudeCodeAgent().serve()
