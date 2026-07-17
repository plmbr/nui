# Devcontainer Harness Example

nui-managed devcontainer sandbox: **no user `devcontainer.json` required**. When ADL sets `harness.type: devcontainer`, nui generates the devcontainer config per session and runs the inner CLI inside the container.

## Prerequisites

```sh
npm install -g @devcontainers/cli
```

Docker must be running. nui **auto-builds** default `nui-devcontainer-*` images on first use.

## Install the agent

```sh
cp dev/harness-examples/devcontainer/devcontainer-claude.yaml ~/.nui/agents/
```

## Usage

1. In nui UI, create a session under **Installed agents** → **Devcontainer Claude**
2. Optionally set a working directory (defaults to `~/.nui/workspaces/<session-id>`)
3. Send a message — nui runs `devcontainer up` on first use, then `devcontainer exec claude ...`

## ADL configuration

```yaml
harness:
  type: devcontainer
  innerHarness: claude-code   # claude-code | pi | codex | opencode
  image: nui-devcontainer-claude-code:latest  # optional override
```

## Build default images manually (optional)

nui builds these automatically on first session use. To pre-build:

```sh
docker build -t nui-devcontainer-claude-code:latest docker/devcontainer-claude-code
docker build -t nui-devcontainer-pi:latest docker/devcontainer-pi
docker build -t nui-devcontainer-codex:latest docker/devcontainer-codex
docker build -t nui-devcontainer-opencode:latest docker/devcontainer-opencode
```

## How it differs from `harness.type: docker`

| | `docker` harness | `devcontainer` harness |
|--|------------------|------------------------|
| User setup | Build/push custom HTTP image | None — nui provisions config |
| Agent transport | HTTP/SSE | Direct CLI via `devcontainer exec` |
| Config | ADL `image` + `containerPort` | ADL `innerHarness` only |

## Lifecycle

| Event | What nui does |
|-------|----------------|
| First message | Auto-build image if missing, generate `~/.nui/sessions/<id>/.devcontainer/devcontainer.json`, `devcontainer up` |
| Chat message | `devcontainer exec <inner-cli> ...` |
| Session delete | `docker stop` |
| Idle 30 min | Stop container (shared idle reaper) |
