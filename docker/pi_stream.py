"""Parse Pi CLI --mode json output into structured harness events."""

from __future__ import annotations

import json
from typing import Any, Generator, Iterator

from claude_stream import _emit_image_events


def parse_pi_stream(lines: Iterator[str]) -> Generator[dict[str, Any], None, None]:
    parser = new_pi_stream_parser()
    for line in lines:
        line = line.strip()
        if not line:
            continue
        try:
            obj = json.loads(line)
        except json.JSONDecodeError:
            continue
        yield from parser.handle(obj)


def new_pi_stream_parser() -> "_PiStreamParser":
    return _PiStreamParser()


class _PiStreamParser:
    def __init__(self) -> None:
        self.emitted_text = False
        self.needs_text_sep = False
        self.seen_tool_starts: set[str] = set()
        self.seen_tool_ends: set[str] = set()
        self.seen_tool_results: set[str] = set()

    def _mark_text_sep_needed(self) -> None:
        if self.emitted_text:
            self.needs_text_sep = True

    def _emit_text(self, text: str) -> Generator[dict[str, Any], None, None]:
        if not text:
            return
        if self.emitted_text and self.needs_text_sep and not text.startswith("\n"):
            text = "\n\n" + text
        self.needs_text_sep = False
        self.emitted_text = True
        yield {"type": "text", "content": text}

    def handle(self, obj: dict[str, Any]) -> Generator[dict[str, Any], None, None]:
        t = obj.get("type")

        if t == "session":
            sid = obj.get("id") or ""
            if sid:
                yield {"type": "session_id", "sessionId": sid}
            return

        if t == "message_update":
            yield from self._handle_message_update(obj)
            return

        if t == "tool_execution_start":
            self._mark_text_sep_needed()
            tool_id = obj.get("toolCallId") or ""
            tool_name = obj.get("toolName") or ""
            if tool_id and tool_id not in self.seen_tool_starts:
                self.seen_tool_starts.add(tool_id)
                yield {"type": "tool_call_start", "toolCallId": tool_id, "toolName": tool_name}
            return

        if t == "tool_execution_end":
            yield from self._emit_tool(
                obj.get("toolCallId") or "",
                obj.get("toolName") or "",
                obj.get("args"),
                obj.get("result"),
            )
            return

        if t == "turn_end":
            for result in obj.get("toolResults") or []:
                if not isinstance(result, dict):
                    continue
                tool_id = result.get("toolCallId") or result.get("toolUseId") or result.get("id") or ""
                tool_name = result.get("toolName") or result.get("name") or ""
                content = result.get("content", result)
                yield from self._emit_tool(tool_id, tool_name, None, content)

    def _handle_message_update(self, obj: dict[str, Any]) -> Generator[dict[str, Any], None, None]:
        ev = obj.get("assistantMessageEvent") or {}
        et = ev.get("type")

        if et == "text_delta":
            delta = ev.get("delta") or ""
            if delta:
                yield from self._emit_text(delta)
            return

        if et == "toolcall_start":
            self._mark_text_sep_needed()
            tool_call = ev.get("toolCall") or {}
            tool_id = tool_call.get("id") or ""
            tool_name = tool_call.get("name") or ""
            if tool_id and tool_id not in self.seen_tool_starts:
                self.seen_tool_starts.add(tool_id)
                yield {"type": "tool_call_start", "toolCallId": tool_id, "toolName": tool_name}
            return

        if et == "toolcall_end":
            tool_call = ev.get("toolCall") or {}
            tool_id = tool_call.get("id") or ""
            tool_name = tool_call.get("name") or ""
            args = tool_call.get("arguments") or tool_call.get("input") or {}
            yield from self._emit_tool(tool_id, tool_name, args, None)

    def _emit_tool(
        self,
        tool_id: str,
        tool_name: str,
        args: Any,
        result: Any,
    ) -> Generator[dict[str, Any], None, None]:
        if not tool_id:
            return

        self._mark_text_sep_needed()

        if tool_id not in self.seen_tool_starts:
            self.seen_tool_starts.add(tool_id)
            yield {"type": "tool_call_start", "toolCallId": tool_id, "toolName": tool_name}

        if args is not None and tool_id not in self.seen_tool_ends:
            args_json = json.dumps(args) if not isinstance(args, str) else args
            yield {"type": "tool_call_args", "toolCallId": tool_id, "toolArgs": args_json}

        if tool_id not in self.seen_tool_ends:
            self.seen_tool_ends.add(tool_id)
            end_args = ""
            if args is not None:
                end_args = json.dumps(args) if not isinstance(args, str) else args
            yield {
                "type": "tool_call_end",
                "toolCallId": tool_id,
                "toolName": tool_name,
                "toolArgs": end_args,
            }

        if result is not None and tool_id not in self.seen_tool_results:
            self.seen_tool_results.add(tool_id)
            content = result if isinstance(result, str) else json.dumps(result)
            yield {"type": "tool_call_result", "toolCallId": tool_id, "content": content}
            yield from _emit_image_events(result)
