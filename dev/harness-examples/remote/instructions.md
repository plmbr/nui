# Remote Harness Example

A standalone HTTP/SSE server — on the same machine or a remote host. nui connects via a custom ADL agent; it does not start or stop the process.

## Files

- `echo_agent.py` — agent implementation (subclasses `NuiAgent`)
- `nui_agent.py` — HTTP server framework

## Running

```sh
python3 dev/harness-examples/remote/echo_agent.py --port 9090

curl http://127.0.0.1:9090/info

curl -N -X POST http://127.0.0.1:9090/run \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

## Connecting via nui UI

Remote agents are configured through **custom ADL**, not a built-in UI picker.

1. Start the agent server (command above).
2. Copy the example ADL into your agents directory:

```sh
cp dev/harness-examples/remote/remote-echo.yaml ~/.nui/agents/
```

3. Create a new session.
4. Under **Installed agents**, select **remote-echo**.
5. Click **Create**.

nui checks docker/remote ADL configuration on session create. Reachability is validated when the first message is sent. Edit the ADL to point at a different host:

```yaml
harness:
  type: remote
  host: 127.0.0.1
  port: 9090
```

## HTTP/SSE protocol

| Endpoint | Description |
|---|---|
| `GET /info` | Health / reachability check |
| `POST /run` | Body: `{message, sessionId?, workingDir?, systemPrompt?, model?}` → SSE |
| `POST /cancel` | Body: `{runId}` — cancel run best-effort |
| `POST /shutdown` | Stop server and release resources |

SSE events:

```
data: {"type":"text","content":"..."}
data: {"type":"done","sessionId":"..."}
data: {"type":"error","error":"..."}
```

## Lifecycle

| Event | What nui does |
|---|---|
| Session create | `GET /info` reachability check |
| Chat message | `POST /run` → SSE stream |
| Session delete | Nothing |
| Server shutdown | Nothing |

You are responsible for starting, restarting, and securing the remote process (TLS proxy, firewall, process supervisor).

## Writing your own remote agent

```python
from nui_agent import NuiAgent

class MyAgent(NuiAgent):
    name = "my-agent"
    version = "0.1.0"

    def run(self, message: str, **kwargs):
        yield f"Working on: {message}\n"

if __name__ == "__main__":
    MyAgent().serve()
```

Run with `python3 my_agent.py --port <n>` and set the same port in your ADL `harness.port`.
