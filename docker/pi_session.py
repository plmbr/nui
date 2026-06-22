"""Persistent Pi CLI session using RPC mode over stdin/stdout."""

from __future__ import annotations

import json
import os
import subprocess
import sys
import threading
from subprocess import PIPE
from typing import Any, Callable, Generator

from pi_stream import new_pi_stream_parser
from harness_env import subprocess_env


WrapArgsFn = Callable[[list[str], str], list[str]]


class PersistentPiSession:
    """Keep one `pi --mode rpc` process alive and send each prompt over stdin."""

    def __init__(self, wrap_args: WrapArgsFn | None = None) -> None:
        self._wrap_args = wrap_args or (lambda args, _wd: args)
        self._lock = threading.Lock()
        self._proc: subprocess.Popen[str] | None = None
        self._stderr_thread: threading.Thread | None = None
        self._stderr_lines: list[str] = []
        self._working_dir = ""
        self._model = ""
        self._system_prompt = ""
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
        with self._lock:
            yield from self._run_turn_locked(
                message, working_dir, resume_session_id, model, system_prompt, kwargs,
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
        system_prompt: str,
        kwargs: dict[str, Any],
    ) -> Generator[dict[str, Any], None, None]:
        resume = resume_session_id or self._session_id
        self._ensure_process(working_dir, resume, model, system_prompt, kwargs)
        assert self._proc is not None and self._proc.stdin is not None and self._proc.stdout is not None

        produced_output = False
        for event in self._prompt_turn(message):
            if event.get("type") == "session_id":
                self._session_id = event.get("sessionId") or self._session_id
                yield event
                continue
            if event.get("type") not in ("error",):
                produced_output = True
            yield event

        if (
            not produced_output
            and resume
            and any("No session found" in line for line in self._stderr_lines)
        ):
            print("[pi] session not found, retrying without session", file=sys.stderr, flush=True)
            self._stop_unlocked()
            self._session_id = ""
            self._ensure_process(working_dir, "", model, system_prompt, kwargs)
            yield from self._prompt_turn(message)

        self._refresh_session_id()

    def _prompt_turn(self, message: str) -> Generator[dict[str, Any], None, None]:
        assert self._proc is not None and self._proc.stdin is not None and self._proc.stdout is not None
        payload = json.dumps({"type": "prompt", "message": message}) + "\n"
        try:
            self._proc.stdin.write(payload)
            self._proc.stdin.flush()
        except (BrokenPipeError, OSError):
            self._stop_unlocked()
            yield {"type": "error", "error": "pi process ended unexpectedly"}
            return

        parser = new_pi_stream_parser()
        while True:
            raw = self._proc.stdout.readline()
            if not raw:
                self._stop_unlocked()
                yield {"type": "error", "error": "pi process ended unexpectedly"}
                return

            line = raw.strip()
            if not line:
                continue

            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue

            if obj.get("type") == "response":
                if obj.get("command") == "prompt" and not obj.get("success"):
                    err = obj.get("error") or "prompt rejected"
                    yield {"type": "error", "error": str(err)}
                    return
                continue

            if obj.get("type") in ("extension_ui_request", "extension_ui_response"):
                continue

            yield from parser.handle(obj)

            if obj.get("type") == "turn_end":
                return

    def _refresh_session_id(self) -> None:
        assert self._proc is not None and self._proc.stdin is not None and self._proc.stdout is not None
        self._proc.stdin.write(json.dumps({"type": "get_state"}) + "\n")
        self._proc.stdin.flush()
        while True:
            raw = self._proc.stdout.readline()
            if not raw:
                return
            try:
                obj = json.loads(raw.strip())
            except json.JSONDecodeError:
                continue
            if obj.get("type") != "response" or obj.get("command") != "get_state":
                continue
            if obj.get("success") and isinstance(obj.get("data"), dict):
                sid = obj["data"].get("sessionId") or ""
                if sid:
                    self._session_id = sid
            return

    def _ensure_process(
        self,
        working_dir: str,
        resume_session_id: str,
        model: str,
        system_prompt: str,
        kwargs: dict[str, Any],
    ) -> None:
        wd = working_dir or os.getcwd()
        if (
            self._proc is not None
            and self._proc.poll() is None
            and self._working_dir == wd
            and self._model == model
            and self._system_prompt == system_prompt
        ):
            return

        self._stop_unlocked()
        if resume_session_id and not self._session_id:
            self._session_id = resume_session_id

        pi_args = ["pi", "--mode", "rpc"]
        if model:
            pi_args += ["--model", model]
        if system_prompt:
            pi_args += ["--system-prompt", system_prompt]
        resume = self._session_id or resume_session_id
        if resume:
            pi_args += ["--session", resume]

        args = self._wrap_args(pi_args, wd)
        cwd = None if os.environ.get("LOOP_BWRAP_PATH") else wd

        self._stderr_lines = []
        proc = subprocess.Popen(
            args,
            stdin=PIPE,
            stdout=PIPE,
            stderr=PIPE,
            cwd=cwd,
            env=subprocess_env(kwargs),
            text=True,
            bufsize=1,
            start_new_session=True,
        )
        self._proc = proc
        self._working_dir = wd
        self._model = model
        self._system_prompt = system_prompt

        def drain_stderr() -> None:
            assert proc.stderr is not None
            for line in proc.stderr:
                text = line.rstrip()
                self._stderr_lines.append(text)
                print(f"[pi stderr] {text}", file=sys.stderr, flush=True)

        self._stderr_thread = threading.Thread(target=drain_stderr, daemon=True)
        self._stderr_thread.start()

    def _stop_unlocked(self) -> None:
        proc = self._proc
        self._proc = None
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
