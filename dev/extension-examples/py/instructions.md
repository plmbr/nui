# TCP JSON-RPC Extension Examples (Reference)

These examples demonstrate the TCP JSON-RPC 2.0 protocol for custom Loop extensions. **They are not wired to `Manager.GetAgent()` today** — Loop's production path uses Go-managed CLI subprocesses and HTTP/SSE connectors instead. Use these to build standalone agents or as a reference for a future `custom` harness type.

## Files

- `loop_agent.py` — extension framework (canonical copy of `extensions/loop_agent.py`)
- `echo_agent.py` — minimal echo harness
- `client.py` — sample client that connects to a running extension

## Running

```sh
# terminal 1 — start the extension
python dev/extension-examples/py/echo_agent.py

# terminal 2 — run the client
python dev/extension-examples/py/client.py
```

## Protocol

On startup, the extension binds a random TCP port and writes `~/.loop/extensions/<name>.json`:

```json
{"host": "127.0.0.1", "port": 52341, "session_id": "...", "pid": 9876}
```

### Methods

| Method | Description |
|---|---|
| `harness.info` | Returns name, version, capabilities |
| `harness.run` | Streams `harness.event` notifications, then returns result |
| `harness.cancel` | Acknowledges cancellation |
| `harness.shutdown` | Releases resources |

### Streaming

`harness.run` sends interleaved JSON-RPC notifications (`harness.event`) and a final response. The `client.py` `Client` class handles this by looping past notifications until the matching response ID arrives.

## Manual testing

```sh
echo '{"jsonrpc":"2.0","id":1,"method":"harness.info","params":{}}' | nc 127.0.0.1 <port>
```

## Dependencies

None — stdlib only.

See also: [extension-design.md](../../extension-design.md), TypeScript examples in `../ts/`.
