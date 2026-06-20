"""Parse OpenCode run --format json output into structured harness events."""

from __future__ import annotations

import json
from typing import Any, Generator, Iterator

from claude_stream import _emit_image_events


def parse_opencode_stream(lines: Iterator[str]) -> Generator[dict[str, Any], None, None]:
    parser = new_opencode_stream_parser()
    for line in lines:
        line = line.strip()
        if not line:
            continue
        try:
            obj = json.loads(line)
        except json.JSONDecodeError:
            continue
        yield from parser.handle(obj)


def new_opencode_stream_parser() -> "_OpenCodeStreamParser":
    return _OpenCodeStreamParser()


class _OpenCodeStreamParser:
    def __init__(self) -> None:
        self.seen_tool_starts: set[str] = set()
        self.seen_tool_ends: set[str] = set()
        self.seen_tool_results: set[str] = set()

    def handle(self, obj: dict[str, Any]) -> Generator[dict[str, Any], None, None]:
        sid = obj.get("sessionID") or ""
        if sid:
            yield {"type": "session_id", "sessionId": sid}

        t = obj.get("type")
        if t == "text":
            part = obj.get("part") or {}
            if part.get("type") == "text":
                text = part.get("text") or ""
                if text:
                    yield {"type": "text", "content": text}
            return

        if t != "tool_use":
            return

        part = obj.get("part") or {}
        state = part.get("state") or {}
        if state.get("status") != "completed":
            return

        tool_id = part.get("callID") or part.get("id") or ""
        tool_name = part.get("tool") or ""
        tool_input = state.get("input") or {}
        tool_output = state.get("output")
        metadata = state.get("metadata") or {}

        if not tool_id:
            return

        if tool_id not in self.seen_tool_starts:
            self.seen_tool_starts.add(tool_id)
            yield {"type": "tool_call_start", "toolCallId": tool_id, "toolName": tool_name}

        args_json = json.dumps(tool_input)
        yield {"type": "tool_call_args", "toolCallId": tool_id, "toolArgs": args_json}

        if tool_id not in self.seen_tool_ends:
            self.seen_tool_ends.add(tool_id)
            yield {
                "type": "tool_call_end",
                "toolCallId": tool_id,
                "toolName": tool_name,
                "toolArgs": args_json,
            }

        if tool_id not in self.seen_tool_results:
            self.seen_tool_results.add(tool_id)
            result_payload: Any = tool_output
            if isinstance(tool_output, str):
                try:
                    result_payload = json.loads(tool_output)
                except json.JSONDecodeError:
                    result_payload = tool_output
            elif metadata:
                result_payload = {"output": tool_output, "metadata": metadata}

            content = result_payload if isinstance(result_payload, str) else json.dumps(result_payload)
            yield {"type": "tool_call_result", "toolCallId": tool_id, "content": content}
            yield from _emit_image_events(result_payload)
            yield from _emit_image_events(metadata)
            yield from _emit_image_events(state)
