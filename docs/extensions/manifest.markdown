---
layout: docs
title: Extension manifest
subtitle: Complete reference for extension.yaml and contribution blocks.
permalink: /docs/extensions/manifest/
---

Every extension requires `extension.yaml` at the package root (or in exactly one immediate subdirectory when installing from a zip or git repo).

## Top-level fields

```yaml
apiVersion: nui.plmbr.dev/extension/v1   # required
name: corp-pack                          # required; must match directory name
version: 1.0.0                           # semver
displayName: Corp Pack                   # UI label
description: Example extension           # optional

contributions:                           # declarative contributions
  # … see below

runtime:                                 # programmatic extensions only
  transport: stdio
  command: ["python3", "-m", "corp_nui_ext"]

install:                                   # programmatic package provenance
  source: pip
  package: corp-nui-ext==1.0.0
```

| Field | Required | Description |
|-------|----------|-------------|
| `apiVersion` | yes | Must be `nui.plmbr.dev/extension/v1` |
| `name` | yes | Extension id; directory name under `~/.nui/extensions/` |
| `version` | recommended | Semver for display and upgrades |
| `displayName` | recommended | Human-readable name in the UI |
| `description` | optional | Short summary |
| `contributions` | declarative | Static contribution lists and runtimes |
| `runtime` | programmatic | How nui spawns the extension process |
| `install` | programmatic | Package manager metadata for upgrade/remove |

## contributions.aiAssets

Inline assets injected into sessions when referenced via `ref:` in agent ADL.

### Custom MCP servers (command tools)

```yaml
contributions:
  aiAssets:
    mcpServers:
      - name: corp-tools
        tools:
          - name: echo
            description: Echo a message back
            command: ["python3", "${NUI_EXTENSION_DIR}/tools/echo.py"]
            inputSchema:
              type: object
              properties:
                message:
                  type: string
                  description: Text to echo
              required: [message]
          - name: reverse
            description: Reverse a message
            command: ["python3", "${NUI_EXTENSION_DIR}/tools/reverse.py"]
```

| Tool field | Description |
|------------|-------------|
| `name` | Tool name exposed to the harness |
| `description` | Shown in MCP `tools/list` |
| `command` | argv; `${NUI_EXTENSION_DIR}` expands to the extension path |
| `inputSchema` | JSON Schema for tool arguments (default: single required `message` string) |

Referenced as `ref: ext:<extension>/corp-tools` in ADL `aiAssets.mcpServers`.

### Custom skills

```yaml
    skills:
      - name: deploy-checklist
        content: |
          ---
          name: deploy-checklist
          description: Pre-deploy verification steps
          ---
          Run through the checklist before merging.
      - name: from-file
        path: skills/review.md
```

Use `name`, `path`, and/or `content` — same schema as ADL `aiAssets.skills`.

### Rules

```yaml
    rules:
      - name: corp-guidelines
        content: |
          Follow corporate security guidelines. Never commit secrets.
      - name: from-file
        path: rules/security.md
```

Referenced as `ref: ext:<extension>/corp-guidelines` in ADL `aiAssets.rules`.

### Agent deployers

```yaml
    agentDeployers:
      - name: docker
        description: Build and deploy agents as Docker images
        command: ["python3", "${NUI_EXTENSION_DIR}/deploy.py"]
```

See [Agent deployers]({{ '/docs/extensions/deployers/' | relative_url }}).

## contributions.catalog

Discoverable lists for the customize UI and ADL `ref:` resolution. **Resolution order** per list type: `source.file` → `source.command` → `catalog.command`.

```yaml
  catalog:
    mcpServers:
      source:
        file: mcp-servers.json
    skills:
      source:
        file: skills.yaml
    command: ["python3", "catalog.py"]   # optional shared list provider
```

List files may be JSON or YAML with a top-level array key (`mcpServers`, `skills`).

Example `mcp-servers.json`:

```json
{
  "mcpServers": [
    {
      "name": "echo-tool",
      "description": "Echo MCP server",
      "command": ["npx", "-y", "@example/echo-mcp"]
    }
  ]
}
```

Example `skills.yaml`:

```yaml
skills:
  - name: code-review
    description: Code review skill
    path: skills/code-review.md
```

