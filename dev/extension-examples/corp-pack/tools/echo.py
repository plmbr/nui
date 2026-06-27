#!/usr/bin/env python3
"""Example command-tool for corp-pack custom MCP server."""

import json
import sys

args = json.load(sys.stdin)
message = args.get("message", "")
print(f"echo: {message}")
