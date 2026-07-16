#!/usr/bin/env python3
"""Echo harness and agent example for programmatic Loop extensions."""

import os
import sys


def _find_sdk_dir() -> str:
    here = os.path.dirname(os.path.abspath(__file__))
    candidates = [
        os.path.join(here, "..", "..", "..", "harness-sdk"),
        os.path.join(here, "..", "..", "harness-sdk"),
        os.path.expanduser("~/.loop/harness-sdk"),
    ]
    for candidate in candidates:
        norm = os.path.abspath(candidate)
        if os.path.isfile(os.path.join(norm, "loop_extension.py")):
            return norm
    raise RuntimeError("loop_extension.py not found (install harness-sdk or run from repo)")


sys.path.insert(0, _find_sdk_dir())

from loop_extension import LoopExtension


class EchoExtension(LoopExtension):
    def get_harnesses(self):
        return [
            {"id": "echo", "displayName": "Echo Harness"}
        ]

    def get_agents(self):
        ext = self.extension_name or "programmatic-echo"
        return [
            {
                "id": "echo-agent",
                "name": "Echo Agent",
                "description": "An agent that echoes back whatever input it receives.",
                "harness": {"type": f"ext:{ext}/echo"},
            }
        ]

    def run_harness(self, harness_id, message, ctx=None):
        if harness_id == "echo":
            yield f"Echo: {message}"


def main():
    EchoExtension().serve()


if __name__ == "__main__":
    main()
