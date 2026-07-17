#!/usr/bin/env python3
"""Built-in claude-code extension for nui."""

import os
import sys
import threading

sys.path.insert(0, os.path.dirname(__file__))
from claude_session import PersistentClaudeSession
from nui_agent import NuiAgent


def _wrap_with_bwrap(claude_args: list[str], working_dir: str) -> list[str]:
    """Prepend bwrap wrapper args if NUI_BWRAP_PATH is set, otherwise return as-is."""
    import os

    bwrap_path = os.environ.get("NUI_BWRAP_PATH", "")
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


class ClaudeCodeAgent(NuiAgent):
    name = "claude-code"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()
        self._claude = PersistentClaudeSession(wrap_args=_wrap_with_bwrap)

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()
        model = kwargs.get("model", "")
        system_prompt = kwargs.get("systemPrompt", "")

        latest_session_id = ""
        for event in self._claude.run_turn(
            message, working_dir, session_id, model, system_prompt,
        ):
            if event.get("type") == "session_id":
                latest_session_id = event.get("sessionId") or ""
                continue
            yield event

        if latest_session_id:
            with self._lock:
                self._sessions[run_id] = latest_session_id

    def get_session_id(self, run_id: str) -> str:
        with self._lock:
            return self._sessions.pop(run_id, "")

    def on_shutdown(self) -> None:
        self._claude.stop()


if __name__ == "__main__":
    ClaudeCodeAgent().serve()
