"""Parse Claude CLI stream-json output into structured harness events."""

from __future__ import annotations

import json
from typing import Any, Generator, Iterator


def parse_claude_stream(lines: Iterator[str]) -> Generator[dict[str, Any], None, None]:
    parser = new_claude_stream_parser()
    for line in lines:
        line = line.strip()
        if not line:
            continue
        try:
            envelope = json.loads(line)
        except json.JSONDecodeError:
            continue
        yield from parser.handle_envelope(envelope)


def new_claude_stream_parser() -> "_ClaudeStreamParser":
    return _ClaudeStreamParser()


class _ClaudeStreamParser:
    def __init__(self) -> None:
        self.emitted_text = False
        self.needs_text_sep = False
        self.seen_tool_starts: set[str] = set()
        self.seen_tool_ends: set[str] = set()
        self.seen_tool_results: set[str] = set()
        self.blocks: dict[int, dict[str, Any]] = {}

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

    def handle_envelope(self, envelope: dict[str, Any]) -> Generator[dict[str, Any], None, None]:
        t = envelope.get("type")
        if t == "stream_event":
            yield from self._handle_stream_event(envelope.get("event") or {})
        elif t == "assistant":
            yield from self._handle_assistant((envelope.get("message") or {}).get("content") or [])
        elif t == "user":
            yield from self._handle_user(envelope)
        elif t == "result":
            if envelope.get("is_error"):
                yield {"type": "error", "error": envelope.get("error") or "unknown error"}
            else:
                sid = envelope.get("session_id") or ""
                if sid:
                    yield {"type": "session_id", "sessionId": sid}

    def _handle_stream_event(self, ev: dict[str, Any]) -> Generator[dict[str, Any], None, None]:
        ev_type = ev.get("type")
        if ev_type == "content_block_delta":
            delta = ev.get("delta") or {}
            if delta.get("type") == "text_delta" and delta.get("text"):
                yield from self._emit_text(delta["text"])
            elif delta.get("type") == "input_json_delta" and delta.get("partial_json"):
                idx = ev.get("index")
                state = self.blocks.get(idx)
                if state and state.get("kind") == "tool_use":
                    state["args"] += delta["partial_json"]
        elif ev_type == "content_block_start":
            block = ev.get("content_block") or {}
            if block.get("type") == "text":
                self._mark_text_sep_needed()
                return
            if block.get("type") != "tool_use" or not block.get("id"):
                return
            self._mark_text_sep_needed()
            idx = ev.get("index")
            self.blocks[idx] = {
                "kind": "tool_use",
                "tool_id": block["id"],
                "tool_name": block.get("name") or "",
                "args": "",
            }
            if block["id"] in self.seen_tool_starts:
                return
            self.seen_tool_starts.add(block["id"])
            yield {
                "type": "tool_call_start",
                "toolCallId": block["id"],
                "toolName": block.get("name") or "",
            }
        elif ev_type == "content_block_stop":
            idx = ev.get("index")
            state = self.blocks.pop(idx, None)
            if not state or state.get("kind") != "tool_use":
                return
            args = state.get("args") or ""
            if args:
                yield {
                    "type": "tool_call_args",
                    "toolCallId": state["tool_id"],
                    "toolArgs": args,
                }
            if state["tool_id"] not in self.seen_tool_ends:
                self.seen_tool_ends.add(state["tool_id"])
                yield {
                    "type": "tool_call_end",
                    "toolCallId": state["tool_id"],
                    "toolName": state["tool_name"],
                    "toolArgs": args,
                }
            self._mark_text_sep_needed()

    def _handle_assistant(self, blocks: list[Any]) -> Generator[dict[str, Any], None, None]:
        for block in blocks:
            if not isinstance(block, dict):
                continue
            btype = block.get("type")
            if btype == "text" and block.get("text") and not self.emitted_text:
                yield from self._emit_text(block["text"])
            elif btype == "tool_use" and block.get("id"):
                self._mark_text_sep_needed()
                tool_id = block["id"]
                args = json.dumps(block.get("input") or {})
                if tool_id not in self.seen_tool_starts:
                    self.seen_tool_starts.add(tool_id)
                    yield {
                        "type": "tool_call_start",
                        "toolCallId": tool_id,
                        "toolName": block.get("name") or "",
                    }
                    yield {"type": "tool_call_args", "toolCallId": tool_id, "toolArgs": args}
                if tool_id not in self.seen_tool_ends:
                    self.seen_tool_ends.add(tool_id)
                    yield {
                        "type": "tool_call_end",
                        "toolCallId": tool_id,
                        "toolName": block.get("name") or "",
                        "toolArgs": args,
                    }
            elif btype == "image":
                yield from _emit_image_events(block)

    def _handle_user(self, envelope: dict[str, Any]) -> Generator[dict[str, Any], None, None]:
        tool_use_id = envelope.get("parent_tool_use_id") or ""
        tool_use_result = envelope.get("tool_use_result")
        message = envelope.get("message") or {}
        content = message.get("content")

        if not tool_use_id and isinstance(content, list):
            for block in content:
                if isinstance(block, dict) and block.get("type") == "tool_result":
                    tool_use_id = block.get("tool_use_id") or tool_use_id

        if tool_use_id and tool_use_result is not None:
            yield from self._emit_tool_result(tool_use_id, tool_use_result)
            yield from _emit_image_events(tool_use_result)

        if isinstance(content, list):
            for block in content:
                if not isinstance(block, dict) or block.get("type") != "tool_result":
                    continue
                tid = block.get("tool_use_id") or ""
                if not tid:
                    continue
                yield from self._emit_tool_result(tid, block.get("content"))
                yield from _emit_image_events(block.get("content"))

    def _emit_tool_result(self, tool_use_id: str, result: Any) -> Generator[dict[str, Any], None, None]:
        if tool_use_id in self.seen_tool_results:
            return
        self.seen_tool_results.add(tool_use_id)
        if isinstance(result, str):
            content = result
        else:
            content = json.dumps(result)
        yield {
            "type": "tool_call_result",
            "toolCallId": tool_use_id,
            "content": content,
        }


def _extract_image_block(block: dict[str, Any]) -> tuple[str, str] | None:
    if block.get("type") != "image":
        return None

    data = block.get("data")
    if isinstance(data, str) and data:
        media_type = block.get("mimeType") or block.get("media_type") or "image/png"
        return media_type, data

    source = block.get("source") or {}
    if not isinstance(source, dict):
        return None

    src_type = source.get("type") or ""
    url = source.get("url") or ""
    media_type = source.get("media_type") or source.get("mediaType") or "image/png"
    b64 = source.get("data") or ""

    if (src_type == "base64" or b64) and b64:
        return media_type, b64
    if url:
        return media_type, url
    return None


def _extract_images(value: Any) -> list[tuple[str, str]]:
    out: list[tuple[str, str]] = []

    def walk(node: Any) -> None:
        if isinstance(node, dict):
            extracted = _extract_image_block(node)
            if extracted:
                out.append(extracted)
            for child in node.values():
                walk(child)
        elif isinstance(node, list):
            for child in node:
                walk(child)

    walk(value)
    return out


def _emit_image_events(value: Any) -> Generator[dict[str, Any], None, None]:
    for media_type, data in _extract_images(value):
        yield {
            "type": "image",
            "imageMediaType": media_type,
            "imageData": data,
        }
