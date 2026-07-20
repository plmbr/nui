import io
import json
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from nui_agent_stdio import NuiAgent


class EchoAgent(NuiAgent):
    name = "echo"
    version = "0.2.0"

    def run(self, message: str, run_id: str, **kwargs):
        yield f"echo:{message}"


def test_harness_info_response():
    agent = EchoAgent()
    req = {"jsonrpc": "2.0", "id": 1, "method": "harness.info", "params": {}}
    out = io.StringIO()
    old_stdout = sys.stdout
    sys.stdout = out
    try:
        agent._dispatch(req)
    finally:
        sys.stdout = old_stdout

    line = out.getvalue().strip()
    msg = json.loads(line)
    assert msg["result"]["name"] == "echo"
    assert msg["result"]["version"] == "0.2.0"
    assert "run" in msg["result"]["capabilities"]


def test_harness_run_emits_text_and_done():
    agent = EchoAgent()
    req = {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "harness.run",
        "params": {"message": "hello", "runId": "run-1"},
    }
    out = io.StringIO()
    old_stdout = sys.stdout
    sys.stdout = out
    try:
        agent._dispatch(req)
    finally:
        sys.stdout = old_stdout

    events = [json.loads(line) for line in out.getvalue().splitlines() if line.strip()]
    text_events = [e for e in events if e.get("method") == "harness.event" and e["params"].get("type") == "text"]
    done_events = [e for e in events if e.get("method") == "harness.event" and e["params"].get("type") == "done"]
    assert text_events[0]["params"]["content"] == "echo:hello"
    assert done_events
    assert events[-1]["result"]["runId"] == "run-1"
