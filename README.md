# The Loop

Loop is a UI for interactive and autonomous agents.

<img src="media/loop-logo.png" alt="Loop Logo" width="400">

## Development

[Development Plan](dev/dev.md)

[Extension Design](dev/extension-design.md)


### Prerequisites

- Go 1.22+
- Node.js 18+

### Project structure

```
loop/
├── main.go                   # entrypoint
├── embed.go                  # embeds ui/dist into the binary
├── cmd/                      # cobra CLI commands
├── internal/
│   ├── model/                # shared Project and ChatMessage structs
│   ├── agent/                # Agent interface, ClaudeCodeAgent, ExtensionAgent,
│   │                         # HTTPExtensionAgent, Manager (process/container lifecycle)
│   ├── server/               # HTTP mux, API handlers, SSE streaming
│   └── store/                # JSON persistence (~/.loop/data.json, settings.json)
└── ui/                       # Vite + React frontend
    └── dist/                 # built output (generated, not committed)
```

### Running in development

Start the React dev server with HMR:

```sh
cd ui
npm install
npm run dev
```

In a separate terminal, build and run the Go server:

```sh
cd ui && npm run build  # ui/dist must exist before go build
cd ..
go run .  ui            # or: go build -o loop_bin . && ./loop_bin ui
```

The Go binary embeds `ui/dist` at compile time, so rebuild after UI changes:

```sh
cd ui && npm run build && cd .. && go build -o loop_bin . --port 3000
./loop_bin ui --port 3000
```

### CLI usage

```
loop ui              # start web server on :8080
loop ui --port 3000  # custom port
loop ui -p 3000      # shorthand
```

### Available endpoints

| Path | Description |
|---|---|
| `/` | React SPA |
| `/assets/*` | Static assets (embedded from `ui/dist`) |
| `/health` | JSON health check |
| `GET/POST /api/projects` | List / create projects |
| `GET/PATCH/DELETE /api/projects/:id` | Get / rename / delete a project |
| `POST /api/projects/:id/chat` | SSE stream — runs the agent |
| `GET /api/projects/:id/history` | Load chat history from Claude session file |
| `GET /api/agent-types` | List available agent types |
| `GET/PUT /api/settings` | Read / write theme setting |

### Agent types

| Type | How Loop connects |
|---|---|
| `claude-code` | Shells out to `claude` CLI with `--output-format stream-json` |
| `pi` | TCP JSON-RPC 2.0 to a managed Python extension process |
| `docker` | Launches a container (`docker run`), connects via HTTP/SSE (`POST /run`) |
| `remote` | Connects to a user-specified host:port via HTTP/SSE (`POST /run`) |

Docker and remote agents implement the HTTP/SSE extension protocol:
`GET /info` (health + metadata), `POST /run` (SSE stream), `POST /cancel`.
