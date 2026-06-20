#!/usr/bin/env python3
"""Codex agent for Loop Docker container.

Speaks HTTP/SSE (HttpLoopAgent). Auth is provided via ANTHROPIC_API_KEY /
ANTHROPIC_BASE_URL forwarded from the host. --ignore-user-config skips any
container-side config that might have MCP servers or conflicting settings.
"""

import os
import subprocess
import sys
import threading
from subprocess import DEVNULL, PIPE

sys.path.insert(0, "/app")
from codex_stream import parse_codex_stream
from http_loop_agent import HttpLoopAgent


class CodexAgent(HttpLoopAgent):
    name = "codex"
    version = "0.1.0"

    def __init__(self):
        self._sessions: dict[str, str] = {}
        self._lock = threading.Lock()

    def run(self, message: str, run_id: str, **kwargs):
        session_id = kwargs.get("sessionId", "")
        working_dir = kwargs.get("workingDir", "") or os.getcwd()

        # Build flags first, then positional args — codex subcommands require
        # options before [SESSION_ID] [PROMPT] positional args.
        flags = [
            "--json",
            "--dangerously-bypass-approvals-and-sandbox",
            "--skip-git-repo-check",
            "--ignore-user-config",
        ]

        base_url = os.environ.get("OPENAI_BASE_URL", "")
        if base_url:
            # Use a custom provider with WebSocket disabled — most OpenAI-compatible
            # gateways don't support WebSocket upgrades.
            # supports_websockets=false makes codex skip straight to HTTPS Responses API.
            flags += [
                "-c", 'model_provider="loop_gateway"',
                "-c", f'model_providers.loop_gateway={{name="loop_gateway",env_key="OPENAI_API_KEY",base_url="{base_url}",supports_websockets=false}}',
            ]

        model = kwargs.get("model", "")
        if model:
            flags += ["-m", model]

        # `codex exec resume` doesn't accept --cd/-C; set the subprocess cwd
        # instead. The host workingDir is bind-mounted at the same path.
        if session_id:
            args = ["codex", "exec", "resume"] + flags + [session_id, message]
        else:
            args = ["codex", "exec"] + flags + [message]

        proc = subprocess.Popen(
            args,
            stdin=DEVNULL,
            stdout=PIPE,
            stderr=PIPE,
            start_new_session=True,
            cwd=working_dir if working_dir and os.path.isdir(working_dir) else None,
        )

        def drain_stderr():
            for line in proc.stderr:
                print(f"[codex stderr] {line.decode().rstrip()}", file=sys.stderr, flush=True)
        threading.Thread(target=drain_stderr, daemon=True).start()

        def stdout_lines():
            for raw in proc.stdout:
                yield raw.decode(errors="replace")

        for event in parse_codex_stream(stdout_lines()):
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
    CodexAgent().serve()
