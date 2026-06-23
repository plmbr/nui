#!/usr/bin/env python3
"""
Minimal echo agent — Docker edition.

Build the image, then create a Loop project with Agent Type → Docker.
Loop launches and manages the container automatically.

    docker build -t loop-echo-agent .
"""

from loop_agent import LoopAgent


class EchoAgent(LoopAgent):
    name = "echo-agent"
    version = "0.1.0"

    def run(self, message: str, **kwargs):
        yield f"Echo: {message}\n"


if __name__ == "__main__":
    EchoAgent().serve()
