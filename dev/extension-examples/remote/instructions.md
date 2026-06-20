# Remote Agent Example

A standalone HTTP/SSE server — on the same machine or a remote host. Loop connects via a custom ADL agent; it does not start or stop the process.

## Files

- `echo_agent.py` — agent implementation (subclasses `LoopAgent`)
- `loop_agent.py` — HTTP server framework

## Running

```sh
python3 dev/extension-examples/remote/echo_agent.py --port 9090

curl http://127.0.0.1:9090/info

curl -N -X POST http://127.0.0.1:9090/run \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

## Connecting via Loop UI

Remote agents are configured through **custom ADL**, not a built-in UI picker.

1. Start the agent server (command above).
2. Ensure `~/.loop/agents/remote-echo.yaml` exists (auto-provisioned on first Loop start).
3. Create a new session.
4. Under **Custom Agents**, select **remote-echo**.
5. Click **Create**.

Loop validates reachability via `GET /info` on create. Edit the ADL to point at a different host:

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

| Event | What Loop does |
|---|---|
| Session create | `GET /info` reachability check |
| Chat message | `POST /run` → SSE stream |
| Session delete | Nothing |
| Server shutdown | Nothing |

You are responsible for starting, restarting, and securing the remote process (TLS proxy, firewall, process supervisor).

## Writing your own remote agent

```python
from loop_agent import LoopAgent

class MyAgent(LoopAgent):
    name = "my-agent"
    version = "0.1.0"

    def run(self, message: str, **kwargs):
        yield f"Working on: {message}\n"

if __name__ == "__main__":
    MyAgent().serve()
```

Run with `python3 my_agent.py --port <n>` and set the same port in your ADL `harness.port`.
