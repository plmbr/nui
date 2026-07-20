import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from nui_hitl import api_url, resolve_nui_run_id, resolve_nui_session_id


def test_resolve_nui_session_id_prefers_explicit():
    assert resolve_nui_session_id("sess-1", {"sessionId": "other"}) == "sess-1"


def test_resolve_nui_session_id_from_ctx_nui_session_id():
    assert resolve_nui_session_id(ctx={"nuiSessionId": "nui-42"}) == "nui-42"


def test_resolve_nui_session_id_from_env(monkeypatch):
    monkeypatch.setenv("NUI_SESSION_ID", "env-sess")
    assert resolve_nui_session_id() == "env-sess"


def test_resolve_nui_run_id_from_ctx():
    assert resolve_nui_run_id(ctx={"runId": "run-9"}) == "run-9"


def test_api_url_defaults(monkeypatch):
    monkeypatch.delenv("NUI_API_URL", raising=False)
    monkeypatch.delenv("NUI_URL", raising=False)
    assert api_url() == "http://127.0.0.1:8080"
    monkeypatch.setenv("NUI_URL", "http://example:9000")
    assert api_url() == "http://example:9000"
