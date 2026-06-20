"""Parse Claude CLI stream-json output into structured harness events."""

from __future__ import annotations

import json
from typing import Any, Generator, Iterator


def parse_claude_stream(lines: Iterator[str]) -> Generator[dict[str, Any], None, None]:
    parser = _ClaudeStreamParser()
    for line in lines:
        line = line.strip()
        if not line:
            continue
        try:
            envelope = json.loads(line)
        except json.JSONDecodeError:
            continue
        yield from parser.handle_envelope(envelope)


class _ClaudeStreamParser:
    def __init__(self) -> None:
        self.emitted_text = False
        self.seen_tool_starts: set[str] = set()
        self.seen_tool_ends: set[str] = set()
        self.seen_tool_results: set[str] = set()
        self.blocks: dict[int, dict[str, Any]] = {}

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
                self.emitted_text = True
                yield {"type": "text", "content": delta["text"]}
            elif delta.get("type") == "input_json_delta" and delta.get("partial_json"):
                idx = ev.get("index")
                state = self.blocks.get(idx)
                if state and state.get("kind") == "tool_use":
                    state["args"] += delta["partial_json"]
        elif ev_type == "content_block_start":
            block = ev.get("content_block") or {}
            if block.get("type") != "tool_use" or not block.get("id"):
                return
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

    def _handle_assistant(self, blocks: list[Any]) -> Generator[dict[str, Any], None, None]:
        for block in blocks:
            if not isinstance(block, dict):
                continue
            btype = block.get("type")
            if btype == "text" and block.get("text") and not self.emitted_text:
                yield {"type": "text", "content": block["text"]}
            elif btype == "tool_use" and block.get("id"):
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
                md = _image_block_markdown(block)
                if md:
                    yield {"type": "text", "content": md}

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

        if isinstance(content, list):
            for block in content:
                if not isinstance(block, dict) or block.get("type") != "tool_result":
                    continue
                tid = block.get("tool_use_id") or ""
                if not tid:
                    continue
                yield from self._emit_tool_result(tid, block.get("content"))
                for md in _image_content_markdown(block.get("content")):
                    yield {"type": "text", "content": md}

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


def _image_block_markdown(block: dict[str, Any]) -> str:
    source = block.get("source") or {}
    return _image_source_markdown(
        source.get("type") or "",
        source.get("media_type") or "",
        source.get("data") or "",
        source.get("url") or "",
    )


def _image_content_markdown(content: Any) -> list[str]:
    out: list[str] = []
    if isinstance(content, list):
        for block in content:
            if isinstance(block, dict) and block.get("type") == "image":
                md = _image_block_markdown(block)
                if md:
                    out.append(md)
        return out
    if isinstance(content, dict):
        _walk_images(content, out)
    return out


def _walk_images(value: Any, out: list[str]) -> None:
    if isinstance(value, dict):
        if value.get("type") == "image":
            source = value.get("source") or {}
            md = _image_source_markdown(
                source.get("type") or "",
                source.get("media_type") or "",
                source.get("data") or "",
                source.get("url") or "",
            )
            if md:
                out.append(md)
        for child in value.values():
            _walk_images(child, out)
    elif isinstance(value, list):
        for child in value:
            _walk_images(child, out)


def _image_source_markdown(src_type: str, media_type: str, data: str, url: str) -> str:
    if src_type == "base64" and data:
        media_type = media_type or "image/png"
        return f"\n\n![image](data:{media_type};base64,{data})\n\n"
    if url:
        return f"\n\n![image]({url})\n\n"
    return ""
