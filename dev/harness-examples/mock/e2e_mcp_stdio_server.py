#!/usr/bin/env python3
"""
Minimal stdio MCP server for nui local Playwright E2E tests.

Exposes a single ping tool that returns "pong".
"""

from __future__ import annotations

import json
import sys
from typing import Any

PROTOCOL_VERSION = "2024-11-05"
SERVER_NAME = "e2e-mcp"
SERVER_VERSION = "1.0.0"

TOOLS: list[dict[str, Any]] = [
    {
        "name": "ping",
        "description": "Health check — returns pong",
        "inputSchema": {
            "type": "object",
            "properties": {},
            "additionalProperties": False,
        },
    }
]


def write(msg: dict[str, Any]) -> None:
    sys.stdout.write(json.dumps(msg) + "\n")
    sys.stdout.flush()


def dispatch(req: dict[str, Any]) -> None:
    method = req.get("method")
    rid = req.get("id")
    params = req.get("params") or {}

    if method == "notifications/initialized":
        return

    if method == "initialize":
        write(
            {
                "jsonrpc": "2.0",
                "id": rid,
                "result": {
                    "protocolVersion": PROTOCOL_VERSION,
                    "capabilities": {"tools": {}},
                    "serverInfo": {"name": SERVER_NAME, "version": SERVER_VERSION},
                },
            }
        )
        return

    if method == "tools/list":
        write({"jsonrpc": "2.0", "id": rid, "result": {"tools": TOOLS}})
        return

    if method == "tools/call":
        name = params.get("name", "")
        if name == "ping":
            write(
                {
                    "jsonrpc": "2.0",
                    "id": rid,
                    "result": {
                        "content": [{"type": "text", "text": "pong"}],
                        "isError": False,
                    },
                }
            )
            return

    if rid is not None:
        write(
            {
                "jsonrpc": "2.0",
                "id": rid,
                "error": {"code": -32601, "message": f"method not found: {method}"},
            }
        )


def main() -> None:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            dispatch(json.loads(line))
        except json.JSONDecodeError:
            continue


if __name__ == "__main__":
    main()
