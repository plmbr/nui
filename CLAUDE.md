# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go)
```sh
go build ./...          # compile
go test . ./cmd/... ./internal/...  # run tests (avoids ui/node_modules)
./scripts/test-all.sh               # Go + Vitest + Playwright E2E
go run . server             # build + run server on :8080
go run . server --port 3000 # custom port (use this for development)
go run . server -a claude-code -m "Review README" -w . --open --hide-input
```

### Frontend (run from `ui/`)
```sh
npm install
npm run dev     # Vite dev server with HMR (proxies /api to Go server)
npm run build   # compile to ui/dist (required before go build)
npm run lint    # ESLint
```

### Full production build
```sh
cd ui && npm run build && cd .. && go build -o nui_bin . && ./nui_bin server
```

> `ui/dist` must exist before `go build` — it is embedded into the binary at compile time via `ui/embed.go` (`//go:embed all:dist`).

### Docker images (run from `docker/`)
```sh
docker build -f claude-code/Dockerfile -t nui-claude-code:latest .
docker build -f pi/Dockerfile          -t nui-pi:latest           .
docker build -f codex/Dockerfile       -t nui-codex:latest        .
docker build -f opencode/Dockerfile    -t nui-opencode:latest     .
```

All docker sandbox images listen on port **8090** and share `http_nui_agent.py`. They forward `ANTHROPIC_API_KEY`, `ANTHROPIC_BASE_URL`, `OPENAI_API_KEY`, and `OPENAI_BASE_URL` from the host. nui adds `--add-host=<hostname>:host-gateway` when a base URL resolves to loopback.

## Architecture

### Request flow

```mermaid
flowchart LR
  Browser -->|REST| API["/api/*"]
  Browser -->|AG-UI SSE| AGUI["/api/sessions/:id/ag-ui"]
  API --> Store["~/.nui/data.json"]
  AGUI --> ADLAgent
  ADLAgent --> Manager
  Manager --> Harnesses["ClaudeCode / Pi / Codex / OpenCode"]
  Manager --> HTTPExt["HTTPExtensionAgent"]
  HTTPExt --> DockerRemote["Docker / Remote"]
```

In development, Vite (`:5173`) proxies `/api` to the Go server.

### Go packages

| Package | Role |
|---|---|
| `cmd/` | Cobra CLI (`nui server`, `nui run`, `nui agent`, `nui extension`, `nui skills`, `nui memory`, `nui schedule`, MCP stdio servers) |
| `internal/server/` | HTTP mux, REST handlers, AG-UI streaming (`agui.go`), orchestrate/home launcher, MCP OAuth, MCP tool UI (`mcp_manager.go`) |
| `internal/model/` | `Session`, `ChatMessage`, ADL structs |
| `internal/store/` | Persistence: `data.json`, `settings.json`, ADL YAML in `agents/`, user plugins in `~/.nui/extensions/`, agent history loaders |
| `internal/extensions/` | Extension registry: manifest scan, list sources (file/catalog RPC), harness/MCP/skill/agent contributions |
| `internal/agents/` | Built-in ADL defs (CLI, API, `nui` master agent) |
| `internal/agent/` | `Agent` interface, harness agents, `ADLAgent` executor, `Manager` lifecycle, `sandbox.go` (bwrap) |

### Agent interface

```go
type Agent interface {
    Name() string
    Run(ctx context.Context, req RunRequest, events chan<- Event) error
}
```

Every session resolves to an ADL definition via `findADLDef()`. `ADLAgent` dispatches to the correct harness per step (or top-level harness for single-step agents).

#### Builtin harnesses (Go-managed subprocesses)

| Harness | Agent struct | CLI |
|---|---|---|
| `claude-code` | `ClaudeCodeAgent` | `claude -p … --output-format stream-json` (persistent session via `claude_session.go`) |
| `pi` | `PiAgent` | `pi --mode rpc` (JSON-RPC over stdin/stdout via `pi_session.go`) |
| `codex` | `CodexAgent` | `codex exec … --json` (JSONL via `codex_session.go`) |
| `opencode` | `OpenCodeAgent` | `opencode serve` + `opencode run --attach` (via `opencode_session.go`) |

Sandbox is set from ADL `harness.sandbox` and propagated via `harnessBuiltinConfig()` → `Manager.getBuiltinAgent()`. Bubblewrap only applies when `sandbox: bubblewrap` (Linux only); `none` runs unsandboxed on the host.

Bind mounts per harness when bwrap is active:
- `claude-code` → `~/.claude`
- `pi` → `~/.pi`
- `codex` → `~/.codex`
- `opencode` → `~/.local/share/opencode`

#### Connector harnesses (HTTP/SSE)

| Harness | Agent struct | Lifecycle |
|---|---|---|
| `docker` | `HTTPExtensionAgent` | nui runs `docker run -d -p 127.0.0.1::<port> <image>`, maps port, health-checks `GET /info` |
| `remote` | `HTTPExtensionAgent` | nui stores `host:port`; no process management |

`Manager` also provides `GetClaudeCodeDocker`, `GetPiDocker`, `GetCodexDocker`, `GetOpenCodeDocker` for `sandbox: docker` on builtin harnesses (images `nui-*:latest`, port 8090).

On delete/shutdown, nui calls `POST /shutdown` on managed containers, then `docker stop`.

#### Extension harnesses (stdio / TCP / HTTP)

Installed extensions contribute harnesses via `contributions.harnesses`. `Manager.getExtensionHarnessAgent()` wires them:

