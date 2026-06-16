# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go)
```sh
go build ./...          # compile
go run . ui             # build + run server on :8080
go run . ui --port 3000 # custom port (use this for development)
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
cd ui && npm run build && cd .. && go build -o loop_bin . && ./loop_bin ui
```

> `ui/dist` must exist before `go build` — it is embedded into the binary at compile time via `//go:embed ui/dist`.

### Docker images (run from `docker/`)
```sh
# Build from the docker/ directory (all images share http_loop_agent.py)
docker build -f claude-code/Dockerfile -t loop-claude-code:latest .
docker build -f pi/Dockerfile          -t loop-pi:latest           .
docker build -f codex/Dockerfile       -t loop-codex:latest        .
```

All docker sandbox images forward `ANTHROPIC_API_KEY` and `ANTHROPIC_BASE_URL` automatically from the host environment. If `ANTHROPIC_BASE_URL` resolves to loopback, Loop adds `--add-host=<hostname>:host-gateway` automatically.

## Architecture

### Request flow
```
Browser → Go HTTP server (:8080)
         ├── /assets/*         — embedded static files from ui/dist
         ├── /api/*            — JSON REST + SSE (internal/server/api.go)
         ├── /health           — health check
         └── /                 — serves ui/dist/index.html (SPA fallback)
```

In development, Vite's dev server (`:5173`) proxies `/api` to the Go server so HMR works without rebuilding.

### Go packages

| Package | Role |
|---|---|
| `cmd/` | Cobra CLI (`loop ui [--port]`); wires embedded `ui/dist` FS into `server.Start()` |
| `internal/server/` | HTTP mux, API handlers, SSE streaming, in-memory state with `sync.RWMutex` |
| `internal/model/` | Shared `Project` and `ChatMessage` structs (avoids import cycles) |
| `internal/store/` | JSON persistence to `~/.loop/data.json` (projects + session IDs) and `~/.loop/settings.json` (theme). Atomic writes: `os.CreateTemp` → write → `os.Rename`. Also reads/deletes Claude Code session files. |
| `internal/agent/` | `Agent` interface, `ClaudeCodeAgent`, `ExtensionAgent` (TCP JSON-RPC), `HTTPExtensionAgent` (HTTP/SSE), `Manager` (process + container lifecycle), and `sandbox.go` (bubblewrap detection + wrapping) |

### Agent interface
```go
type Agent interface {
    Name() string
    Run(ctx context.Context, req RunRequest, events chan<- Event) error
}
```
`ClaudeCodeAgent.Run` launches `claude -p <msg> --output-format stream-json --verbose --dangerously-skip-permissions --include-partial-messages --model <model> [--resume <sessionID>] [--system-prompt <prompt>]` and streams parsed SSE events (`EventText`, `EventDone`, `EventError`) back over a channel. The channel is consumed by `handleProjectChat`, which forwards events to the browser as `text/event-stream`. On Linux, when `bwrap` is available, the `claude` process runs inside a bubblewrap sandbox (read-only rootfs; `workDir` and `~/.claude` bind-mounted read-write; network preserved).

`ExtensionAgent` speaks JSON-RPC 2.0 over TCP to a managed Python/TS extension process (`pi` and `claude-code` harness types). `HTTPExtensionAgent` talks to Docker and remote agents via HTTP/SSE (`POST /run` → `text/event-stream`; `GET /info` for health checks).

`Manager` handles the full lifecycle: launching Python extension processes (writing/reading `~/.loop/extensions/<projectID>.json` connection files), starting Docker containers (`docker run -d -p 127.0.0.1::<port>`), resolving mapped ports via `docker port`, waiting for HTTP readiness, and stopping everything on delete/shutdown. Docker container URLs are cached in-process; remote agents are stateless (Loop just stores the configured host:port). Python extension processes receive `LOOP_BWRAP_PATH` when bwrap is available so they can sandbox their subprocesses.

#### Agent types and step harness types

There are five built-in agent types selectable at session creation, mapping directly to the five ADL step harness types:

| Agent type | Harness type | How it runs |
|---|---|---|
| Claude Code | `claude-code` | Runs `claude` CLI as a local subprocess |
| pi | `pi` | Runs `pi` CLI as a local subprocess |
| codex | `codex` | Runs `codex` CLI as a local subprocess |
| docker | `docker` | Connects to an HTTP/SSE agent in a Docker container (`image`, `containerPort` required) |
| remote | `remote` | Connects to a pre-running HTTP/SSE agent over the network (`host`, `port` required) |

`claude-code`, `pi`, and `codex` harnesses support a `sandbox` field:
- `none` — run directly on the host (default)
- `bubblewrap` — bubblewrap sandbox (Linux only)
- `docker` — runs inside a Docker container

Sandbox variants and custom configurations are expressed via user-defined ADL in `~/.loop/agents/*.yaml` rather than as separate named built-in types. The `docker/` directory contains `http_loop_agent.py` (shared HTTP/SSE base class), `claude-code/claude_code_http.py` + `Dockerfile`, `pi/pi_http.py` + `Dockerfile`, and `codex/codex_http.py` + `Dockerfile`.

#### Sandbox (`sandbox.go`)

`GetBwrapStatus()` detects bubblewrap once (singleton) and returns a cached `BwrapStatus`. `WrapWithBwrap(bwrapPath, bin, args, workDir)` prepends the bwrap invocation with appropriate bind-mounts. Linux-only; macOS falls back to running the subprocess unsandboxed.

### Persistence
- `~/.loop/data.json` — projects array + `sessions` map (project ID → Claude session ID). Loaded on startup via `initStore()`, saved after create/delete/rename and after a new session ID arrives.
- `~/.loop/settings.json` — `{"theme": "light"|"dark"}`. Read/written on every settings GET/PUT.
- Claude Code session files live at `~/.claude/projects/<dirHash>/<sessionID>.jsonl` where `dirHash = strings.ReplaceAll(workingDir, "/", "-")`. History is loaded by `store.LoadClaudeHistory` and deleted by `store.DeleteClaudeSession`.
- When `workingDir` is empty, both the agent runner and the history loader fall back to `os.Getwd()` (the server's working directory).

### API routes
All routes are registered in `internal/server/api.go`:
- `GET/POST /api/projects` — list / create
- `GET/PATCH/DELETE /api/projects/:id` — get / rename / delete (delete also removes the Claude session file)
- `POST /api/projects/:id/chat` — SSE stream; runs the agent, saves new session ID
- `GET /api/projects/:id/history` — load chat history from Claude's JSONL file
- `GET /api/agent-types` — static list of available agent types
- `GET/PUT /api/settings` — theme setting
- `GET /api/capabilities` — sandbox capabilities (bwrap availability; used by Settings UI to show a warning on unsupported platforms)

### Frontend structure
- `App.tsx` — root; owns `projects` and `selectedId` state, passes handlers down
- `api.ts` — all fetch calls in one place; `request<T>()` handles errors and 204 No Content
- `types.ts` — shared TypeScript interfaces mirroring Go model structs
- `contexts/theme.tsx` — `ThemeProvider`; `localStorage` as first-paint cache, `/api/settings` as source of truth
- Components: `AppSidebar` (project list + new project + settings trigger), `ProjectDetails` (rename inline, delete with confirmation dialog), `ChatPanel` (SSE streaming, markdown rendering, history load on project select)

### UI stack
Tailwind CSS v4 (CSS-based config in `index.css`), shadcn/ui components built on Base UI (`@base-ui/react`), `react-markdown` + `rehype-highlight` for assistant message rendering, `@tailwindcss/typography` (`prose prose-sm dark:prose-invert`) on assistant bubbles.
