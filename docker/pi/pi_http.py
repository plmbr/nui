#!/usr/bin/env python3
"""Pi agent for nui Docker container."""

import os
import sys
import threading

sys.path.insert(0, "/app")
from http_nui_agent import HttpNuiAgent
from pi_session import PersistentPiSession


class PiAgent(HttpNuiAgent):
    name = "pi"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()
        self._pi = PersistentPiSession()

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()
        model = kwargs.get("model", "")
        system_prompt = kwargs.get("systemPrompt", "")

        latest_session_id = ""
        for event in self._pi.run_turn(
            message, working_dir, session_id, model, system_prompt, **kwargs,
        ):
            if event.get("type") == "session_id":
                latest_session_id = event.get("sessionId") or ""
                continue
            yield event

        if latest_session_id:
            with self._lock:
                self._sessions[run_id] = latest_session_id
        elif self._pi.session_id:
            with self._lock:
                self._sessions[run_id] = self._pi.session_id

    def get_session_id(self, run_id: str) -> str:
        with self._lock:
            return self._sessions.pop(run_id, "")

    def on_shutdown(self) -> None:
        self._pi.stop()


def _write_models_json():
    import json

    base_url = os.environ.get("ANTHROPIC_BASE_URL", "")
    if not base_url:
        return
    api_key = os.environ.get("ANTHROPIC_API_KEY") or "sk-dummy"
    agent_dir = os.path.expanduser("~/.pi/agent")
    os.makedirs(agent_dir, exist_ok=True)
    models_path = os.path.join(agent_dir, "models.json")
    config = {"providers": {"anthropic": {"baseUrl": base_url, "apiKey": api_key}}}
    with open(models_path, "w") as f:
        json.dump(config, f)
    print(f"[pi] set anthropic baseUrl -> {base_url}", file=sys.stderr, flush=True)


if __name__ == "__main__":
    _write_models_json()
    PiAgent().serve()
