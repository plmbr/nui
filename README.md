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
├── main.go                 # entrypoint
├── embed.go                # embeds ui/dist into the binary
├── cmd/                    # cobra CLI commands
├── internal/server/        # HTTP server
└── ui/                     # Vite + React frontend
    └── dist/               # built output (generated, not committed)
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
cd ui && npm run build && cd .. && go build -o loop_bin .
./loop_bin ui
```

### CLI usage

```
loop ui              # start web server on :8080
loop ui --port 3000  # custom port
loop ui -p 3000      # shorthand
```

### Available endpoints

| Path       | Description          |
|------------|----------------------|
| `/`        | React app            |
| `/assets/` | Static assets        |
| `/health`  | JSON health check    |
