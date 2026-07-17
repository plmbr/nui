#!/usr/bin/env python3
"""Built-in codex extension for nui."""

import os
import sys
import threading

sys.path.insert(0, os.path.dirname(__file__))
from codex_session import PersistentCodexSession
from nui_agent import NuiAgent


def _wrap_with_bwrap(codex_args: list[str], working_dir: str) -> list[str]:
    """Prepend bwrap wrapper args if NUI_BWRAP_PATH is set, otherwise return as-is."""
    bwrap_path = os.environ.get("NUI_BWRAP_PATH", "")
    if not bwrap_path:
        return codex_args

    home = os.path.expanduser("~")
    codex_dir = os.path.join(home, ".codex")
    os.makedirs(codex_dir, exist_ok=True)

    bwrap_args = [
        bwrap_path,
        "--unshare-user", "--unshare-ipc", "--unshare-pid", "--unshare-uts",
        "--ro-bind", "/", "/",
        "--proc", "/proc",
        "--dev", "/dev",
        "--tmpfs", "/tmp",
        "--die-with-parent",
        "--bind", codex_dir, codex_dir,
    ]
    if working_dir:
        bwrap_args += ["--bind", working_dir, working_dir, "--chdir", working_dir]
    bwrap_args += ["--"]
    return bwrap_args + codex_args


class CodexAgent(NuiAgent):
    name = "codex"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()
        self._codex = PersistentCodexSession(wrap_args=_wrap_with_bwrap)

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()
        model = kwargs.get("model", "")
        system_prompt = kwargs.get("systemPrompt", "")

        latest_session_id = ""
        for event in self._codex.run_turn(
            message, working_dir, session_id, model, system_prompt,
        ):
            if event.get("type") == "session_id":
                latest_session_id = event.get("sessionId") or ""
                continue
            yield event

        if latest_session_id:
            with self._lock:
                self._sessions[run_id] = latest_session_id
        elif self._codex.session_id:
            with self._lock:
                self._sessions[run_id] = self._codex.session_id

    def get_session_id(self, run_id: str) -> str:
        with self._lock:
            return self._sessions.pop(run_id, "")

    def on_shutdown(self) -> None:
        self._codex.stop()


if __name__ == "__main__":
    CodexAgent().serve()
