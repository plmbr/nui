#!/usr/bin/env python3
"""Example stdio HITL channel — logs deliveries to stderr."""

import json
import os
import sys


def _find_sdk_dir() -> str | None:
    candidates = []
    if sdk := os.environ.get("LOOP_HITL_SDK_DIR"):
        candidates.append(sdk)
    ext_dir = os.path.dirname(os.path.abspath(__file__))
    candidates.extend([
        ext_dir,
        os.path.join(ext_dir, "..", "..", "..", "harness-sdk"),
        os.path.expanduser("~/.loop/harness-sdk"),
    ])
    seen: set[str] = set()
    for candidate in candidates:
        if not candidate:
            continue
        norm = os.path.abspath(candidate)
        if norm in seen:
            continue
        seen.add(norm)
        if os.path.isfile(os.path.join(norm, "loop_hitl_channel.py")):
            return norm
    return None


_sdk_dir = _find_sdk_dir()
if _sdk_dir is None:
    sys.stderr.write("loop_hitl_channel.py not found (set LOOP_HITL_SDK_DIR)\n")
    sys.exit(1)
sys.path.insert(0, _sdk_dir)

from loop_hitl_channel import LoopHITLChannelProvider


class DemoHITLChannelHost(LoopHITLChannelProvider):
    name = "hitl-demo-channels"
    version = "1.0.0"

    def on_deliver(self, channel_id, request, **kwargs):
        payload = request.get("payload") or {}
        title = payload.get("title") or payload.get("message") or request.get("kind", "hitl")
        sys.stderr.write(
            f"[hitl-demo] deliver channel={channel_id} "
            f"requestId={request.get('requestId')} title={title!r}\n"
        )
        sys.stderr.flush()
        return {"ok": True, "delivered": True}


if __name__ == "__main__":
    DemoHITLChannelHost().serve()