| Transport | Go client |
|---|---|
| `stdio` (default) | `stdioHarnessAgent` |
| `tcp` | `ExtensionAgent` (JSON-RPC 2.0) |
| `http` | `HTTPExtensionAgent` |

ADL agents use `harness.type: ext:<extension>/<harness-id>`. Framework: `harness-sdk/nui_agent_stdio.py`. See `dev/extension-api.md`.

#### Standalone reference examples (not registered)

- `dev/harness-examples/py|ts/` — TCP JSON-RPC demos without an `extension.yaml`; not selectable as agent types
- `harness-sdk/nui_agent.py` — TCP framework used by those examples

### HTTP/SSE protocol (docker + remote)

| Endpoint | Description |
|---|---|
| `GET /info` | Health check + metadata |
| `POST /run` | Body: `{message, sessionId?, workingDir?, systemPrompt?, model?}` → `text/event-stream` |
| `POST /cancel` | Body: `{runId}` — cancel current run |
| `POST /shutdown` | Stop subprocesses; used by nui on container teardown |

SSE `data:` events support `text`, `done`, `error`, and tool-call/image event types (see `eventFromHarnessParams` in `extension.go`).

### Persistence

| File | Contents |
|---|---|
| `~/.nui/data.json` | `sessions`, `agentSessions` (nui session ID → agent session ID), `sessionMessages` (UI chat text) |
| `~/.nui/settings.json` | `theme`, `defaultAgentType`, `defaultHarness`, `lastAgentType`, `lastSessionId`, `sidebarOpen`, `disabledExtensions` |
| `~/.nui/sessions/<session-id>/` | Per-session harness config (MCP, skills, system prompt, `.devcontainer/`); removed on session delete |
| `~/.nui/workspaces/<session-id>/` | Isolated working dir when ADL does not request user `workingDirInput`; removed on session delete |
| `$TMPDIR/nui-uploads/<session-id>/` | Pasted/dropped chat attachments; removed on session delete |
| `~/.nui/runs/<runID>.jsonl` | Durable run event logs (AG-UI + headless); removed when the owning session is deleted |
| `~/.nui/hitl-requests.json` | HITL request/response envelopes; session entries removed on session delete |
| `~/.nui/schedules.json` | Interval schedules (`lastSessionId` is a pointer only; not cleaned with sessions) |
| `~/.nui/memory/` | Persistent user/agent memory markdown (not session-scoped) |
| `~/.nui/agents/*.yaml` | User ADL definitions; loaded on every `GET /api/agent-types` |
| `~/.nui/extensions/<name>/` | Backend extensions (`extension.yaml` + contribution list files); see `dev/extension-api.md` |
| `~/.nui/connections/*.json` | Harness TCP/HTTP handshake files (`host`, `port`, `session_id`, `pid`) — harness-scoped, not per-session |
| `~/.nui/opencode-sessions/` | Shared OpenCode Docker data mount (`~/.local/share/opencode` in container) |
| Agent history files | Claude: `~/.claude/projects/<dirHash>/<id>.jsonl`; pi/codex/opencode via respective `store/*_history.go` loaders; deleted with the nui session when an agent session id is known |

UI loads persisted `sessionMessages` first on session select; falls back to agent history files if empty.

### API routes

Registered in `internal/server/api.go` and `agui.go`:

- `GET/POST /api/sessions` — list / create (docker/remote ADL config validated on create; agents start on first message)
- `POST /api/sessions/ensure-default` — return last session or create one with the default agent
- `GET/PATCH/DELETE /api/sessions/:id` — get / rename / delete
- `GET/PUT /api/sessions/:id/messages` — persisted UI messages
- `POST /api/sessions/:id/ag-ui` — **primary chat endpoint** (AG-UI protocol)
- `POST /api/sessions/:id/chat` — legacy raw agent events (unused by UI)
- `GET /api/sessions/:id/history` — agent-side session history
- `GET /api/agent-types` — builtin + user ADL types
- `GET /api/directories` — working-dir autocomplete
- `GET/PUT /api/settings` — user preferences (partial PUT supported)
- `GET /api/bootstrap` — one-shot CLI bootstrap (`sessionId`, `initialPrompt`); consumed on first read
- `GET /api/capabilities` — bwrap availability
- `GET /api/extensions` — installed extensions and contribution ids
- `POST /api/extensions/reload` — rescan `~/.nui/extensions/`

### Frontend structure

- `App.tsx` — session list + selection; restores `lastSessionId`, `sidebarOpen`, and CLI bootstrap prompt
- `useSessionChat.ts` — AG-UI client (`@ag-ui/client`); handles text, tool calls, images, MCP app frames
- `ChatPanel.tsx` / `ConversationPanel.tsx` — chat UI
- `NewSessionPanel.tsx` — builtin harness picker + custom ADL agent cards
- `api.ts` — REST client
- `types.ts` — TypeScript mirrors of Go model structs

### ADL executor status

Implemented in `internal/agent/adl.go`:
- Topological step scheduling (`dependsOn`)
- Per-step harness/model/systemPrompt override
- Named outputs → downstream inputs
- All six harness types + sandbox variants
- `aiAssets.mcpServers`, `aiAssets.skills`, legacy `skill`, `systemPrompt`, `env`/`harness.env`, and `promptMode` provisioned to harness subprocesses / UI via `~/.nui/sessions/<session-id>/`

### UI stack

Tailwind CSS v4, shadcn/ui on Base UI (`@base-ui/react`), `react-markdown` + `rehype-highlight`, `@ag-ui/client` for streaming.
