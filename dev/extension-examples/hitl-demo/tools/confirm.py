#!/usr/bin/env python3
"""Custom MCP tool that confirms an action via Loop HITL REST API."""

import json
import os
import sys


def _find_sdk_dir() -> str | None:
    candidates = [
        os.environ.get("LOOP_HITL_SDK_DIR", ""),
        os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", "harness-sdk"),
        os.path.expanduser("~/.loop/harness-sdk"),
    ]
    for candidate in candidates:
        if candidate and os.path.isfile(os.path.join(candidate, "loop_hitl.py")):
            return os.path.abspath(candidate)
    return None


_sdk_dir = _find_sdk_dir()
if _sdk_dir is None:
    print("loop_hitl.py not found", file=sys.stderr)
    sys.exit(1)
sys.path.insert(0, _sdk_dir)

from loop_hitl import request_approval


args = json.load(sys.stdin)
message = args.get("message", "Confirm this action?")
title = args.get("title", "Confirm action")

resp = request_approval(title=title, message=message)
print(json.dumps(resp))
