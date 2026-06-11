# Loop Extension Examples

## Files

- `echo_agent.py` — minimal extension harness (server)
- `client.py` — sample client that connects to a running extension

## Running

```sh
# terminal 1 — start the extension
python dev/extension-examples/echo_agent.py

# terminal 2 — run the client
python dev/extension-examples/client.py

# or specify a different extension name
python dev/extension-examples/client.py my-agent
```

## How it works

### echo_agent.py

1. Binds a random TCP port, writes `~/.loop/extensions/echo-agent.json` with `{host, port, session_id, pid}`
2. Accepts multiple connections (one thread per client)
3. Reads newline-delimited JSON-RPC 2.0 requests and handles three methods:
   - `harness.info` — returns name, version, capabilities
   - `harness.run` — sends a `harness.event` notification (streaming text), then a `done` event, then the final response
   - `harness.cancel` — acknowledges immediately
4. On shutdown (Ctrl+C), removes the connection file

### client.py

1. Reads `~/.loop/extensions/echo-agent.json` (errors if the extension is not running)
2. Connects over TCP
3. Calls `harness.info` and prints the result
4. Calls `harness.run` with a message — `harness.event` notifications are printed as they arrive (streaming), then prints the final result

The `Client` class handles the one tricky part of JSON-RPC streaming: notifications (messages without an `id`) arrive interleaved with the response, so `call()` loops past them and dispatches via `_on_notification` until the matching response ID arrives.

## Manual testing

```sh
# start the agent, note the port in stderr output, then:
echo '{"jsonrpc":"2.0","id":1,"method":"harness.info","params":{}}' | nc 127.0.0.1 <port>
```

## Dependencies

None — both files use Python stdlib only.
