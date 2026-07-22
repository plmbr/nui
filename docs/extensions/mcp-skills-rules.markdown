---
layout: docs
title: MCP servers, skills & rules
subtitle: Custom command tools, catalog lists, and ADL ref resolution.
permalink: /docs/extensions/mcp-skills-rules/
---

Extensions can contribute MCP servers, skills, and rules in two ways: **inline aiAssets** (bundled with the extension) and **catalog lists** (discoverable entries for the customize UI and ADL `ref:`).

## Custom MCP servers (aiAssets)

Command-tool MCP servers group multiple tools. Each tool runs a CLI command; nui passes tool arguments as **JSON on stdin**.

### Manifest

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
            inputSchema:
              type: object
              properties:
                message:
                  type: string
              required: [message]
```

| Field | Description |
|-------|-------------|
| `name` | Server name — used in `ref: ext:<extension>/<name>` |
| `tools[].name` | MCP tool name |
| `tools[].description` | Shown in `tools/list` |
| `tools[].command` | argv; `${NUI_EXTENSION_DIR}` expands at materialization time |
| `tools[].inputSchema` | JSON Schema (optional; default is `{message: string}` required) |

### Tool script contract

```python
#!/usr/bin/env python3
"""Custom MCP tool — read JSON args from stdin, print result to stdout."""

import json
import sys

args = json.load(sys.stdin)
message = args.get("message", "")
print(f"echo: {message}")
```

Exit code non-zero → tool error. stderr is captured for diagnostics.

### HITL from a tool script

```python
#!/usr/bin/env python3
import json
import os
import sys

sys.path.insert(0, os.path.expanduser("~/.nui/harness-sdk"))
from nui_hitl import ask_user

args = json.load(sys.stdin)
if not args.get("confirmed"):
    answer = ask_user(questions=[{
        "question": f"Run action on {args.get('target')}?",
        "options": ["Yes", "No"],
    }])
    # handle answer …
print("ok")
```

### Materialization

nui uses [`harness-sdk/nui_mcp_tools.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_mcp_tools.py) (copied to `~/.nui/harness-sdk/`). Harness config is written per session under `~/.nui/sessions/<session-id>/`.

Override proxy location: `NUI_MCP_TOOLS_PATH`

### ADL reference

```yaml
aiAssets:
  mcpServers:
    - name: corp-tools
      ref: ext:corp-pack/corp-tools
```

## Catalog MCP servers

Standard MCP servers (stdio, http, sse) listed for discovery — same schema as ADL `aiAssets.mcpServers`.

### File source

`mcp-servers.json`:

```json
{
  "mcpServers": [
    {
      "name": "echo-tool",
      "description": "Example stdio MCP server",
      "command": ["npx", "-y", "@example/echo-mcp"],
      "env": {
        "LOG_LEVEL": "info"
      }
    }
  ]
}
```

```yaml
contributions:
  catalog:
    mcpServers:
      source:
        file: mcp-servers.json
```

ADL:

```yaml
aiAssets:
  mcpServers:
    - ref: ext:corp-pack/echo-tool
```

## Skills

### Inline aiAssets skills

```yaml
contributions:
  aiAssets:
    skills:
      - name: deploy-checklist
        content: |
          ---
          name: deploy-checklist
          description: Pre-deploy verification steps
          ---
          1. Run tests
          2. Check migrations
          3. Verify feature flags
      - name: review
        path: skills/code-review.md
```

ADL: `ref: ext:corp-pack/deploy-checklist`

### Catalog skills

`skills.yaml`:

```yaml
skills:
  - name: code-review
    description: Structured code review skill
    path: skills/code-review.md
```

```yaml
contributions:
  catalog:
    skills:
      source:
        file: skills.yaml
```

Paths resolve relative to the extension directory.

## Rules

Rules are markdown instruction files provisioned to harness-specific config directories.

```yaml
contributions:
  aiAssets:
    rules:
      - name: corp-guidelines
        content: |
          # Corporate guidelines
          - Never commit secrets
          - Use approved dependencies only
      - name: security
        path: rules/security.md
```

ADL:

```yaml
aiAssets:
  rules:
    - ref: ext:corp-pack/corp-guidelines
```

### Harness materialization

| Harness | Output path | Registration |
|---------|-------------|--------------|
| `claude-code` | `rules/<name>.md` | Auto-discovered by Claude Code |
| `codex` | `rules/<name>.md` | `config.toml` `instructions` |
| `pi` | `pi-agent/rules/<name>.md` | Claude-compatible layout |
| `opencode` | `rules/<name>.md` | `opencode.json` `instructions` |

## Catalog source resolution

Per list type (mcpServers, skills):

1. `contributions.catalog.<type>.source.file`
2. `contributions.catalog.<type>.source.command`
3. `contributions.catalog.command` (shared provider)

See [Dynamic catalog]({{ '/docs/extensions/catalog/' | relative_url }}) for RPC methods.

## Combining assets in an agent

```yaml
# agents.yaml (extension-contributed agent)
agents:
  - id: reviewer
    name: Code Reviewer
    harness:
      type: claude-code
      model: claude-sonnet-4-6
    aiAssets:
      skills:
        - name: code-review
          ref: ext:corp-pack/code-review
      mcpServers:
        - name: echo-tool
          ref: ext:corp-pack/echo-tool
        - name: corp-tools
          ref: ext:corp-pack/corp-tools
      rules:
        - name: corp-guidelines
          ref: ext:corp-pack/corp-guidelines
```

## Legacy formats (deprecated)

Still supported:

- Root-level `mcpServers` in `extension.yaml`
- `contributions.mcpServers` / `contributions.skills` without `aiAssets` / `catalog` wrapper

Prefer the `contributions.aiAssets` + `contributions.catalog` structure for new extensions.
