---
layout: docs
title: Agent deployers
subtitle: Deploy user ADL agents to remote platforms from the CLI.
permalink: /docs/extensions/deployers/
---

Agent deployers are named commands that package and deploy user-defined ADL agents to a remote platform (Docker, Kubernetes, internal PaaS, etc.). Registry URLs, image tags, and authentication are **extension-owned** — nui passes the agent definition and bundled assets.

Deployer ids: `ext:<extension>/<name>`

## Manifest

```yaml
contributions:
  aiAssets:
    agentDeployers:
      - name: docker
        description: Build and deploy agents as Docker images
        command: ["python3", "${NUI_EXTENSION_DIR}/deploy.py"]
```

## CLI

```bash
nui agent deployers
nui agent deploy ext:docker-deployer/docker my-agent
```

HTTP equivalent: `POST /api/agents/:id/deploy` with body `{"deployerId":"ext:docker-deployer/docker"}`

## Invocation protocol

nui spawns the deployer `command`, writes **one JSON line** to stdin, reads **one JSON line** from stdout.

### Request

```json
{
  "action": "deploy",
  "deployerId": "ext:docker-deployer/docker",
  "agentId": "my-agent",
  "definition": {
    "id": "my-agent",
    "name": "My Agent",
    "harness": {"type": "docker", "image": "my-harness:latest"},
    "aiAssets": {}
  },
  "assets": {
    "skills": [],
    "mcpServers": [],
    "rules": []
  }
}
```

| Field | Description |
|-------|-------------|
| `action` | Currently `deploy` |
| `deployerId` | Full deployer id |
| `agentId` | User agent id from `~/.nui/agents/` |
| `definition` | Resolved ADL definition |
| `assets` | Materialized skills, MCP configs, and rules bundled for packaging |

### Response (success)

```json
{
  "ok": true,
  "deploymentId": "nui-my-agent-1.0.0",
  "status": "ready",
  "message": "built image nui-my-agent:1.0.0",
  "endpoint": {
    "host": "127.0.0.1",
    "port": 9090
  }
}
```

### Response (failure)

```json
{
  "ok": false,
  "error": "docker build failed: ..."
}
```

## Example deployer (Python)

Minimal stdin/stdout handler:

```python
#!/usr/bin/env python3
import json
import sys


def emit(resp: dict) -> None:
    sys.stdout.write(json.dumps(resp) + "\n")
    sys.stdout.flush()


def main() -> None:
    line = sys.stdin.readline()
    req = json.loads(line)
    agent_id = req.get("agentId", "agent")
    definition = req.get("definition") or {}

    # Build image, push to registry, start container — extension-specific
    image_tag = f"nui-{agent_id}:latest"
    emit({
        "ok": True,
        "deploymentId": image_tag,
        "status": "ready",
        "message": f"built {image_tag}",
        "endpoint": {"host": "127.0.0.1", "port": 9090},
    })


if __name__ == "__main__":
    main()
```

The full **docker-deployer** example reads `deploy-config.yaml`, generates a Dockerfile, runs `docker build`, and optionally push/run:

```yaml
# deploy-config.yaml
registry: ""
baseImage: python:3.12-slim
push: false
run: false
containerPort: 9090
```

Environment overrides:

| Variable | Effect |
|----------|--------|
| `DOCKER_REGISTRY` | Image registry prefix |
| `DOCKER_DEPLOY_PUSH` | `true` to push after build |
| `DOCKER_DEPLOY_RUN` | `true` to run container locally |
| `DOCKER_BASE_IMAGE` | Override base image |

## Docker harness packaging

Deployers typically produce images compatible with ADL `harness.type: docker` — HTTP/SSE on `containerPort` with `GET /info`, `POST /run`, etc. See [Harness protocols]({{ '/docs/harness-protocols/' | relative_url }}).

The docker-deployer extension includes `nui_agent.py` — a minimal HTTP harness copied into the image.

## Programmatic extensions

```python
from nui_extension import NuiExtension

class MyExtension(NuiExtension):
    def get_deployers(self):
        return [{"name": "docker", "description": "Deploy to Docker"}]

    def deploy(self, deployer_id, agent_id, definition, assets, ctx=None):
        return {
            "ok": True,
            "deploymentId": f"{agent_id}-1",
            "status": "ready",
            "message": "deployed",
        }
```

Wire method: `extension.deploy`

## Install example

```bash
nui extension add dev/extension-examples/docker-deployer
nui agent deploy ext:docker-deployer/docker my-agent
```

Source: [`dev/extension-examples/docker-deployer/`](https://github.com/plmbr/nui/tree/main/dev/extension-examples/docker-deployer/)
