# Extension Protocols for Loop

> **Source of truth:** the Go code in `internal/agent/` and the runnable examples in `dev/extension-examples/`.

Loop uses **two extension transports**, depending on how the agent runs:

```mermaid
flowchart LR
  subgraph production [Production — wired to Manager]
    GoHarnesses["Go harness agents\n(claude / pi / codex / opencode)"]
    HTTPExt["HTTPExtensionAgent\n(docker / remote)"]
  end

  subgraph reference [Reference — examples only]
    TCPExt["ExtensionAgent\n(TCP JSON-RPC)"]
    PyTS["dev/extension-examples/py|ts/"]
  end

  Loop[Loop Manager] --> GoHarnesses
  Loop --> HTTPExt
  TCPExt -.->|not wired| Loop
  PyTS -.-> TCPExt
```

## 1. Builtin harnesses (Go subprocess) — primary path

The four built-in CLI agents are **not** Python/TypeScript extensions. Go structs in `internal/agent/` manage CLI subprocesses directly:

| Harness | Implementation |
|---|---|
| `claude-code` | `ClaudeCodeAgent` + `persistentClaudeSession` |
| `pi` | `PiAgent` + `persistentPiSession` (`pi --mode rpc`) |
| `codex` | `CodexAgent` + `persistentCodexSession` |
| `opencode` | `OpenCodeAgent` + `persistentOpenCodeSession` |

Sandbox (`harness.sandbox` in ADL) is propagated via `builtin_config.go` → `Manager.getBuiltinAgent()`.

For `sandbox: docker`, builtin harnesses use HTTP/SSE inside Loop-managed containers (`docker/` images, port **8090**).

## 2. HTTP/SSE — docker and remote connectors

Used for:
- ADL agents with `harness.type: docker` or `remote`
- Builtin `sandbox: docker` (via `HTTPExtensionAgent` talking to `loop-*` images)

### Endpoints

| Endpoint | Description |
|---|---|
| `GET /info` | Health check; returns `{name, version, capabilities}` |
| `POST /run` | Body: `{message, sessionId?, workingDir?, systemPrompt?, model?}` → `text/event-stream` |
| `POST /cancel` | Body: `{runId}` — cancel current run |
| `POST /shutdown` | Release subprocess resources; Loop calls before `docker stop` |

### SSE events

```json
{"type": "text", "content": "..."}
{"type": "done", "sessionId": "..."}
{"type": "error", "error": "..."}
```

Extended types (tool calls, images) are supported by the Go client in `extension.go`.

### Implementations

| Location | Port | Notes |
|---|---|---|
| `docker/http_loop_agent.py` | 8090 | Builtin sandbox images (`loop-claude-code`, etc.) |
| `dev/extension-examples/docker/` | 9090 | User custom agents; ADL `containerPort` |
| `dev/extension-examples/remote/` | user-defined | Standalone server, no lifecycle management |

### Lifecycle

| Event | Docker | Remote |
|---|---|---|
| Session create | `docker run` + `GET /info` health check | `GET /info` reachability check |
| Chat message | `POST /run` → SSE | `POST /run` → SSE |
| Session delete | `POST /shutdown` + `docker stop` | Nothing |
| Server shutdown | Stop all managed containers | Nothing |

See [docker/instructions.md](extension-examples/docker/instructions.md) and [remote/instructions.md](extension-examples/remote/instructions.md).

## 3. TCP JSON-RPC — reference for custom extension authors

`ExtensionAgent` in `internal/agent/extension.go` implements a TCP JSON-RPC 2.0 **client**, but `Manager.GetAgent()` does **not** launch or connect to TCP extensions today. The protocol and frameworks exist for third-party agents and future wiring.

### Connection file

Extensions write `~/.loop/extensions/<name>.json`:

```json
{"host": "127.0.0.1", "port": 52341, "session_id": "...", "pid": 9876}
```

### Methods

| Method | Description |
|---|---|
| `harness.info` | Returns name, version, capabilities |
| `harness.run` | Params: `{message, runId, sessionId?, workingDir?, systemPrompt?, model?}`; streams `harness.event` notifications |
| `harness.cancel` | Params: `{runId}` |
| `harness.shutdown` | Release resources |

### Frameworks

| Language | Framework | Example |
|---|---|---|
| Python | `extensions/loop_agent.py` (canonical) | `dev/extension-examples/py/echo_agent.py` |
| TypeScript | `dev/extension-examples/ts/loop_agent.ts` | `dev/extension-examples/ts/echo_agent.ts` |

Test with `dev/extension-examples/py/client.py` or `ts/client.ts`.

## Historical note

The research below evaluated JSON-RPC over stdio/TCP, go-plugin, ZeroMQ, and MCP as extension transports. The production implementation chose:

- **Go-native subprocess management** for builtin CLI harnesses (simpler, no IPC overhead)
- **HTTP/SSE** for docker/remote (proxy-friendly, standard tooling)
- **TCP JSON-RPC kept as reference** for custom extension authors

---

## Prior Art (research)

| System | Transport | Reconnect mechanism |
|---|---|---|
| **Jupyter** | ZeroMQ (5 sockets) | Connection file with port + session ID |
| **LSP / DAP** | stdio JSON-RPC 2.0 | Process restart (no reconnect needed) |
| **hashicorp/go-plugin** | gRPC over local TCP | `ReattachConfig{Protocol, Addr, Pid}` |

### Why not ZeroMQ?
ZeroMQ requires native bindings (`pyzmq`, `zeromq.js`) — significant install friction. Plain TCP + connection file achieves the same reconnect pattern.

### Why not go-plugin?
go-plugin is Go-centric; cross-language gRPC boilerplate is non-trivial for extension authors.

### The MCP question
MCP uses JSON-RPC 2.0 over stdio or SSE with official SDKs. Loop already surfaces MCP tool UI for Claude tool events via AG-UI. Full MCP-as-extension-protocol remains an open question.

---

## Open Questions

1. **Wire TCP extensions into Manager?** Add a `custom` harness type that launches `extensions/*.py`?
2. **Crash policy:** respawn on connection loss or surface error to user?
3. **MCP as wire protocol?** Reduces custom protocol surface but binds to evolving external spec.

[Extension Examples](extension-examples)
