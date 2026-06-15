#!/usr/bin/env python3
"""Pi agent for Loop Docker container.

Same logic as extensions/pi.py but speaks HTTP/SSE (HttpLoopAgent)
instead of TCP JSON-RPC. Auth credentials must be supplied via ANTHROPIC_API_KEY
(forwarded by Loop from the host environment).
"""

import json
import os
import subprocess
import sys
import threading
from subprocess import DEVNULL, PIPE

sys.path.insert(0, "/app")
from http_loop_agent import HttpLoopAgent


class PiAgent(HttpLoopAgent):
    name = "pi"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()

        yield from self._run_pi(message, run_id, session_id, working_dir)

    def _run_pi(self, message: str, run_id: str, session_id: str, working_dir: str):
        args = ["pi", "-p", message, "--mode", "json"]
        if session_id:
            args += ["--session", session_id]

        proc = subprocess.Popen(
            args, stdin=DEVNULL, stdout=PIPE, stderr=PIPE,
            cwd=working_dir, start_new_session=True,
        )

        stderr_lines = []

        def drain_stderr():
            for line in proc.stderr:
                text = line.decode().rstrip()
                stderr_lines.append(text)
                print(f"[pi stderr] {text}", file=sys.stderr, flush=True)
        threading.Thread(target=drain_stderr, daemon=True).start()

        produced_output = False
        for raw in proc.stdout:
            line = raw.decode(errors="replace").strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue

            produced_output = True
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

        # If pi exited with no output due to a missing session, retry fresh.
        if not produced_output and session_id and any("No session found" in l for l in stderr_lines):
            print("[pi] session not found in container, retrying without session", file=sys.stderr, flush=True)
            yield from self._run_pi(message, run_id, "", working_dir)

    def get_session_id(self, run_id: str) -> str:
        with self._lock:
            return self._sessions.pop(run_id, "")


def _write_models_json():
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
