# The Loop

Loop is a self-hosted UI for interactive AI agent sessions. A Go backend embeds a React frontend and runs agent harnesses locally, in Docker, or over the network.

<img src="media/loop-logo.png" alt="Loop Logo" width="400">

## Documentation

- [Product & technical spec](dev/dev.md) — architecture, ADL schema, roadmap
- [Extension protocols](dev/extension-design.md) — HTTP/SSE and TCP JSON-RPC for custom agents
- [ADL examples](dev/adl/examples/) — sample agent/workflow YAML files

## Architecture

```mermaid
flowchart TB
  subgraph ui [Browser]
    Chat[ChatPanel / useSessionChat]
  end

  subgraph server [Go server :8080]
    API[REST API]
    AGUI["/api/sessions/:id/ag-ui"]
    ADL[ADLAgent]
    Mgr[Manager]
  end

  subgraph builtins [Builtin harnesses — Go subprocess]
    CC[ClaudeCodeAgent]
    PI[PiAgent]
    CX[CodexAgent]
    OC[OpenCodeAgent]
  end

  subgraph connectors [Connectors — HTTP/SSE]
    HTTP[HTTPExtensionAgent]
    Docker[(Docker container)]
    Remote[(Remote server)]
  end

  Chat -->|AG-UI SSE| AGUI
  AGUI --> ADL
  ADL --> Mgr
  Mgr --> CC & PI & CX & OC
  Mgr --> HTTP
  HTTP --> Docker & Remote
```

