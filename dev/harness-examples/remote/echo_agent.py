#!/usr/bin/env python3
"""
Minimal echo agent — remote edition.

Listens on a fixed port so nui (or any JSON-RPC client) can connect by
host:port. Works locally for testing and on any reachable host in production.

Run:
    python3 echo_agent.py --port 9090

Then create a nui project with:
    Agent Type: Remote
    Host: 127.0.0.1   (or the remote IP)
    Port: 9090
"""

from nui_agent import NuiAgent


class EchoAgent(NuiAgent):
    name = "echo-agent"
    version = "0.1.0"

    def run(self, message: str, **kwargs):
        yield f"Echo: {message}\n"


if __name__ == "__main__":
    EchoAgent().serve()
