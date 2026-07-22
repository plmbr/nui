---
layout: docs
title: Documentation
subtitle: Developer guides, extension API reference, CLI, and harness protocols.
permalink: /docs/
---

nui is a self-hosted web UI for interactive AI agent sessions. This site hosts the **developer documentation** — everything you need to build extensions, custom harnesses, and ADL agents.

## Quick links

| Topic | Description |
|-------|-------------|
| [Developer guide]({{ '/docs/developers/' | relative_url }}) | Build from source, architecture, REST API, contributing |
| [ADL]({{ '/docs/adl/' | relative_url }}) | Agent Definition Language — YAML schema for custom agents |
| [Harness protocols]({{ '/docs/harness-protocols/' | relative_url }}) | HTTP/SSE, stdio JSON-RPC, and TCP harness wire formats |
| [Extension API]({{ '/docs/extensions/' | relative_url }}) | Full reference for extension manifests, SDKs, and examples |
| [CLI reference]({{ '/cli/' | relative_url }}) | `nui server`, extensions, skills, memory, evals, schedules |

## Extension API (start here for extension authors)

Extensions add harnesses, MCP servers, skills, rules, agents, mention providers, HITL channels, storage backends, and deployers to nui. They install into `~/.nui/extensions/<name>/` and are discovered at server start.

```bash
nui extension add ./my-extension
nui extension add https://github.com/example/my-extension.git
nui extension list
nui extension remove my-extension
```

**Recommended reading order:**

1. [Getting started]({{ '/docs/extensions/getting-started/' | relative_url }}) — install the corp-pack example, directory layout
2. [Manifest]({{ '/docs/extensions/manifest/' | relative_url }}) — `extension.yaml` schema and contribution types
3. [Harnesses]({{ '/docs/extensions/harnesses/' | relative_url }}) — stdio, TCP, and HTTP extension harnesses
4. [MCP, skills & rules]({{ '/docs/extensions/mcp-skills-rules/' | relative_url }}) — custom tools, catalog lists, ADL refs
5. [Programmatic SDK]({{ '/docs/extensions/programmatic/' | relative_url }}) — Python, TypeScript, and Go `NuiExtension` packages

Shipped examples in the [main repository](https://github.com/plmbr/nui/tree/main/dev/extension-examples/):

| Example | What it demonstrates |
|---------|---------------------|
| `corp-pack` | Harnesses, custom MCP tools, skills, rules, catalog, agents, mentions |
| `hitl-demo` | HITL channels, REST bridge, `ask_user` from tools |
| `storage-demo` | Session history and memory persistence handlers |
| `docker-deployer` | Agent deployer that builds Docker images |
| `programmatic-echo` | Programmatic Python extension (no static contribution files) |

## Features overview

Product-oriented feature pages (less technical depth):

- [Built-in agents]({{ '/features/agents/' | relative_url }})
- [ADL agents]({{ '/features/adl/' | relative_url }})
- [Extensions]({{ '/features/extensions/' | relative_url }})
- [MCP integration]({{ '/features/mcp/' | relative_url }})
- [Sandboxing]({{ '/features/sandbox/' | relative_url }})
- [Headless & scheduled runs]({{ '/features/headless/' | relative_url }})

## Source repository

Implementation details and the latest markdown sources also live in the [plmbr/nui](https://github.com/plmbr/nui) repository under `dev/` and `DEVELOPERS.md`. This site is the canonical **on-site** developer reference.
