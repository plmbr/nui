# Docker Agent Example

A Loop extension that runs inside a Docker container. The agent exposes an
HTTP/SSE server on a fixed container port; Loop maps it to a random host port
at runtime.

## Files

- `echo_agent.py` — the agent implementation (subclasses `LoopAgent`)
- `loop_agent.py` — HTTP server framework (`GET /info`, `POST /run`, `POST /cancel`)
- `Dockerfile` — builds the container image

## Setup

```sh
# Build the image — this is the only manual step required
docker build -t loop-echo-agent .
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
# Run the container manually
docker run --rm -p 127.0.0.1:9090:9090 loop-echo-agent

# In another terminal — check info
curl http://127.0.0.1:9090/info

# Stream a response
curl -N -X POST http://127.0.0.1:9090/run \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

## How it works

### HTTP/SSE protocol

| Endpoint | Description |
|---|---|
| `GET /info` | Returns `{"name", "version", "capabilities"}` — also used as a health check |
| `POST /run` | Body: `{"message", "sessionId"?, "workingDir"?}`; response: `text/event-stream` |
| `POST /cancel` | Body: `{"runId"}`; stops the current run best-effort |

SSE events are JSON-encoded `data:` lines:

```
data: {"type":"text","content":"..."}
data: {"type":"done","sessionId":"..."}
data: {"type":"error","error":"..."}
```

### Lifecycle

| Event | What Loop does |
|---|---|
| Project created | `docker run -d -p 127.0.0.1::<containerPort> <image>` |
| Chat message sent | `POST /run` → SSE stream |
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

    def run(self, message: str, **kwargs):
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