## contributions.harnesses

```yaml
  harnesses:
    source:
      file: harnesses.yaml
    runtime:
      transport: stdio       # stdio | tcp | http
      command: ["python3", "harness_host.py"]
      # host: 127.0.0.1     # tcp/http
      # port: 0             # tcp — 0 = ephemeral
```

`harnesses.yaml`:

```yaml
harnesses:
  - id: echo
    displayName: Echo
    description: Repeats your message
```

Agent type id: `ext:<extension>/<id>`.

## contributions.agents

```yaml
  agents:
    source:
      file: agents.yaml
```

Full ADL agent definitions. IDs namespaced as `ext:<extension>/<agent-id>`.

## contributions.mentionProviders

```yaml
  mentionProviders:
    source:
      file: mention-providers.yaml
    runtime:
      transport: stdio
      command: ["python3", "mention_host.py"]
```

`mention-providers.yaml`:

```yaml
mentionProviders:
  - id: corp-refs
    displayName: Corp References
```

Agents opt in via `aiAssets.mentionProviders`. See [Mention providers]({{ '/docs/extensions/mentions/' | relative_url }}).

## contributions.hitlChannels

```yaml
  hitlChannels:
    source:
      file: hitl-channels.yaml
    runtime:
      transport: stdio
      command: ["python3", "hitl_channel_host.py"]
```

`hitl-channels.yaml`:

```yaml
hitlChannels:
  - id: demo-slack
    displayName: Demo Slack Channel
    description: Forwards HITL prompts to Slack (example)
```

Referenced in ADL as `ext:<extension>/demo-slack`. See [HITL]({{ '/docs/extensions/hitl/' | relative_url }}).

## contributions.storage

```yaml
  storage:
    source:
      file: storage-handlers.yaml
    runtime:
      command: ["python3", "storage_host.py"]
```

`storage-handlers.yaml`:

```yaml
storageHandlers:
  - id: postgres-sessions
    kind: sessionHistory
    agentTypes: ["ext:corp/reviewer", "claude-code"]
  - id: agent-notes
    kind: agentMemory
    agentTypes: ["ext:corp/reviewer"]
  - id: user-cloud
    kind: userMemory
```

See [Storage handlers]({{ '/docs/extensions/storage/' | relative_url }}).

## Environment variable expansion

| Variable | Available in |
|----------|--------------|
| `${NUI_EXTENSION_DIR}` | `command` arrays in manifest and list files |
| `${localEnv:VAR}` | ADL and some harness configs (not extension yaml) |

## Legacy fields (deprecated)

These still work but prefer `contributions.aiAssets` and `contributions.catalog`:

- Root-level `mcpServers`
- `contributions.mcpServers` / `contributions.skills` without `catalog` or `aiAssets` wrapper

## JSON Schema

Machine-readable schema: [`sdk/protocol/extension-v1.schema.json`](https://github.com/plmbr/nui/blob/main/sdk/protocol/extension-v1.schema.json)

## Full example manifest

```yaml
apiVersion: nui.plmbr.dev/extension/v1
name: corp-pack
version: 1.0.0
displayName: Corp Pack
description: Example extension with harnesses, MCP servers, skills, and agents

contributions:
  aiAssets:
    mcpServers:
      - name: corp-tools
        tools:
          - name: echo
            description: Echo a message back
            command: ["python3", "${NUI_EXTENSION_DIR}/tools/echo.py"]
    skills:
      - name: deploy-checklist
        content: |
          ---
          name: deploy-checklist
          description: Pre-deploy verification steps
          ---
          Run through the checklist before merging.
    rules:
      - name: corp-guidelines
        content: |
          Follow corporate security guidelines. Never commit secrets.

  catalog:
    mcpServers:
      source:
        file: mcp-servers.json
    skills:
      source:
        file: skills.yaml

  harnesses:
    source:
      file: harnesses.yaml
    runtime:
      transport: stdio
      command: ["python3", "harness_host.py"]

  agents:
    source:
      file: agents.yaml

  mentionProviders:
    source:
      file: mention-providers.yaml
    runtime:
      transport: stdio
      command: ["python3", "mention_host.py"]
```
