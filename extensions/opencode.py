#!/usr/bin/env python3
"""Built-in opencode agent extension for Loop."""

import os
import sys
import threading

sys.path.insert(0, os.path.dirname(__file__))
from loop_agent import LoopAgent
from opencode_session import PersistentOpenCodeSession


class OpenCodeAgent(LoopAgent):
    name = "opencode"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()
        self._opencode = PersistentOpenCodeSession()

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()
        model = kwargs.get("model", "")
        system_prompt = kwargs.get("systemPrompt", "")

        latest_session_id = ""
        for event in self._opencode.run_turn(
            message, working_dir, session_id, model, system_prompt,
        ):
            if event.get("type") == "session_id":
                latest_session_id = event.get("sessionId") or ""
                continue
            yield event

        if latest_session_id:
            with self._lock:
                self._sessions[run_id] = latest_session_id
        elif self._opencode.session_id:
            with self._lock:
                self._sessions[run_id] = self._opencode.session_id

    def get_session_id(self, run_id: str) -> str:
        with self._lock:
            return self._sessions.pop(run_id, "")

    def on_shutdown(self) -> None:
        self._opencode.stop()


if __name__ == "__main__":
    OpenCodeAgent().serve()
