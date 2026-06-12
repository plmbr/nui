# Remote Agent Example

A Loop extension that runs as a standalone HTTP/SSE server — on the same
machine or a remote host. Loop connects to it using a stored host:port; there
is no process lifecycle management (Loop doesn't start or stop it).

## Files

- `echo_agent.py` — the agent implementation (subclasses `LoopAgent`)
- `loop_agent.py` — HTTP server framework (`GET /info`, `POST /run`, `POST /cancel`)

## Running

```sh
# Start the agent on port 9090
python3 dev/extension-examples/remote/echo_agent.py --port 9090

# Smoke-test it manually
curl http://127.0.0.1:9090/info

curl -N -X POST http://127.0.0.1:9090/run \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

## Connecting via Loop UI

1. Create a new project.
2. Choose **Agent Type → Remote**.
3. Set **Host** to `127.0.0.1` (or the IP/hostname of the remote machine).
4. Set **Port** to `9090`.
5. Click **Create**.

Loop checks reachability via `GET /info` on create, then calls `POST /run` for
each chat message.

## How it works

### HTTP/SSE protocol

| Endpoint | Description |
|---|---|
| `GET /info` | Returns `{"name", "version", "capabilities"}` — also used as a health/reachability check |
| `POST /run` | Body: `{"message", "sessionId"?, "workingDir"?}`; response: `text/event-stream` |
| `POST /cancel` | Body: `{"runId"}`; stops the current run best-effort |

SSE events are JSON-encoded `data:` lines:

```
data: {"type":"text","content":"..."}
data: {"type":"done","sessionId":"..."}
data: {"type":"error","error":"..."}
```

### Lifecycle

Loop owns no process or container for a remote agent. You are responsible for
starting, restarting, and stopping the process.

| Event | What Loop does |
|---|---|
| Project created | `GET /info` reachability check |
| Chat message sent | `POST /run` → SSE stream |
| Project deleted | Nothing (process keeps running) |
| Server shutdown | Nothing |

### Running on a remote machine

```sh
# On the remote host
python3 echo_agent.py --port 9090

# In Loop UI
#   Host: <remote-ip>
#   Port: 9090
```

For production use, run the agent under a process supervisor (systemd,
supervisord, etc.) and secure the endpoint (TLS terminating proxy, SSH tunnel,
firewall rules) as needed. Because the protocol is plain HTTP, any standard
reverse proxy (nginx, Caddy, Traefik) can add TLS without changes to the agent.

## Writing your own remote agent

```python
from loop_agent import LoopAgent

class MyAgent(LoopAgent):
    name = "my-agent"
    version = "0.1.0"

    def run(self, message: str, **kwargs):
        # workingDir and sessionId are passed as kwargs when present
        yield "Working...\n"
        yield call_my_backend(message)

if __name__ == "__main__":
    MyAgent().serve()
```

Run with `python3 my_agent.py --port <n>` and configure the same port in Loop.
