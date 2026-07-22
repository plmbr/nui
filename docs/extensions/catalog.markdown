---
layout: docs
title: Dynamic catalog provider
subtitle: JSON-RPC list providers for harnesses, MCP servers, skills, and agents.
permalink: /docs/extensions/catalog/
---

When contribution lists change at runtime (database, API, tenant config), use a **catalog provider** — a stdio JSON-RPC process that answers list queries instead of static yaml/json files.

## Source resolution order

Per list type (`mcpServers`, `skills`, harnesses, agents):

1. `contributions.catalog.<type>.source.file`
2. `contributions.catalog.<type>.source.command`
3. `contributions.catalog.command` (shared provider for all list types)

## Manifest

### Per-type command

```yaml
contributions:
  catalog:
    mcpServers:
      source:
        command: ["python3", "catalog_mcp.py"]
    skills:
      source:
        command: ["python3", "catalog_skills.py"]
```

### Shared command

```yaml
contributions:
  catalog:
    command: ["python3", "catalog.py"]
```

One process handles all list methods.

## Wire protocol

| Method | Result |
|--------|--------|
| `extension.initialize` | `{apiVersion, extensionName, capabilities}` |
| `extension.listHarnesses` | `{harnesses: [...]}` |
| `extension.listMCPServers` | `{mcpServers: [...]}` |
| `extension.listSkills` | `{skills: [...]}` |
| `extension.listAgents` | `{agents: [...]}` |
| `extension.shutdown` | cleanup |

List item shapes match static file entries — see [Manifest]({{ '/docs/extensions/manifest/' | relative_url }}).

## Python framework

[`harness-sdk/nui_catalog.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_catalog.py)

```python
#!/usr/bin/env python3
from nui_catalog import NuiCatalogProvider


class CorpCatalog(NuiCatalogProvider):
    name = "corp-catalog"
    version = "1.0.0"

    def list_harnesses(self):
        return [
            {"id": "echo", "displayName": "Echo"},
            {"id": "reverse", "displayName": "Reverse"},
        ]

    def list_mcp_servers(self):
        return [{
            "name": "dynamic-tool",
            "description": "Loaded from API",
            "command": ["npx", "-y", "my-mcp"],
        }]

    def list_skills(self):
        return [{
            "name": "oncall",
            "description": "Oncall runbook",
            "path": "skills/oncall.md",
        }]

    def list_agents(self):
        return []


if __name__ == "__main__":
    CorpCatalog().serve()
```

## Programmatic extensions

Programmatic `NuiExtension` subclasses return lists from `get_harnesses()`, `get_mcp_servers()`, `get_skills()`, `get_agents()` during `extension.initialize` — no separate catalog process required.

```python
class MyExt(NuiExtension):
    def get_harnesses(self):
        return fetch_from_api("/harnesses")

    def get_skills(self):
        return load_skills_from_db()
```

## Hybrid pattern

Common approach:

- **Static** `aiAssets` for bundled tools and rules
- **Dynamic catalog** for tenant-specific MCP servers and skills
- **Static** `harnesses.yaml` + multiplex host for simple harnesses

```yaml
contributions:
  aiAssets:
    mcpServers:
      - name: bundled-tools
        tools: [...]
  catalog:
    mcpServers:
      source:
        command: ["python3", "tenant_catalog.py"]
  harnesses:
    source:
      file: harnesses.yaml
    runtime:
      command: ["python3", "harness_host.py"]
```

## Lifecycle

nui spawns the catalog provider when resolving lists (customize UI, agent type enumeration, ref resolution). Calls `extension.shutdown` on server stop or extension reload.

Errors in the provider are logged; nui falls back to empty lists for that contribution type.
