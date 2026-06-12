# Remote Agent Example

A Loop extension that runs as a standalone process — on the same machine or
a remote host. Loop connects to it using a stored host:port; there is no
process lifecycle management (Loop doesn't start or stop it).

## Files

- `echo_agent.py` — the agent implementation (subclasses `LoopAgent`)
- `loop_agent.py` — server-mode framework (supports `--port` and `0.0.0.0` binding)

## Running

```sh
# Start the agent on port 9090
python3 dev/extension-examples/remote/echo_agent.py --port 9090

# Smoke-test it manually (optional)
echo '{"jsonrpc":"2.0","id":1,"method":"harness.info","params":{}}' \
  | nc 127.0.0.1 9090
```

## Connecting via Loop UI

1. Create a new project.
2. Choose **Agent Type → Remote**.
3. Set **Host** to `127.0.0.1` (or the IP/hostname of the remote machine).
4. Set **Port** to `9090`.
5. Click **Create**.

Loop verifies TCP reachability on create, then connects for each chat message.

## How it works

### Why `--port` and `0.0.0.0`?

Standard Loop extensions bind to `127.0.0.1` on a random port and write a
connection file. Remote agents skip both steps:

- `--port` makes the listening port predictable so you can configure it in Loop.
- Binding to `0.0.0.0` makes the agent reachable from other hosts (not just
  loopback). When running locally for testing, this is equivalent to
  `127.0.0.1` from the client's perspective.
- No connection file is written; Loop uses the host:port stored in the project
  config instead.

### Lifecycle

Loop owns no process or container for a remote agent. You are responsible for
starting, restarting, and stopping the process. Loop will surface an error when
the agent is unreachable.

| Event | What Loop does |
|---|---|
| Project created | TCP connect check (fails fast if unreachable) |
| Chat message sent | TCP connect → `harness.run` RPC → stream events |
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

For production use you would want to run the agent under a process supervisor
(systemd, supervisord, etc.) and secure the port (firewall, TLS proxy, SSH
tunnel) as needed.

## Writing your own remote agent

```python
from loop_agent import LoopAgent

class MyAgent(LoopAgent):
    name = "my-agent"
    version = "0.1.0"

    def run(self, message: str, run_id: str, **kwargs):
        # workingDir and sessionId are passed as kwargs when present
        yield "Working...\n"
        yield call_my_backend(message)

if __name__ == "__main__":
    MyAgent().serve()
```

Run with `python3 my_agent.py --port <n>` and configure the same port in Loop.
