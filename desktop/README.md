# nui desktop

Native desktop shell for nui, built with [Wails v2](https://wails.io/).

The window hosts a platform webview that navigates to the existing local HTTP
server (`http://127.0.0.1:8080` by default) and React UI. REST and AG-UI SSE
stay on real `net/http` (not the Wails asset server), so streaming works.

## Prerequisites

- Same as the main nui project (Go 1.26+, Node for UI builds)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2.10+ (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`)
- Platform webview libraries (macOS: built-in; Windows: WebView2; Linux: webkit2gtk)

## Develop

Build the UI once (embedded into the desktop binary via `nui/ui`):

```sh
cd ../ui && npm run build
```

Run the desktop app:

```sh
cd ../desktop
wails dev
# or:
CGO_ENABLED=1 go run .
```

If `nui server` is already listening on the port, the desktop app attaches to it
instead of starting a second server. Override the port with `NUI_PORT`.

Harnesses spawn built-in MCP servers via `os.Executable()` (`viz-mcp`,
`agent-mcp`, `hitl-mcp`, `orchestrator-mcp`). The desktop binary handles those
args on stdio before opening the GUI so MCP works without a separate `nui` CLI.

## Build

```sh
../scripts/build-desktop.sh
```

Options for CI / cross-target native builds:

```sh
./scripts/build-desktop.sh --platform linux/amd64 --tags webkit2_41
./scripts/build-desktop.sh --skip-ui --platform darwin/arm64
```

Package a release archive from `desktop/build/bin/`:

```sh
./scripts/package-desktop.sh v0.4.0 darwin arm64
```

Or manually:

```sh
cd ../ui && npm run build
cd ../desktop && wails build
```

Output is under `desktop/build/bin/`. GitHub Actions builds desktop on darwin (amd64/arm64), windows (amd64), and linux (amd64/arm64) in CI and on release.

## Notes

- The CLI binary (`nui`) remains a separate `CGO_ENABLED=0` build from the repo
  root. Desktop packaging does not replace it.
- This directory is its own Go module (`nui/desktop`) with `replace nui => ../`
  so root `go test ./...` and static CLI releases stay free of Wails/CGO.
