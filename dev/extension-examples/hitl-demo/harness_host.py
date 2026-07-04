#!/usr/bin/env python3
"""Extension harness that asks the user a structured question via ask_user()."""

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
        if os.path.isfile(os.path.join(norm, "loop_agent_stdio.py")):
            return norm
    return None


_sdk_dir = _find_sdk_dir()
if _sdk_dir is None:
    sys.stderr.write("loop_agent_stdio.py not found (set LOOP_HITL_SDK_DIR)\n")
    sys.exit(1)
sys.path.insert(0, _sdk_dir)

from loop_agent_stdio import LoopAgent


class HitlAskHarness(LoopAgent):
    name = "hitl-demo-harness"
    version = "1.0.0"

    def run(self, message: str, run_id: str, **kwargs):
        yield "I'll ask you a quick question before continuing.\n\n"
        resp = self.ask_user(
            title="Demo question",
            message=message or "What should I do next?",
            questions=[{
                "question": "Pick an option",
                "header": "Next step",
                "options": [
                    {"label": "Continue", "description": "Proceed with the task"},
                    {"label": "Stop", "description": "Cancel and explain why"},
                ],
            }],
            session_id=os.environ.get("LOOP_SESSION_ID", ""),
            run_id=run_id,
        )
        yield f"\n\nYou responded: {json.dumps(resp.get('answers', resp), indent=2)}\n"


if __name__ == "__main__":
    HitlAskHarness().serve_stdio()
