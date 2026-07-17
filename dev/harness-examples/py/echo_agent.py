#!/usr/bin/env python3
from nui_agent import NuiAgent


class EchoAgent(NuiAgent):
    name = "echo-agent"
    version = "0.1.0"

    def run(self, message: str, run_id: str, **kwargs):
        yield f"Echo: {message}\n"


if __name__ == "__main__":
    EchoAgent().serve()
