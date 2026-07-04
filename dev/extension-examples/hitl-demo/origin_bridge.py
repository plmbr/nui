#!/usr/bin/env python3
"""
REST-only HITL origin bridge example.

Polls Loop for pending requests routed to an extension channel and auto-responds.
Run alongside Loop — no stdio JSON-RPC runtime required for this channel.

    LOOP_API_URL=http://127.0.0.1:8080 \\
    HITL_CHANNEL_ID=ext:hitl-demo/demo-webhook \\
    python3 origin_bridge.py
"""

import json
import os
import sys
import time
import urllib.error
import urllib.request


def api_url() -> str:
    for key in ("LOOP_API_URL", "LOOP_URL"):
        value = os.environ.get(key, "").strip()
        if value:
            return value.rstrip("/")
    return "http://127.0.0.1:8080"


def _get(path: str):
    req = urllib.request.Request(api_url() + path, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())


def _post(path: str, body: dict):
    data = json.dumps(body).encode()
    req = urllib.request.Request(
        api_url() + path,
        data=data,
        headers={"Content-Type": "application/json", "Accept": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())


def list_pending() -> list[dict]:
    return _get("/api/hitl/requests?pending=true")


def respond(request_id: str, channel: str, answers: dict):
    return _post(f"/api/hitl/requests/{request_id}/respond", {
        "status": "answered",
        "answers": answers,
        "respondedBy": {"channel": channel, "actor": "origin-bridge"},
    })


def routed_to(request: dict, channel_id: str) -> bool:
    channels = (request.get("routing") or {}).get("channels") or []
    return channel_id in channels


def main() -> int:
    channel_id = os.environ.get("HITL_CHANNEL_ID", "ext:hitl-demo/demo-webhook")
    poll_seconds = float(os.environ.get("HITL_POLL_SECONDS", "2"))
    seen: set[str] = set()

    sys.stderr.write(f"[origin-bridge] watching {channel_id} at {api_url()}\n")
    while True:
        try:
            for req in list_pending():
                rid = req.get("requestId", "")
                if not rid or rid in seen:
                    continue
                if not routed_to(req, channel_id):
                    continue
                seen.add(rid)
                payload = req.get("payload") or {}
                title = payload.get("title") or payload.get("message") or req.get("kind", "")
                sys.stderr.write(f"[origin-bridge] auto-answer {rid}: {title!r}\n")
                respond(rid, channel_id, {"auto": True, "note": "answered by origin_bridge.py demo"})
        except urllib.error.URLError as exc:
            sys.stderr.write(f"[origin-bridge] poll error: {exc}\n")
        time.sleep(poll_seconds)


if __name__ == "__main__":
    raise SystemExit(main())
