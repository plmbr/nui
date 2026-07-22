---
layout: docs
title: Extension API
subtitle: Add harnesses, MCP servers, skills, agents, HITL, storage, and deployers to nui.
permalink: /docs/extensions/
---

Extensions are packages installed under `~/.nui/extensions/<name>/`. Each extension has a manifest (`extension.yaml`) and optional contribution list files. nui loads them at startup and exposes contributed harnesses and agents in `GET /api/agent-types`.

## Two extension shapes

| Shape | Discovery | `extension.yaml` |
|-------|-----------|------------------|
| **Declarative pack** | nui reads yaml and list files | Full `contributions` block |
| **Programmatic package** | `extension.initialize` IPC → `getHarnesses()`, … | `runtime` + `install` only |

Most examples in this documentation use **declarative packs**. For Python/TS/Go packages with a `NuiExtension` subclass, see [Programmatic SDK]({{ '/docs/extensions/programmatic/' | relative_url }}).

## Install and manage

```bash
# Local directory or zip
nui extension add ./my-extension
nui extension add ./corp-pack.zip

# Git repository (shallow clone, copy to ~/.nui/extensions/)
nui extension add https://github.com/example/my-extension.git

# npm / pip / go packages (programmatic)
nui extension add npm:@corp/nui-ext@1.0.0
nui extension add pip:corp-nui-ext==1.0.0

nui extension list
nui extension remove corp-pack
```

Re-installing replaces the existing copy. Pick up changes without restarting:

```bash
curl -X POST http://127.0.0.1:8080/api/extensions/reload
```

Or use **Settings → Extensions** in the UI to enable/disable extensions.

## Directory layout

```
~/.nui/extensions/
  corp-pack/
    extension.yaml       # manifest (required)
    harnesses.yaml       # harness list (optional)
    mcp-servers.json     # catalog MCP servers (optional)
    skills.yaml          # catalog skills (optional)
    agents.yaml          # ADL agents (optional)
    tools/               # scripts for custom MCP tools (optional)
    harness_host.py      # stdio harness runtime
    mention_host.py      # mention provider runtime
    hitl_channel_host.py # HITL channel runtime
    storage_host.py      # storage handler runtime
```

Try the shipped examples:

```bash
nui extension add dev/extension-examples/corp-pack
nui extension add dev/extension-examples/hitl-demo
nui extension add dev/extension-examples/storage-demo
nui extension add dev/extension-examples/docker-deployer
```

(When installing from a git clone of nui, use the path relative to your checkout.)

## Contribution types

| Contribution | ADL / runtime | Docs |
|--------------|---------------|------|
| `harnesses` | `harness.type: ext:<ext>/<id>` | [Harnesses]({{ '/docs/extensions/harnesses/' | relative_url }}) |
| `aiAssets.mcpServers` | Custom command-tool MCP servers | [MCP, skills & rules]({{ '/docs/extensions/mcp-skills-rules/' | relative_url }}) |
| `aiAssets.skills` / `rules` | `ref: ext:<ext>/<name>` | [MCP, skills & rules]({{ '/docs/extensions/mcp-skills-rules/' | relative_url }}) |
| `catalog` | Discoverable MCP/skills lists | [MCP, skills & rules]({{ '/docs/extensions/mcp-skills-rules/' | relative_url }}) |
| `agents` | `ext:<ext>/<agent-id>` agent types | [Getting started]({{ '/docs/extensions/getting-started/' | relative_url }}) |
| `mentionProviders` | `aiAssets.mentionProviders` refs | [Mentions]({{ '/docs/extensions/mentions/' | relative_url }}) |
| `hitlChannels` | `hitl.channels` in ADL | [HITL]({{ '/docs/extensions/hitl/' | relative_url }}) |
| `storage` | Replaces built-in persistence per scope | [Storage]({{ '/docs/extensions/storage/' | relative_url }}) |
| `aiAssets.agentDeployers` | `nui agent deploy` | [Deployers]({{ '/docs/extensions/deployers/' | relative_url }}) |

## Referencing extension assets in ADL

Extension ids use the `ext:<extension>/<item-id>` convention:

```yaml
harness:
  type: ext:corp-pack/echo

aiAssets:
  mcpServers:
    - ref: ext:corp-pack/corp-tools
  skills:
    - ref: ext:corp-pack/deploy-checklist
  rules:
    - ref: ext:corp-pack/corp-guidelines
  mentionProviders:
    - ref: ext:corp-pack/corp-refs

hitl:
  mode: interactive
  channels:
    - nui-ui
    - ext:hitl-demo/demo-slack
```

## Python harness SDK

nui copies author-facing modules to `~/.nui/harness-sdk/` on first use:

| Module | Purpose |
|--------|---------|
| `nui_agent_stdio.py` | Declarative stdio harness framework |
| `nui_extension.py` | Programmatic extension base class |
| `nui_catalog.py` | Dynamic catalog RPC |
| `nui_hitl.py` | REST HITL client |
| `nui_hitl_channel.py` | Stdio HITL channel host |
| `nui_mention.py` | Stdio mention provider host |
| `nui_mcp_tools.py` | Stdio MCP proxy for custom tools |
| `nui_storage.py` | Stdio storage handler host |

Reinstall from CLI: `nui harness-sdk reinstall`

## Security

Extensions run as the nui user with full host access — equivalent to shell scripts and MCP server commands. **Only install extensions you trust.**

## Documentation map

1. [Getting started]({{ '/docs/extensions/getting-started/' | relative_url }}) — walkthrough with corp-pack
2. [Manifest]({{ '/docs/extensions/manifest/' | relative_url }}) — complete `extension.yaml` reference
3. [Harnesses]({{ '/docs/extensions/harnesses/' | relative_url }}) — stdio, TCP, HTTP
4. [MCP, skills & rules]({{ '/docs/extensions/mcp-skills-rules/' | relative_url }})
5. [Mention providers]({{ '/docs/extensions/mentions/' | relative_url }})
6. [HITL]({{ '/docs/extensions/hitl/' | relative_url }})
7. [Storage handlers]({{ '/docs/extensions/storage/' | relative_url }})
8. [Agent deployers]({{ '/docs/extensions/deployers/' | relative_url }})
9. [Programmatic SDK]({{ '/docs/extensions/programmatic/' | relative_url }})
10. [Dynamic catalog]({{ '/docs/extensions/catalog/' | relative_url }})
11. [REST API]({{ '/docs/extensions/rest-api/' | relative_url }})
