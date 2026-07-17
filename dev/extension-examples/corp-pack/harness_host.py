#!/usr/bin/env python3
"""Multiplex harness host — dispatches by NUI_HARNESS_ID / harnessId param."""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "..", "harness-sdk"))

from nui_agent_stdio import NuiAgent


class HarnessHost(NuiAgent):
    name = "corp-pack-host"
    version = "1.0.0"

    def run(self, message: str, run_id: str, **kwargs):
        harness_id = kwargs.get("harnessId") or os.environ.get("NUI_HARNESS_ID", "echo")
        if harness_id == "reverse":
            yield message[::-1]
        else:
            yield f"You said: {message}"


if __name__ == "__main__":
    HarnessHost().serve_stdio()
