#!/usr/bin/env python3
"""
Minimal echo agent — remote edition.

Listens on a fixed port so Loop (or any JSON-RPC client) can connect by
host:port. Works locally for testing and on any reachable host in production.

Run:
    python3 echo_agent.py --port 9090

Then create a Loop project with:
    Agent Type: Remote
    Host: 127.0.0.1   (or the remote IP)
    Port: 9090
"""

from loop_agent import LoopAgent


class EchoAgent(LoopAgent):
    name = "echo-agent"
    version = "0.1.0"

    def run(self, message: str, run_id: str, **kwargs):
        yield f"Echo: {message}\n"


if __name__ == "__main__":
    EchoAgent().serve()
