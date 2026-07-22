---
layout: page
title: Extensions
subtitle: Install harnesses, MCP servers, skills, and agents from local directories, zip files, or git URLs.
permalink: /features/extensions/
---

Extensions add capabilities to nui via a manifest (`extension.yaml`) and contribution list files. Installed extensions live in `~/.nui/extensions/<name>/`.

## Install

```bash
nui extension add ./my-extension
nui extension add https://github.com/example/my-extension.git
nui extension remove my-extension
```

Manage installed extensions from **Settings → Extensions**, or disable individual extensions without uninstalling.

## Contribution types

| Contribution | Description |
|---|---|
| `harnesses` | stdio, TCP, or HTTP harness agents |
| `mcpServers` | MCP server definitions injected into harnesses |
| `skills` | Skill files available to agents |
| `agents` | ADL agent definitions |

## Harness transports

Extension harnesses wire through three transports:

| Transport | Go client |
|---|---|
| `stdio` (default) | `stdioHarnessAgent` |
| `tcp` | `ExtensionAgent` (JSON-RPC 2.0) |
| `http` | `HTTPExtensionAgent` |

ADL agents reference extension harnesses as `harness.type: ext:<extension>/<harness-id>`.

## Framework

Extension authors can use the Python harness SDK:

- [`harness-sdk/nui_agent_stdio.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_agent_stdio.py) — stdio harness framework
- [`dev/extension-api.md`](https://github.com/plmbr/nui/blob/main/dev/extension-api.md) — manifest schema and contribution format
