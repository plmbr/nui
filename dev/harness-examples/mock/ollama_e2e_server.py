#!/usr/bin/env python3
"""
Minimal Ollama API mock for nui Playwright E2E tests.

Simulates weak local models that print tool JSON as assistant text instead of
using native tool_calls. nui should recover tool calls and never show raw JSON.

Run:
    python3 ollama_e2e_server.py --port 11435
"""

from __future__ import annotations

import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any


MODEL = "e2e-mock:latest"


def last_user_message(messages: list[dict[str, Any]]) -> str:
    for msg in reversed(messages):
        if msg.get("role") == "user":
            content = msg.get("content", "")
            if isinstance(content, str):
                return content.strip()
    return ""


def has_tool_results_since_last_user(messages: list[dict[str, Any]]) -> bool:
    for msg in reversed(messages):
        if msg.get("role") == "user":
            break
        if msg.get("role") == "tool":
            return True
    return False


def stream_chunks(handler: BaseHTTPRequestHandler, pieces: list[str]) -> None:
    handler.send_response(200)
    handler.send_header("Content-Type", "application/x-ndjson")
    handler.end_headers()
    for piece in pieces:
        payload = {
            "model": MODEL,
            "created_at": "2026-01-01T00:00:00Z",
            "message": {"role": "assistant", "content": piece},
            "done": False,
        }
        handler.wfile.write((json.dumps(payload) + "\n").encode("utf-8"))
        handler.wfile.flush()
    done = {
        "model": MODEL,
        "created_at": "2026-01-01T00:00:00Z",
        "message": {"role": "assistant", "content": ""},
        "done": True,
        "done_reason": "stop",
    }
    handler.wfile.write((json.dumps(done) + "\n").encode("utf-8"))
    handler.wfile.flush()


def stream_native_tool_call(handler: BaseHTTPRequestHandler, tool_name: str, args: dict[str, Any]) -> None:
    handler.send_response(200)
    handler.send_header("Content-Type", "application/x-ndjson")
    handler.end_headers()
    payload = {
        "model": MODEL,
        "created_at": "2026-01-01T00:00:00Z",
        "message": {
            "role": "assistant",
            "content": "",
            "tool_calls": [{"function": {"name": tool_name, "arguments": args}}],
        },
        "done": False,
    }
    handler.wfile.write((json.dumps(payload) + "\n").encode("utf-8"))
    handler.wfile.flush()
    done = {
        "model": MODEL,
        "created_at": "2026-01-01T00:00:00Z",
        "message": {"role": "assistant", "content": ""},
        "done": True,
        "done_reason": "stop",
    }
    handler.wfile.write((json.dumps(done) + "\n").encode("utf-8"))
    handler.wfile.flush()


def stream_native_ask_user(handler: BaseHTTPRequestHandler, args: dict[str, Any]) -> None:
    stream_native_tool_call(handler, "ask_user", args)


def plain_text_reply(user_text: str) -> list[str]:
    lower = user_text.lower().strip()
    if lower in {"hi", "hello", "hmm"}:
        return ["Hello! How can I help you today?"]
    if "what is 2+2" in lower or lower == "2+2":
        return ["2+2 equals 4."]
    if "2+2" in lower and ("prefer" in lower or "choose" in lower):
        return ["Thanks for confirming. 2+2 equals 4."]
    if "what can you do" in lower or "what can u do" in lower or "what can u fo" in lower:
        return [
            "I can answer questions, help with code, and create charts when you ask for them."
        ]
    if "capital of france" in lower:
        return ["France's capital is Paris."]
    return [f"Mock echo: {user_text}"]


def bad_ask_user_json() -> str:
    return json.dumps(
        {
            "name": "ask_user",
            "parameters": {
                "title": "Math calculation",
                "questions": [
                    {
                        "question": "What is 2+2?",
                        "options": [
                            {"label": "4", "description": "Correct"},
                            {"label": "5", "description": "Incorrect"},
                        ],
                    }
                ],
            },
        },
        separators=(",", ":"),
    )


