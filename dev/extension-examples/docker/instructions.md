# Docker Agent Example

A Loop extension that runs inside a Docker container. The agent exposes an HTTP/SSE server; Loop maps the container port to a random host port at runtime.

## Files

- `echo_agent.py` — agent implementation (subclasses `LoopAgent`)
- `loop_agent.py` — HTTP server framework
- `Dockerfile` — container image

## Setup

```sh
cd dev/extension-examples/docker
docker build -t loop-echo-agent .
```

## Connecting via Loop UI

Docker agents are configured through **custom ADL**, not a built-in UI picker.

1. Ensure `~/.loop/agents/docker-echo.yaml` exists (auto-provisioned on first Loop start, or copy from the template in `internal/store/store.go`).
2. Create a new session.
3. Under **Custom Agents**, select **docker-echo**.
4. Click **Create**.

Loop validates the connector on create (`docker run` + `GET /info`), then connects on each chat message.

To use your own image, edit the ADL file:

```yaml
harness:
  type: docker
  image: loop-echo-agent    # your image name
  containerPort: 9090       # port your container listens on
```

## Smoke-testing without Loop

```sh
docker run --rm -p 127.0.0.1:9090:9090 loop-echo-agent

curl http://127.0.0.1:9090/info

curl -N -X POST http://127.0.0.1:9090/run \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

## HTTP/SSE protocol

| Endpoint | Description |
|---|---|
| `GET /info` | `{"name", "version", "capabilities"}` — health check |
| `POST /run` | Body: `{message, sessionId?, workingDir?, systemPrompt?, model?}` → SSE |
| `POST /cancel` | Body: `{runId}` — cancel run best-effort |
| `POST /shutdown` | Stop server; Loop calls before `docker stop` |

SSE events:

```
data: {"type":"text","content":"..."}
data: {"type":"done","sessionId":"..."}
data: {"type":"error","error":"..."}
```

## Lifecycle

| Event | What Loop does |
|---|---|
| Session create | `docker run` + `GET /info` health check |
| Chat message | `POST /run` → SSE stream |
| Session delete | `POST /shutdown` + `docker stop` |
| Server shutdown | Stop all managed containers |

## Port conventions

| Image type | Container port |
|---|---|
| User extension examples (this directory) | **9090** |
| Builtin sandbox images (`docker/` in repo root) | **8090** |

## Writing your own Docker agent

```python
from loop_agent import LoopAgent

class MyAgent(LoopAgent):
    name = "my-agent"
    version = "0.1.0"

    def run(self, message: str, **kwargs):
        yield f"You said: {message}\n"

if __name__ == "__main__":
    MyAgent().serve()
```

```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY loop_agent.py my_agent.py ./
EXPOSE 9090
CMD ["python3", "my_agent.py", "--port", "9090"]
```

Then create an ADL file in `~/.loop/agents/` pointing at your image and port.
