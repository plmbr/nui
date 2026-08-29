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

On macOS, Finder/Dock launches inherit a stripped `PATH`. At startup the app
merges the login-shell `PATH` and common install dirs (`~/.local/bin`, Homebrew,
nvm default, `~/.opencode/bin`, …) so builtin CLI agents (`claude`, `pi`,
`codex`, `opencode`) are detected in the new-session picker and runnable from
harnesses.

### Bundled CLI (first-launch install)

`build-desktop.sh` also builds a `CGO_ENABLED=0` `nui` CLI and stages it:

- macOS: `nui.app/Contents/Resources/nui`
- Windows / Linux: `nui.exe` / `nui` next to the desktop binary

On GUI launch (after MCP dispatch), the app copies that binary into the same
dirs as the public installers when missing:

| OS | Install path |
|---|---|
| macOS / Linux | `~/.local/bin/nui` |
| Windows | `%LOCALAPPDATA%\nui\nui.exe` |

An existing install at that path is never replaced (for example when you installed
a newer CLI via `install.sh` before opening the desktop app).

Override with `NUI_INSTALL_DIR`. State is recorded in `~/.nui/desktop-cli.json`.

PATH setup:

- Windows: adds the install dir to the user `Path` when absent
- Unix: prepends the dir for the current process, and appends an idempotent
  `# >>> nui-desktop-cli >>>` block to `~/.zprofile` (zsh) or
  `~/.bash_profile` / `~/.profile` when the dir is not already on `PATH`

Open a new terminal (or restart Cursor) after first launch so external MCP
configs with `"command": "nui"` pick up the install. `wails dev` / `go run`
without a staged sidecar is a no-op.

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

### macOS Gatekeeper

Downloaded `nui.app` builds are ad-hoc signed (not notarized), so Gatekeeper may show “malware” / “damaged” until quarantine is cleared:

```sh
xattr -cr ~/Downloads/nui.app
open ~/Downloads/nui.app
```

Developer ID signing + notarization is documented under “Future macOS codesigning” in [DEVELOPERS.md](../DEVELOPERS.md).

## Notes

- Desktop archives bundle a companion CLI for first-launch PATH install (see above).
  Standalone CLI releases (`scripts/build-release.sh`) remain available for
  curl/irm installs without the desktop app.
- This directory is its own Go module (`nui/desktop`) with `replace nui => ../`
  so root `go test ./...` and static CLI releases stay free of Wails/CGO.