def bad_ask_user_capabilities_json() -> str:
    return json.dumps(
        {
            "name": "ask_user",
            "parameters": {
                "message": "I can perform various tasks. Which one would you like me to do?",
                "questions": [
                    {
                        "header": "Choose an action",
                        "options": [
                            {
                                "label": "Answer a question",
                                "description": "I can provide information on any topic",
                            },
                            {
                                "label": "Create a chart",
                                "description": "I can generate a chart using Chart.js",
                            },
                        ],
                    }
                ],
            },
        },
        separators=(",", ":"),
    )


def bad_show_visualization_json() -> str:
    return json.dumps(
        {
            "name": "show_visualization",
            "parameters": {
                "title": "Capabilities",
                "html": "<!DOCTYPE html><html><body><p>The answer to 2+2 is 4.</p></body></html>",
            },
        },
        separators=(",", ":"),
    )


def response_pieces(user_text: str) -> list[str]:
    lower = user_text.lower()
    if "what is 2+2" in lower or lower == "2+2":
        # Weak model prefixes prose before the tool JSON blob.
        text = "Sure! " + bad_ask_user_json()
        return list(text)
    if lower in {"hi", "hmm", "hello"}:
        return ["Hello! How can I help you today?"]
    return [f"Mock echo: {user_text}"]


def user_requested_explicit_choices(user_text: str) -> bool:
    lower = user_text.lower()
    phrases = (
        "which do you prefer",
        "which would you prefer",
        "choose between",
        "pick one",
        "pick between",
        "multiple choice",
        "select one",
    )
    return any(phrase in lower for phrase in phrases)


def native_ask_user_args(user_text: str) -> dict[str, Any]:
    lower = user_text.lower()
    if "what can you do" in lower or "what can u do" in lower or "what can u fo" in lower:
        return {
            "message": "I can perform various tasks. Which one would you like me to do?",
            "questions": [
                {
                    "header": "Choose an action",
                    "options": [
                        {
                            "label": "Answer a question",
                            "description": "I can provide information on any topic",
                        },
                        {
                            "label": "Create a chart",
                            "description": "I can generate a chart using Chart.js",
                        },
                    ],
                }
            ],
        }
    return {
        "title": "Math calculation",
        "questions": [
            {
                "question": "What is 2+2?",
                "options": [
                    {"label": "4", "description": "Correct"},
                    {"label": "5", "description": "Incorrect"},
                ],
            }
        ],
    }


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt: str, *args: Any) -> None:
        return

    def do_GET(self) -> None:
        if self.path != "/api/tags":
            self.send_error(404)
            return
        body = json.dumps({"models": [{"name": MODEL}]}).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self) -> None:
        if self.path != "/api/chat":
            self.send_error(404)
            return
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length)
        try:
            req = json.loads(raw.decode("utf-8"))
        except json.JSONDecodeError:
            self.send_error(400)
            return
        user_text = last_user_message(req.get("messages") or [])
        has_tools = bool(req.get("tools"))
        messages = req.get("messages") or []
        if not has_tools:
            stream_chunks(self, plain_text_reply(user_text))
            return
        if has_tool_results_since_last_user(messages):
            stream_chunks(self, plain_text_reply(user_text))
            return
        lower = user_text.lower()
        if user_requested_explicit_choices(user_text):
            stream_native_ask_user(self, native_ask_user_args(user_text))
            return
        if "what can you do" in lower or "what can u do" in lower or "what can u fo" in lower:
            stream_native_ask_user(self, native_ask_user_args(user_text))
            return
        if "capital of france" in lower:
            stream_native_tool_call(
                self,
                "show_visualization",
                {
                    "title": "Answer",
                    "html": "<!DOCTYPE html><html><body><p>France's capital is Paris.</p></body></html>",
                },
            )
            return
        stream_chunks(self, response_pieces(user_text))


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=11435)
    args = parser.parse_args()
    server = ThreadingHTTPServer(("127.0.0.1", args.port), Handler)
    print(f"ollama e2e mock listening on http://127.0.0.1:{args.port}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
