---
layout: page
title: Sandboxing
subtitle: Run agents on the host, in bubblewrap namespaces, or in Docker containers.
permalink: /features/sandbox/
---

Sandbox mode is set per agent via ADL `harness.sandbox` and propagated to the harness subprocess.

## Modes

| Mode | Description |
|---|---|
| `none` | Runs unsandboxed on the host |
| `bubblewrap` | Linux-only namespace isolation via bubblewrap |
| `docker` | Managed Docker container with port mapping and health checks |

Bubblewrap only applies when `sandbox: bubblewrap` is set and bwrap is available on the system. macOS native sandboxing is not yet implemented.

## Bind mounts (bubblewrap)

When bubblewrap is active, harness-specific config directories are bind-mounted:

| Harness | Mount |
|---|---|
| `claude-code` | `~/.claude` |
| `pi` | `~/.pi` |
| `codex` | `~/.codex` |
| `opencode` | `~/.local/share/opencode` |

## Docker harnesses

For `sandbox: docker` on built-in harnesses, nui uses pre-built images:

| Image | Port |
|---|---|
| `nui-claude-code:latest` | 8090 |
| `nui-pi:latest` | 8090 |
| `nui-codex:latest` | 8090 |
| `nui-opencode:latest` | 8090 |

Build images from the `docker/` directory:

```bash
cd docker
docker build -f claude-code/Dockerfile -t nui-claude-code:latest .
```

Custom ADL agents can also use `harness.type: docker` with any image that implements the HTTP/SSE harness protocol.

## Remote harnesses

For agents running on another machine, use `harness.type: remote` with a `host:port`. nui stores the connection and forwards requests over HTTP/SSE — no process management on the remote side beyond what you set up.