**Session flow:** every session has an `agentType` that resolves to an ADL definition (built-in or `~/.loop/agents/*.yaml`). `ADLAgent` runs the harness — single-step for simple agents, multi-step DAG for workflows. The UI streams chat over the [AG-UI protocol](https://github.com/ag-ui-protocol/ag-ui) at `POST /api/sessions/:id/ag-ui`, not the legacy `/chat` endpoint.

## Prerequisites

- Go 1.22+
- Node.js 18+
- Agent CLIs on `PATH` as needed: `claude`, `pi`, `codex`, `opencode`
- Docker (optional) — for `sandbox: docker` and custom docker-harness ADL agents

## Project structure

```
loop/
├── main.go, embed.go          # entrypoint; embeds ui/dist
├── cmd/                       # cobra CLI (`loop ui`)
├── internal/
│   ├── model/                 # Session, ChatMessage, ADL structs
│   ├── agent/                 # Agent interface, harness implementations, ADL executor
│   ├── server/                # HTTP mux, REST + AG-UI streaming
│   └── store/                 # JSON persistence (~/.loop/)
├── docker/                    # Builtin sandbox images (HTTP/SSE, port 8090)
├── extensions/                # Reference TCP JSON-RPC framework (not wired to Manager)
├── dev/
│   ├── dev.md                 # product spec
│   ├── extension-design.md    # extension protocols
│   ├── extension-examples/    # runnable docker/remote/TCP examples
│   └── adl/examples/          # sample ADL YAML
└── ui/                        # Vite + React frontend
```

## Running in development

Terminal 1 — Vite dev server (proxies `/api` to Go):

```sh
cd ui && npm install && npm run dev
```

Terminal 2 — Go server (`ui/dist` must exist before `go build`):

```sh
cd ui && npm run build && cd ..
go run . ui              # default :8080
go run . ui --port 3000  # custom port
```

Production build:

```sh
cd ui && npm run build && cd .. && go build -o loop_bin . && ./loop_bin ui
```

## CLI

```
loop ui              # start web server on :8080
loop ui --port 3000  # custom port
loop ui -p 3000      # shorthand
loop ui --open       # open http://localhost:8080 with a new blank session
```

### Launch an agent from the CLI

Create a session on startup and optionally run an initial prompt in the UI:

```sh
loop ui --agent-type "Claude Code" --prompt "Review the README" --open
loop ui -a pi -m "Summarize this repo" -w ~/my/project --open
```

| Flag | Short | Description |
|---|---|---|
| `--open` | | Open the web UI in the system default browser. Creates a new blank session with the default agent and selects it (instead of resuming the last session). |
| `--agent-type` | `-a` | ADL agent name (builtin or `~/.loop/agents/*.yaml`). Creates a new session on startup and starts with the sidebar closed. |
| `--prompt` | `-m` | Initial prompt. The UI selects the new session, fills the input, and auto-sends it. Also starts with the sidebar closed. |
| `--working-dir` | `-w` | Working directory for the session (defaults to the current directory). |

Agent type names match the New Session dialog — e.g. `Claude Code`, `pi`, `codex`, `opencode`, or a custom ADL name like `docker-echo`. Legacy aliases such as `claude-code` also work.

On startup Loop:

1. Creates the session and validates docker/remote connectors if needed
2. Saves `lastAgentType` and `lastSessionId` to `~/.loop/settings.json`
3. Exposes the prompt once via `GET /api/bootstrap` for the UI to consume

If no sessions exist when the server starts (and `--agent-type` was not passed), Loop automatically creates one using `lastAgentType` from settings, or the first available built-in agent (`Claude Code`, `pi`, `codex`, or `opencode`).

### User preferences

Stored in `~/.loop/settings.json` and restored on reload:

| Field | Saved when | Restored as |
|---|---|---|
| `theme` | Settings sheet toggle | Light/dark theme |
| `lastAgentType` | Session create (UI or CLI) | Default in New Session dialog |
| `lastSessionId` | Session selection | Auto-select on startup (if session still exists) |
| `sidebarOpen` | Sidebar toggle | Sidebar expanded/collapsed state (offcanvas — no gutter when closed) |

`PUT /api/settings` accepts partial updates — send only the fields you want to change.

## API endpoints

| Path | Description |
|---|---|
| `/` | React SPA |
| `/assets/*` | Static assets (embedded from `ui/dist`) |
| `/health` | JSON health check |
| `GET/POST /api/sessions` | List / create sessions |
| `POST /api/sessions/ensure-default` | Return or create the default session |
| `GET/PATCH/DELETE /api/sessions/:id` | Get / rename / delete a session |
| `GET/PUT /api/sessions/:id/messages` | Read / replace persisted UI messages |
| `POST /api/sessions/:id/ag-ui` | **Primary chat** — AG-UI SSE stream |
| `POST /api/sessions/:id/chat` | Legacy raw agent-event SSE (unused by UI) |
| `GET /api/sessions/:id/history` | Load history from agent session files |
| `GET /api/agent-types` | Builtin + user-defined ADL agent types |
| `GET /api/directories` | Working-directory autocomplete |
| `GET/PUT /api/settings` | User preferences (theme, last agent/session, sidebar) |
| `GET /api/bootstrap` | One-shot CLI bootstrap state (`sessionId`, `initialPrompt`) |
| `GET /api/capabilities` | Sandbox capabilities (bwrap availability) |

## Agent types

### Built-in (UI → Standard)

Four CLI harnesses, selectable as pills in the New Session dialog:

| Name | Harness | Runs |
|---|---|---|
| Claude Code | `claude-code` | `claude` CLI subprocess |
| pi | `pi` | `pi --mode rpc` subprocess |
| codex | `codex` | `codex exec` subprocess |
| opencode | `opencode` | `opencode serve` + `opencode run` |

### Custom (UI → Custom Agents)

ADL YAML files in `~/.loop/agents/*.yaml`. On first run, Loop provisions starter templates:

- `docker-echo.yaml` — HTTP/SSE agent in Docker (`loop-echo-agent:9090`)
- `remote-echo.yaml` — HTTP/SSE agent at `127.0.0.1:9090`
- `opencode-docker.yaml` — opencode in `loop-opencode:latest`

Use these for `docker` and `remote` harness types, sandbox variants (`bubblewrap`, `docker`), and multi-step workflows.

### Sandbox options

For `claude-code`, `pi`, `codex`, and `opencode` (set in ADL `harness.sandbox`):

| Value | Behavior |
|---|---|
| `none` | Run on host (default) |
| `bubblewrap` | Wrap subprocess with `bwrap` (Linux only) |
| `docker` | Run in a Loop-managed container (`loop-<harness>:latest`, port **8090**) |

### Docker / remote connectors

Custom ADL agents with `harness.type: docker` or `remote` use the HTTP/SSE protocol. Loop validates connector configuration on session create; containers and remote connections start on the first message. See [extension examples](dev/extension-examples/).

**Port note:** builtin sandbox images in `docker/` listen on **8090**; user extension examples use **9090** (configured via ADL `containerPort`).
