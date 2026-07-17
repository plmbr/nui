"""
nui command-tool MCP proxy — stdio MCP server.

Reads a tools definition JSON file and exposes each tool as an MCP tool that
runs a CLI command with JSON-serialized arguments on stdin.

Usage:
    python3 nui_mcp_tools.py /path/to/tools.json
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
from typing import Any


PROTOCOL_VERSION = "2024-11-05"
SERVER_NAME = "nui-mcp-tools"
SERVER_VERSION = "1.0.0"

DEFAULT_INPUT_SCHEMA: dict[str, Any] = {
    "type": "object",
    "properties": {
        "message": {
            "type": "string",
            "description": "Input text for the tool",
        }
    },
    "required": ["message"],
}


class MCPToolsServer:
    def __init__(self, config_path: str) -> None:
        with open(config_path, encoding="utf-8") as f:
            self.config = json.load(f)
        self.extension_dir = self.config.get("extensionDir", "")
        self.tools = {t["name"]: t for t in self.config.get("tools", [])}

    def _write(self, msg: dict[str, Any]) -> None:
        sys.stdout.write(json.dumps(msg) + "\n")
        sys.stdout.flush()

    def _tool_schema(self, tool: dict[str, Any]) -> dict[str, Any]:
        schema = tool.get("inputSchema")
        if isinstance(schema, dict) and schema.get("type") == "object":
            return schema
        return DEFAULT_INPUT_SCHEMA

    def _tool_list(self) -> list[dict[str, Any]]:
        out: list[dict[str, Any]] = []
        for tool in self.config.get("tools", []):
            out.append({
                "name": tool["name"],
                "description": tool.get("description") or tool["name"],
                "inputSchema": self._tool_schema(tool),
            })
        return out

    def _run_tool(self, name: str, arguments: dict[str, Any]) -> dict[str, Any]:
        tool = self.tools.get(name)
        if tool is None:
            return {
                "content": [{"type": "text", "text": f"unknown tool: {name}"}],
                "isError": True,
            }
        command = tool.get("command") or []
        if not command:
            return {
                "content": [{"type": "text", "text": f"tool {name}: command not configured"}],
                "isError": True,
            }
        cwd = self.extension_dir or None
        try:
            proc = subprocess.run(
                command,
                input=json.dumps(arguments).encode("utf-8"),
                capture_output=True,
                cwd=cwd,
                check=False,
            )
        except OSError as exc:
            return {
                "content": [{"type": "text", "text": f"failed to run tool {name}: {exc}"}],
                "isError": True,
            }

        parts: list[str] = []
        if proc.stdout:
            parts.append(proc.stdout.decode("utf-8", errors="replace"))
        if proc.stderr:
            stderr = proc.stderr.decode("utf-8", errors="replace")
            if stderr.strip():
                parts.append(stderr)
        text = "".join(parts).strip()
        if not text and proc.returncode == 0:
            text = "(no output)"
        is_error = proc.returncode != 0
        if is_error and not text:
            text = f"command exited with code {proc.returncode}"
        return {
            "content": [{"type": "text", "text": text}],
            "isError": is_error,
        }

    def _dispatch(self, req: dict[str, Any]) -> None:
        method = req.get("method", "")
        rid = req.get("id")
        params = req.get("params") or {}

        if method == "notifications/initialized":
            return

        if method == "initialize":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {
                    "protocolVersion": PROTOCOL_VERSION,
                    "capabilities": {"tools": {}},
                    "serverInfo": {"name": SERVER_NAME, "version": SERVER_VERSION},
                },
            })
            return

        if method == "tools/list":
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "result": {"tools": self._tool_list()},
            })
            return

        if method == "tools/call":
            name = params.get("name", "")
            arguments = params.get("arguments") or {}
            if not isinstance(arguments, dict):
                arguments = {}
            result = self._run_tool(name, arguments)
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if rid is not None:
            self._write({
                "jsonrpc": "2.0",
                "id": rid,
                "error": {"code": -32601, "message": f"method not found: {method}"},
            })

    def serve(self) -> None:
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            try:
                req = json.loads(line)
            except json.JSONDecodeError:
                continue
            self._dispatch(req)


def main() -> None:
    if len(sys.argv) != 2:
        print("usage: nui_mcp_tools.py <tools.json>", file=sys.stderr)
        sys.exit(2)
    config_path = sys.argv[1]
    if not os.path.isfile(config_path):
        print(f"tools config not found: {config_path}", file=sys.stderr)
        sys.exit(2)
    MCPToolsServer(config_path).serve()


if __name__ == "__main__":
    main()
