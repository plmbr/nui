"""Persistent Claude CLI session using the stream-json stdin/stdout protocol."""

from __future__ import annotations

import json
import os
import subprocess
import sys
import threading
from subprocess import PIPE
from typing import Any, Callable, Generator

from claude_stream import new_claude_stream_parser
from harness_env import subprocess_env


WrapArgsFn = Callable[[list[str], str], list[str]]


def run_ephemeral_claude_turn(
    message: str,
    working_dir: str,
    model: str = "",
    system_prompt: str = "",
    wrap_args: WrapArgsFn | None = None,
    **kwargs: Any,
) -> Generator[dict[str, Any], None, None]:
    """One-shot Claude turn that never resumes an existing session or mutates a persistent session."""
    wrap = wrap_args or (lambda args, _wd: args)
    wd = working_dir or os.getcwd()

    claude_args = [
        "claude",
        "-p",
        "--input-format",
        "stream-json",
        "--output-format",
        "stream-json",
        "--verbose",
        "--dangerously-skip-permissions",
        "--include-partial-messages",
    ]
    if model:
        claude_args += ["--model", model]
    if system_prompt:
        claude_args += ["--system-prompt", system_prompt]
    if kwargs.get("userScopeHarness"):
        claude_args += ["--setting-sources", "user,project,local"]
        mcp_path = os.path.join("/home/nui/.nui/session-config", ".claude.json")
        if os.path.isfile(mcp_path):
            claude_args += ["--mcp-config", mcp_path]

    args = wrap(claude_args, wd)
    cwd = None if os.environ.get("NUI_BWRAP_PATH") else wd

    proc = subprocess.Popen(
        args,
        stdin=PIPE,
        stdout=PIPE,
        stderr=PIPE,
        cwd=cwd,
        env=subprocess_env(kwargs),
        start_new_session=True,
    )
    assert proc.stdin is not None and proc.stdout is not None

    def drain_stderr() -> None:
        assert proc.stderr is not None
        for line in proc.stderr:
            print(f"[claude stderr] {line.decode(errors='replace').rstrip()}", file=sys.stderr, flush=True)

    threading.Thread(target=drain_stderr, daemon=True).start()

    user_msg: dict[str, Any] = {
        "type": "user",
        "message": {
            "role": "user",
            "content": [{"type": "text", "text": message}],
        },
    }
    payload = (json.dumps(user_msg) + "\n").encode()
    try:
        proc.stdin.write(payload)
        proc.stdin.flush()
        proc.stdin.close()
    except (BrokenPipeError, OSError):
        proc.kill()
        yield {"type": "error", "error": "claude process ended unexpectedly"}
        return

    parser = new_claude_stream_parser()
    while True:
        raw = proc.stdout.readline()
        if not raw:
            proc.wait(timeout=5)
            yield {"type": "error", "error": "claude process ended unexpectedly"}
            return

        line = raw.decode(errors="replace").strip()
        if not line:
            continue

        try:
            envelope = json.loads(line)
        except json.JSONDecodeError:
            continue

        yield from parser.handle_envelope(envelope)

        msg_type = envelope.get("type")
        if msg_type == "result":
            if envelope.get("is_error"):
                err = envelope.get("error")
                if isinstance(err, dict):
                    err = err.get("message") or json.dumps(err)
                yield {"type": "error", "error": str(err or "unknown error")}
            proc.wait(timeout=5)
            return

        if msg_type == "error":
            err = envelope.get("error")
            if isinstance(err, dict):
                err = err.get("message") or json.dumps(err)
            yield {"type": "error", "error": str(err or "unknown error")}
            proc.wait(timeout=5)
            return


class PersistentClaudeSession:
    """Keep one Claude CLI process alive and send each prompt over stdin."""

    def __init__(self, wrap_args: WrapArgsFn | None = None) -> None:
        self._wrap_args = wrap_args or (lambda args, _wd: args)
        self._lock = threading.Lock()
        self._proc: subprocess.Popen[bytes] | None = None
        self._stderr_thread: threading.Thread | None = None
        self._working_dir = ""
        self._model = ""
        self._system_prompt = ""
        self._claude_session_id = ""

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

    def _run_turn_locked(
        self,
        message: str,
        working_dir: str,
        resume_session_id: str,
        model: str,
        system_prompt: str,
        kwargs: dict[str, Any],
    ) -> Generator[dict[str, Any], None, None]:
        self._ensure_process(working_dir, resume_session_id, model, system_prompt, kwargs)
        assert self._proc is not None and self._proc.stdin is not None and self._proc.stdout is not None

        user_msg: dict[str, Any] = {
            "type": "user",
            "message": {
                "role": "user",
                "content": [{"type": "text", "text": message}],
            },
        }
        session_id = self._claude_session_id or resume_session_id
        if session_id:
            user_msg["session_id"] = session_id

        payload = (json.dumps(user_msg) + "\n").encode()
        try:
            self._proc.stdin.write(payload)
            self._proc.stdin.flush()
        except (BrokenPipeError, OSError):
            self._stop_unlocked()
            self._ensure_process(working_dir, resume_session_id, model, system_prompt, kwargs)
            assert self._proc is not None and self._proc.stdin is not None and self._proc.stdout is not None
            self._proc.stdin.write(payload)
            self._proc.stdin.flush()

        parser = new_claude_stream_parser()
        while True:
            raw = self._proc.stdout.readline()
            if not raw:
                self._stop_unlocked()
                yield {"type": "error", "error": "claude process ended unexpectedly"}
                return

            line = raw.decode(errors="replace").strip()
            if not line:
                continue

            try:
                envelope = json.loads(line)
            except json.JSONDecodeError:
                continue

            yield from parser.handle_envelope(envelope)

            msg_type = envelope.get("type")
            if msg_type == "result":
                if not envelope.get("is_error"):
                    sid = envelope.get("session_id") or ""
                    if sid:
                        self._claude_session_id = sid
                return

            if msg_type == "error":
                err = envelope.get("error")
                if isinstance(err, dict):
                    err = err.get("message") or json.dumps(err)
                yield {"type": "error", "error": str(err or "unknown error")}
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
        if resume_session_id and not self._claude_session_id:
            self._claude_session_id = resume_session_id

        claude_args = [
            "claude",
            "-p",
            "--input-format",
            "stream-json",
            "--output-format",
            "stream-json",
            "--verbose",
            "--dangerously-skip-permissions",
            "--include-partial-messages",
        ]
        if model:
            claude_args += ["--model", model]
        if system_prompt:
            claude_args += ["--system-prompt", system_prompt]
        resume = self._claude_session_id or resume_session_id
        if resume:
            claude_args += ["--resume", resume]
        if kwargs.get("userScopeHarness"):
            claude_args += ["--setting-sources", "user,project,local"]
            mcp_path = os.path.join("/home/nui/.nui/session-config", ".claude.json")
            if os.path.isfile(mcp_path):
                claude_args += ["--mcp-config", mcp_path]

        args = self._wrap_args(claude_args, wd)
        cwd = None if os.environ.get("NUI_BWRAP_PATH") else wd

        proc = subprocess.Popen(
            args,
            stdin=PIPE,
            stdout=PIPE,
            stderr=PIPE,
            cwd=cwd,
            env=subprocess_env(kwargs),
            start_new_session=True,
        )
        self._proc = proc
        self._working_dir = wd
        self._model = model
        self._system_prompt = system_prompt

        def drain_stderr() -> None:
            assert proc.stderr is not None
            for line in proc.stderr:
                print(f"[claude stderr] {line.decode(errors='replace').rstrip()}", file=sys.stderr, flush=True)

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
