"""
Loop HITL REST client for harness authors and extension tools.

Creates requests via POST /api/hitl/requests and blocks on GET /api/hitl/requests/:id/wait.
"""

from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from typing import Any


DEFAULT_API_URL = "http://127.0.0.1:8080"


def api_url() -> str:
    for key in ("LOOP_API_URL", "LOOP_URL"):
        value = os.environ.get(key, "").strip()
        if value:
            return value.rstrip("/")
    return DEFAULT_API_URL


def _request(method: str, path: str, body: dict | None = None) -> Any:
    url = api_url() + path
    data = None
    headers = {"Accept": "application/json"}
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as resp:
            raw = resp.read().decode()
            if not raw:
                return {}
            return json.loads(raw)
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode().strip()
        raise RuntimeError(f"HITL {method} {path} failed ({exc.code}): {detail}") from exc


def create_request(
    *,
    kind: str = "question",
    payload: dict[str, Any] | None = None,
    session_id: str = "",
    run_id: str = "",
    routing: dict[str, Any] | None = None,
    ttl_seconds: int | None = None,
    request_id: str = "",
    correlation_id: str = "",
    step_name: str = "",
) -> dict[str, Any]:
    session_id = session_id or os.environ.get("LOOP_SESSION_ID", "")
    run_id = run_id or os.environ.get("LOOP_RUN_ID", "")
    body: dict[str, Any] = {
        "kind": kind,
        "payload": payload or {},
        "sessionId": session_id,
        "runId": run_id,
    }
    if routing:
        body["routing"] = routing
    if ttl_seconds is not None:
        body["ttlSeconds"] = ttl_seconds
    if request_id:
        body["requestId"] = request_id
    if correlation_id:
        body["correlationId"] = correlation_id
    if step_name:
        body["stepName"] = step_name
    return _request("POST", "/api/hitl/requests", body)


def wait_response(request_id: str) -> dict[str, Any]:
    return _request("GET", f"/api/hitl/requests/{request_id}/wait")


def respond(request_id: str, *, status: str = "answered", answers: dict[str, Any] | None = None, channel: str = "loop-ui") -> dict[str, Any]:
    body: dict[str, Any] = {
        "status": status,
        "respondedBy": {"channel": channel},
    }
    if answers is not None:
        body["answers"] = answers
    return _request("POST", f"/api/hitl/requests/{request_id}/respond", body)


def list_pending(*, session_id: str = "", run_id: str = "") -> list[dict[str, Any]]:
    query = "pending=true"
    if session_id:
        query += f"&sessionId={session_id}"
    if run_id:
        query += f"&runId={run_id}"
    out = _request("GET", f"/api/hitl/requests?{query}")
    return out if isinstance(out, list) else []


def ask_user(
    *,
    questions: list[dict[str, Any]] | None = None,
    title: str = "",
    message: str = "",
    session_id: str = "",
    run_id: str = "",
    routing: dict[str, Any] | None = None,
    emit_event=None,
) -> dict[str, Any]:
    payload = {
        "questions": questions or [],
        "title": title,
        "message": message,
    }
    req = create_request(
        kind="question",
        payload=payload,
        session_id=session_id,
        run_id=run_id,
        routing=routing,
    )
    if emit_event:
        emit_event(req)
    return wait_response(req["requestId"])


def request_approval(
    *,
    title: str = "",
    message: str = "",
    tool_name: str = "",
    tool_input: dict[str, Any] | None = None,
    description: str = "",
    session_id: str = "",
    run_id: str = "",
    routing: dict[str, Any] | None = None,
    emit_event=None,
) -> dict[str, Any]:
    payload = {
        "title": title,
        "message": message,
        "toolName": tool_name,
        "toolInput": tool_input or {},
        "description": description,
    }
    req = create_request(
        kind="approval",
        payload=payload,
        session_id=session_id,
        run_id=run_id,
        routing=routing,
    )
    if emit_event:
        emit_event(req)
    return wait_response(req["requestId"])
