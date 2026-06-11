# Extension Process Protocol for Loop [AI generated]

[Extension Examples](extension-examples)

## Recommendation: JSON-RPC 2.0 over stdio + local TCP for reconnect

The research strongly converges on a two-phase transport — the same pattern used by LSP, DAP, and Jupyter:

**Phase 1 — Launch:** Loop spawns the extension process and communicates over **stdin/stdout using JSON-RPC 2.0**. This is the LSP/DAP default and is trivially easy to implement in Python and TypeScript.

**Phase 2 — Reconnect:** On startup the extension binds a **local TCP socket** at a random port and writes a **connection file** (JSON, à la Jupyter) to a well-known path (e.g. `~/.loop/extensions/<name>.json`). When Loop restarts, it reads the file and reconnects without restarting the extension process.

---

## Prior Art

| System | Transport | Reconnect mechanism |
|---|---|---|
| **Jupyter** | ZeroMQ (5 sockets) | Connection file with port + session ID |
| **LSP / DAP** | stdio JSON-RPC 2.0 | Process restart (no reconnect needed) |
| **hashicorp/go-plugin** | gRPC over local TCP | `ReattachConfig{Protocol, Addr, Pid}` |

### Why not ZeroMQ (Jupyter)?
ZeroMQ requires native bindings (`pyzmq`, `zeromq.js`) — significant install friction for extension authors. The same connection-file + reconnect pattern works with plain TCP sockets.

### Why not go-plugin (hashicorp)?
go-plugin is the canonical Go library for this and has explicit `ReattachConfig` support. However, extensions written in non-Go languages via gRPC face non-trivial boilerplate. Better to own a simpler JSON-RPC protocol.

---

## The MCP Question

**Should Loop just use MCP as its extension wire protocol?**

MCP (Model Context Protocol) uses JSON-RPC 2.0 over stdio *or* SSE, has official Python and TypeScript SDKs, and Loop's existing `ClaudeCodeAgent` is already Claude-adjacent infrastructure. The MCP Python SDK exposes a FastMCP server in ~10 lines; the TypeScript SDK is equally minimal. This is worth evaluating before designing a custom protocol.

---

## Proposed Design

```
Extension process lifecycle:
  1. Loop spawns: python my_harness.py --connection-file ~/.loop/extensions/my_harness.json
  2. Extension binds TCP, writes: {"host":"127.0.0.1","port":52341,"session_id":"abc123","pid":9876}
  3. Loop connects over TCP, speaks JSON-RPC 2.0
  4. On Loop restart: read connection file → reconnect if pid is alive → else respawn

Extension API (JSON-RPC methods Loop calls):
  harness.run(request)    → streams events back
  harness.cancel(runId)
  harness.info()          → name, version, capabilities

Connection file location: ~/.loop/extensions/<name>.json
Persist pid + port in data.json alongside session IDs for restart recovery
```

### Python extension (~20 lines)

```python
from jsonrpcserver import method, serve
@method
def run(request): ...
serve()
```

### TypeScript extension

```typescript
import { createConnection } from 'vscode-jsonrpc/node'  // standalone, no LSP needed
```

---

## Open Questions

1. **Adopt MCP instead?** MCP gives you the wire protocol, Python/TS SDKs, and tool/resource schema for free. Downside: Loop's extension contract binds to an evolving external spec.
2. **Crash policy:** go-plugin does not auto-restart crashed extensions. Should Loop respawn on connection loss, or surface the error to the user?
3. **gRPC vs JSON-RPC:** gRPC (go-plugin style) gives typed schemas via Protobuf. JSON-RPC is simpler to author. For Loop's expected use case (small single-developer harnesses), JSON-RPC wins.
4. **Connection file persistence:** `~/.loop/data.json` (already used) or a dedicated per-extension sidecar file?

---

## Research Basis

- 104 agents, 22 sources fetched, 25 claims adversarially verified (21 confirmed, 4 killed)
- Key sources: jupyter-client docs, vscode-languageserver-node/jsonrpc README, hashicorp/go-plugin README + pkg.go.dev, MCP specification + Python/TypeScript SDKs
- Refuted claims: go-plugin uses Unix sockets (uses local TCP), LSP default transport is IPC pipes (it's stdio), go-plugin cross-language gRPC is low-friction for extension authors (boilerplate is non-trivial)
