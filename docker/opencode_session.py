"""Persistent OpenCode session using a long-lived `opencode serve` process."""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import threading
from subprocess import DEVNULL, PIPE
from typing import Any, Generator

from opencode_stream import new_opencode_stream_parser
from harness_env import subprocess_env


class PersistentOpenCodeSession:
    """Keep one `opencode serve` process alive and route prompts through it."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._server: subprocess.Popen[str] | None = None
        self._stderr_thread: threading.Thread | None = None
        self._base_url = ""
        self._working_dir = ""
        self._model = ""
        self._session_id = ""

    def run_turn(
        self,
        message: str,
        working_dir: str,
        resume_session_id: str = "",
        model: str = "",
        system_prompt: str = "",
        **kwargs: Any,
    ) -> Generator[dict[str, Any], None, None]:
        _ = system_prompt
        with self._lock:
            yield from self._run_turn_locked(
                message, working_dir, resume_session_id, model, kwargs,
            )

    def stop(self) -> None:
        with self._lock:
            self._stop_unlocked()

    @property
    def session_id(self) -> str:
        return self._session_id

    def _run_turn_locked(
        self,
        message: str,
        working_dir: str,
        resume_session_id: str,
        model: str,
        kwargs: dict[str, Any],
    ) -> Generator[dict[str, Any], None, None]:
        self._ensure_server(working_dir, model, kwargs)
        session_id = self._session_id or resume_session_id

        args = [
            "opencode", "run", "--format", "json",
            "--attach", self._base_url,
        ]
        if session_id:
            args += ["--session", session_id]
        wd = working_dir or os.getcwd()
        if wd:
            args += ["--dir", wd]
        if model:
            args += ["-m", model]
        args += [message]

        proc = subprocess.Popen(
            args,
            stdin=DEVNULL,
            stdout=PIPE,
            stderr=PIPE,
            env=subprocess_env(kwargs),
            text=True,
            bufsize=1,
            start_new_session=True,
        )

        def drain_stderr() -> None:
            assert proc.stderr is not None
            for line in proc.stderr:
                print(f"[opencode stderr] {line.rstrip()}", file=sys.stderr, flush=True)

        threading.Thread(target=drain_stderr, daemon=True).start()

        parser = new_opencode_stream_parser()
        assert proc.stdout is not None
        for raw in proc.stdout:
            line = raw.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue
            sid = obj.get("sessionID") or obj.get("sessionId") or ""
            if sid:
                self._session_id = sid
            yield from parser.handle(obj)

        proc.wait()

    def _ensure_server(self, working_dir: str, model: str, kwargs: dict[str, Any]) -> None:
        wd = working_dir or os.getcwd()
        if (
            self._server is not None
            and self._server.poll() is None
            and self._base_url
            and self._working_dir == wd
            and self._model == model
        ):
            return

        self._stop_unlocked()
        proc = subprocess.Popen(
            ["opencode", "serve", "--port", "0", "--hostname", "127.0.0.1"],
            stdin=DEVNULL,
            stdout=PIPE,
            stderr=PIPE,
            cwd=wd,
            env=subprocess_env(kwargs),
            text=True,
            bufsize=1,
            start_new_session=True,
        )
        self._server = proc
        self._working_dir = wd
        self._model = model
        self._base_url = self._wait_for_server_url(proc)
        if not self._base_url:
            self._stop_unlocked()
            raise RuntimeError("opencode serve failed to start")

    def _wait_for_server_url(self, proc: subprocess.Popen[str]) -> str:
        pattern = re.compile(r"listening on (https?://[^\s]+)", re.IGNORECASE)
        assert proc.stderr is not None
        deadline = threading.Event()

        def read_stderr() -> None:
            for line in proc.stderr:
                print(f"[opencode serve] {line.rstrip()}", file=sys.stderr, flush=True)
                match = pattern.search(line)
                if match:
                    self._base_url = match.group(1).rstrip("/")
                    deadline.set()
                    return

        self._stderr_thread = threading.Thread(target=read_stderr, daemon=True)
        self._stderr_thread.start()
        if not deadline.wait(timeout=15):
            return ""
        return self._base_url

    def _stop_unlocked(self) -> None:
        proc = self._server
        self._server = None
        self._base_url = ""
        self._working_dir = ""
        if proc is None:
            return
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=2)
