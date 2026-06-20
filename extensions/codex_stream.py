"""Parse Codex exec --json output into structured harness events."""

from __future__ import annotations

import json
from typing import Any, Generator, Iterator

from claude_stream import _emit_image_events


def parse_codex_stream(lines: Iterator[str]) -> Generator[dict[str, Any], None, None]:
    parser = new_codex_stream_parser()
    for line in lines:
        line = line.strip()
        if not line:
            continue
        try:
            envelope = json.loads(line)
        except json.JSONDecodeError:
            continue
        yield from parser.handle(envelope)


def new_codex_stream_parser() -> "_CodexStreamParser":
    return _CodexStreamParser()


class _CodexStreamParser:
    def __init__(self) -> None:
        self.seen_tool_starts: set[str] = set()
        self.seen_tool_ends: set[str] = set()
        self.seen_tool_results: set[str] = set()

    def handle(self, envelope: dict[str, Any]) -> Generator[dict[str, Any], None, None]:
        t = envelope.get("type") or ""

        if t == "thread.started":
            thread_id = envelope.get("thread_id") or ""
            if thread_id:
                yield {"type": "session_id", "sessionId": thread_id}
            return

        if t in ("item.started", "item.updated", "item.completed"):
            yield from self._handle_item(t, envelope.get("item") or {})
            return

        if t == "error":
            msg = envelope.get("message") or (envelope.get("error") or {}).get("message") or ""
            if msg:
                yield {"type": "error", "error": msg}
            return

        if t == "turn.failed":
            msg = envelope.get("message") or (envelope.get("error") or {}).get("message") or "turn failed"
            yield {"type": "error", "error": msg}

    def _handle_item(self, event_type: str, item: dict[str, Any]) -> Generator[dict[str, Any], None, None]:
        item_type = item.get("type") or ""

        if item_type == "agent_message":
            if event_type == "item.completed":
                text = item.get("text") or ""
                if text:
                    yield {"type": "text", "content": text}
            return

        if item_type != "mcp_tool_call":
            return

        tool_id = item.get("id") or ""
        if not tool_id:
            return

        server = item.get("server") or ""
        tool = item.get("tool") or ""
        tool_name = f"{server}/{tool}" if server else tool

        if event_type == "item.started":
            if tool_id not in self.seen_tool_starts:
                self.seen_tool_starts.add(tool_id)
                yield {"type": "tool_call_start", "toolCallId": tool_id, "toolName": tool_name}
            return

        if event_type != "item.completed":
            return

        if tool_id not in self.seen_tool_starts:
            self.seen_tool_starts.add(tool_id)
            yield {"type": "tool_call_start", "toolCallId": tool_id, "toolName": tool_name}

        args = item.get("arguments")
        if args is None:
            args_json = "{}"
        elif isinstance(args, str):
            args_json = args
        else:
            args_json = json.dumps(args)
        yield {"type": "tool_call_args", "toolCallId": tool_id, "toolArgs": args_json}

        if tool_id not in self.seen_tool_ends:
            self.seen_tool_ends.add(tool_id)
            yield {
                "type": "tool_call_end",
                "toolCallId": tool_id,
                "toolName": tool_name,
                "toolArgs": args_json,
            }

        if tool_id in self.seen_tool_results:
            return
        self.seen_tool_results.add(tool_id)

        status = item.get("status") or ""
        err = (item.get("error") or {}).get("message") or ""
        if status == "failed" or err:
            yield {"type": "tool_call_result", "toolCallId": tool_id, "content": err or "tool failed"}
            return

        result = item.get("result")
        if result is not None:
            content = result if isinstance(result, str) else json.dumps(result)
            yield {"type": "tool_call_result", "toolCallId": tool_id, "content": content}
            yield from _emit_image_events(result)
