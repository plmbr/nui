#!/usr/bin/env python3
"""Codex agent for nui Docker container."""

import os
import sys
import threading

sys.path.insert(0, "/app")
from codex_session import PersistentCodexSession
from http_nui_agent import HttpNuiAgent


class CodexAgent(HttpNuiAgent):
    name = "codex"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()
        self._codex = PersistentCodexSession()

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()
        model = kwargs.get("model", "")
        system_prompt = kwargs.get("systemPrompt", "")

        latest_session_id = ""
        for event in self._codex.run_turn(
            message, working_dir, session_id, model, system_prompt, **kwargs,
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
