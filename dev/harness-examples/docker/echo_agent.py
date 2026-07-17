#!/usr/bin/env python3
"""
Minimal echo agent — Docker edition.

Build the image, then create a nui project with Agent Type → Docker.
nui launches and manages the container automatically.

    docker build -t nui-echo-agent .
"""

from nui_agent import NuiAgent


class EchoAgent(NuiAgent):
    name = "echo-agent"
    version = "0.1.0"

    def run(self, message: str, **kwargs):
        yield f"Echo: {message}\n"


if __name__ == "__main__":
    EchoAgent().serve()
