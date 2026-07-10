# Devcontainer Harness Example

Loop-managed devcontainer sandbox: **no user `devcontainer.json` required**. When ADL sets `harness.type: devcontainer`, Loop generates the devcontainer config per session and runs the inner CLI inside the container.

## Prerequisites

```sh
npm install -g @devcontainers/cli
```

Docker must be running. Loop **auto-builds** default `loop-devcontainer-*` images on first use.

## Install the agent

```sh
cp dev/harness-examples/devcontainer/devcontainer-claude.yaml ~/.loop/agents/
```

## Usage

1. In Loop UI, create a session under **Installed agents** → **Devcontainer Claude**
2. Optionally set a working directory (defaults to `~/.loop/workspaces/<session-id>`)
3. Send a message — Loop runs `devcontainer up` on first use, then `devcontainer exec claude ...`

## ADL configuration

```yaml
harness:
  type: devcontainer
  innerHarness: claude-code   # claude-code | pi | codex | opencode
  image: loop-devcontainer-claude-code:latest  # optional override
```

## Build default images manually (optional)

Loop builds these automatically on first session use. To pre-build:

```sh
docker build -t loop-devcontainer-claude-code:latest docker/devcontainer-claude-code
docker build -t loop-devcontainer-pi:latest docker/devcontainer-pi
docker build -t loop-devcontainer-codex:latest docker/devcontainer-codex
docker build -t loop-devcontainer-opencode:latest docker/devcontainer-opencode
```

## How it differs from `harness.type: docker`

| | `docker` harness | `devcontainer` harness |
|--|------------------|------------------------|
| User setup | Build/push custom HTTP image | None — Loop provisions config |
| Agent transport | HTTP/SSE | Direct CLI via `devcontainer exec` |
| Config | ADL `image` + `containerPort` | ADL `innerHarness` only |

## Lifecycle

| Event | What Loop does |
|-------|----------------|
| First message | Auto-build image if missing, generate `~/.loop/sessions/<id>/.devcontainer/devcontainer.json`, `devcontainer up` |
| Chat message | `devcontainer exec <inner-cli> ...` |
| Session delete | `docker stop` |
| Idle 30 min | Stop container (shared idle reaper) |
