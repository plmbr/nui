#!/usr/bin/env python3
from loop_agent import LoopAgent


class EchoAgent(LoopAgent):
    name = "echo-agent"
    version = "0.1.0"

    def run(self, message: str, run_id: str):
        yield f"Echo: {message}\n"


if __name__ == "__main__":
    EchoAgent().serve()
