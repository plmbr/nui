"""Persistent Codex session manager.

Codex does not yet expose a multi-turn stdin protocol like Claude stream-json.
This keeps thread IDs across turns and reuses exec arguments/config until the
Loop session ends.
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import threading
from subprocess import DEVNULL, PIPE
from typing import Any, Callable, Generator

from codex_stream import new_codex_stream_parser


WrapArgsFn = Callable[[list[str], str], list[str]]

_CODEX_BINARY_CANDIDATES = (
    "codex",
    "/Applications/Codex.app/Contents/Resources/codex",
)


def find_codex_binary() -> str:
    """Locate the codex CLI, matching the Go agent's search order."""
    override = os.environ.get("LOOP_CODEX_PATH", "").strip()
    if override:
        return override

    found = shutil.which("codex")
    if found:
        return found

    for path in _CODEX_BINARY_CANDIDATES[1:]:
        if os.path.isfile(path) and os.access(path, os.X_OK):
            return path

    return "codex"


class PersistentCodexSession:
    def __init__(self, wrap_args: WrapArgsFn | None = None, binary: str | None = None) -> None:
        self._wrap_args = wrap_args or (lambda args, _wd: args)
        self._binary = binary or find_codex_binary()
        self._lock = threading.Lock()
        self._proc: subprocess.Popen[str] | None = None
        self._working_dir = ""
        self._model = ""
        self._thread_id = ""
        self._extra_flags: list[str] = []

    def run_turn(
        self,
        message: str,
        working_dir: str,
        resume_session_id: str = "",
        model: str = "",
        system_prompt: str = "",
    ) -> Generator[dict[str, Any], None, None]:
        _ = system_prompt
        with self._lock:
            yield from self._run_turn_locked(
                message, working_dir, resume_session_id, model,
            )

    def stop(self) -> None:
        with self._lock:
            self._stop_unlocked()

    @property
    def session_id(self) -> str:
        return self._thread_id

    def _run_turn_locked(
        self,
        message: str,
        working_dir: str,
        resume_session_id: str,
        model: str,
    ) -> Generator[dict[str, Any], None, None]:
        wd = working_dir or os.getcwd()
        if self._working_dir != wd or self._model != model:
            self._thread_id = ""
        self._working_dir = wd
        self._model = model

        thread_id = self._thread_id or resume_session_id
        flags = self._build_flags(model)
        if not os.path.isfile(self._binary) and not shutil.which(self._binary):
            yield {
                "type": "error",
                "error": (
                    "codex CLI not found. Install Codex or set LOOP_CODEX_PATH to the binary "
                    f"(checked PATH and {_CODEX_BINARY_CANDIDATES[1]})"
                ),
            }
            return
        if thread_id:
            args = [self._binary, "exec", "resume"] + flags + [thread_id, message]
        else:
            args = [self._binary, "exec"] + flags + [message]

        args = self._wrap_args(args, wd)
        cwd = None if os.environ.get("LOOP_BWRAP_PATH") else wd

        proc = subprocess.Popen(
            args,
            stdin=DEVNULL,
            stdout=PIPE,
            stderr=PIPE,
            cwd=cwd,
            text=True,
            bufsize=1,
            start_new_session=True,
        )
        self._proc = proc

        def drain_stderr() -> None:
            assert proc.stderr is not None
            for line in proc.stderr:
                print(f"[codex stderr] {line.rstrip()}", file=sys.stderr, flush=True)

        threading.Thread(target=drain_stderr, daemon=True).start()

        parser = new_codex_stream_parser()
        assert proc.stdout is not None
        for raw in proc.stdout:
            line = raw.strip()
            if not line:
                continue
            try:
                envelope = json.loads(line)
            except json.JSONDecodeError:
                continue

            if envelope.get("type") == "thread.started":
                tid = envelope.get("thread_id") or ""
                if tid:
                    self._thread_id = tid

            yield from parser.handle(envelope)

            if envelope.get("type") in ("turn.completed", "turn.failed", "error"):
                break

        proc.wait()
        self._proc = None

    def _build_flags(self, model: str) -> list[str]:
        flags = [
            "--json",
            "--dangerously-bypass-approvals-and-sandbox",
            "--skip-git-repo-check",
            "--ignore-user-config",
        ]
        base_url = os.environ.get("OPENAI_BASE_URL", "")
        if base_url:
            flags += [
                "-c", 'model_provider="loop_gateway"',
                "-c",
                f'model_providers.loop_gateway={{name="loop_gateway",env_key="OPENAI_API_KEY",base_url="{base_url}",supports_websockets=false}}',
            ]
        if model:
            flags += ["-m", model]
        return flags

    def _stop_unlocked(self) -> None:
        proc = self._proc
        self._proc = None
        if proc is None:
            return
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=2)
