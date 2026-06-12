# Docker Agent Example

A Loop extension that runs inside a Docker container. The agent binds to a
fixed port inside the container; Loop maps it to a random host port at runtime.

## Files

- `echo_agent.py` — the agent implementation (subclasses `LoopAgent`)
- `loop_agent.py` — server-mode framework (supports `--port` and `0.0.0.0` binding)
- `Dockerfile` — builds the container image

## Setup

```sh
# Build the image — this is the only manual step required
docker build -t loop-echo-agent dev/extension-examples/docker
```

## Connecting via Loop UI

1. Create a new project.
2. Choose **Agent Type → Docker**.
3. Set **Docker Image** to `loop-echo-agent`.
4. Set **Container Port** to `9090`.
5. Click **Create**.

Loop starts the container, resolves the mapped host port, and connects — no
manual `docker run` needed.

## Smoke-testing without Loop

```sh
docker run --rm -p 127.0.0.1::9090 loop-echo-agent &
# docker port <id> 9090  →  127.0.0.1:<hostPort>
echo '{"jsonrpc":"2.0","id":1,"method":"harness.info","params":{}}' \
  | nc 127.0.0.1 <hostPort>
```

## How it works

### Why `--port` and `0.0.0.0`?

Standard Loop extensions (`py/echo_agent.py`) bind to `127.0.0.1` on a random
port and write a connection file so the Go server can find them. Docker agents
can't use that mechanism — the container has its own filesystem and its own
loopback interface.

Instead:

1. The Dockerfile `CMD` passes `--port 9090` to the agent.
2. `loop_agent.py` detects the flag and binds to `0.0.0.0:9090` (reachable
   through Docker's port mapping).
3. No connection file is written; Loop discovers the host port externally via
   `docker port`.

### Lifecycle

| Event | What Loop does |
|---|---|
| Project created | `docker run -d -p 127.0.0.1::<containerPort> <image>` |
| Chat message sent | TCP connect → `harness.run` RPC → stream events |
| Project deleted | `docker stop <containerID>` |
| Server shutdown | `docker stop` all managed containers |

## Writing your own Docker agent

Replace `echo_agent.py` with your own logic:

```python
from loop_agent import LoopAgent
import subprocess

class MyAgent(LoopAgent):
    name = "my-agent"
    version = "0.1.0"

    def run(self, message: str, run_id: str, **kwargs):
        # workingDir and sessionId are passed as kwargs when present
        yield "Thinking...\n"
        result = subprocess.run(["my-tool", message], capture_output=True, text=True)
        yield result.stdout

if __name__ == "__main__":
    MyAgent().serve()
```

Update the `Dockerfile` to install any dependencies:

```dockerfile
FROM python:3.12-slim
WORKDIR /app
RUN pip install my-dependency
COPY loop_agent.py my_agent.py ./
EXPOSE 9090
CMD ["python3", "my_agent.py", "--port", "9090"]
```
