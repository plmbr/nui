---
layout: docs
title: Extension REST API
subtitle: HTTP endpoints for extensions, deployers, and HITL.
permalink: /docs/extensions/rest-api/
---

These endpoints are served by the nui Go server (default `http://127.0.0.1:8080`). In development, the Vite dev server proxies `/api` to the Go backend.

## Extensions

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/extensions` | Installed extensions and contribution item ids |
| `POST` | `/api/extensions/reload` | Rescan `~/.nui/extensions/` |

### `GET /api/extensions` response shape

```json
{
  "extensions": [
    {
      "name": "corp-pack",
      "displayName": "Corp Pack",
      "version": "1.0.0",
      "disabled": false,
      "harnesses": ["echo", "reverse"],
      "agents": ["echo-bot", "reviewer"],
      "mcpServers": ["corp-tools", "echo-tool"],
      "skills": ["deploy-checklist", "code-review"]
    }
  ]
}
```

Exact fields vary by installed contributions.

## Agent types

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/agent-types` | Builtin, user, and extension ADL agent types |

Extension harnesses appear as `ext:<extension>/<harness-id>`. Extension agents as `ext:<extension>/<agent-id>`.

## Agent deployers

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/agent-deployers` | Installed extension deployers |
| `POST` | `/api/agents/:id/deploy` | Deploy user agent |

Deploy request body:

```json
{
  "deployerId": "ext:docker-deployer/docker"
}
```

See [Agent deployers]({{ '/docs/extensions/deployers/' | relative_url }}) for the stdin/stdout protocol.

## HITL

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/hitl-channels` | Built-in and extension channel ids |
| `POST` | `/api/hitl/requests` | Create a HITL request |
| `GET` | `/api/hitl/requests/:id/wait` | Block until answered (SSE or long-poll) |
| `POST` | `/api/hitl/requests/:id/respond` | Submit an answer |
| `GET` | `/api/hitl/requests?pending=true` | List pending requests |

### Create request

```bash
curl -s -X POST http://127.0.0.1:8080/api/hitl/requests \
  -H 'Content-Type: application/json' \
  -d '{
    "sessionId": "SESSION_ID",
    "runId": "RUN_ID",
    "kind": "question",
    "payload": {
      "title": "Confirm",
      "message": "Proceed?",
      "questions": [{"question": "OK?", "options": ["Yes", "No"]}]
    },
    "routing": {"channels": ["nui-ui"]}
  }'
```

### Respond

```bash
curl -s -X POST http://127.0.0.1:8080/api/hitl/requests/REQUEST_ID/respond \
  -H 'Content-Type: application/json' \
  -d '{
    "status": "answered",
    "answers": [{"question": "OK?", "answer": "Yes"}],
    "respondedBy": {"channel": "nui-ui"}
  }'
```

See [HITL]({{ '/docs/extensions/hitl/' | relative_url }}) for SDK helpers and channel delivery.

## Mentions

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/sessions/:id/mentions` | List mention items for chat `@` menu |

Query params: `parent`, `query`, `limit`

## Sessions (related)

Extension storage handlers integrate with standard session APIs:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/sessions/:id/messages` | UI messages (may read from storage handler) |
| `PUT` | `/api/sessions/:id/messages` | Persist messages |
| `POST` | `/api/sessions/:id/ag-ui` | AG-UI chat stream |

## Capabilities

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/capabilities` | Server capabilities (e.g. bubblewrap availability) |

## Authentication

The nui server binds to localhost by default. There is no built-in API auth — deploy behind a reverse proxy with authentication if exposing beyond localhost.

## Environment variables for extension processes

Set by nui during harness/tool runs:

| Variable | Description |
|----------|-------------|
| `NUI_API_URL` | Base URL for REST calls (e.g. `http://127.0.0.1:8080`) |
| `NUI_SESSION_ID` | Active session uuid |
| `NUI_RUN_ID` | Active run uuid |
| `NUI_EXTENSION_DIR` | Extension install path |
| `NUI_EXTENSION_NAME` | Extension manifest name |
