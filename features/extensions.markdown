---
layout: page
title: Extensions
subtitle: Install harnesses, MCP servers, skills, and agents from local directories, zip files, or git URLs.
permalink: /features/extensions/
---

Extensions add capabilities to nui via a manifest (`extension.yaml`) and contribution list files. Installed extensions live in `~/.nui/extensions/<name>/`.

**Full developer reference:** [Extension API documentation]({{ '/docs/extensions/' | relative_url }}) — manifest schema, harness SDK, HITL, storage, deployers, and worked examples.

## Install

```bash
nui extension add ./my-extension
nui extension add https://github.com/example/my-extension.git
nui extension list
nui extension remove my-extension
```

Manage installed extensions from **Settings → Extensions**, or disable individual extensions without uninstalling.

## Contribution types

| Contribution | Description |
|---|---|
| `harnesses` | stdio, TCP, or HTTP harness agents |
| `aiAssets.mcpServers` | Custom command-tool MCP servers |
| `aiAssets.skills` / `rules` | Bundled skills and instruction files |
| `catalog` | Discoverable MCP servers and skills |
| `agents` | ADL agent definitions |
| `mentionProviders` | `@`-mention autocomplete sources |
| `hitlChannels` | Human-in-the-loop delivery channels |
| `storage` | Session history and memory backends |
| `agentDeployers` | Deploy user agents to remote platforms |

## Harness transports

Extension harnesses wire through three transports:

| Transport | Go client |
|---|---|
| `stdio` (default) | `stdioHarnessAgent` |
| `tcp` | `ExtensionAgent` (JSON-RPC 2.0) |
| `http` | `HTTPExtensionAgent` |

ADL agents reference extension harnesses as `harness.type: ext:<extension>/<harness-id>`.

## Get started

1. [Getting started with extensions]({{ '/docs/extensions/getting-started/' | relative_url }}) — install corp-pack and run your first extension agent
2. [Manifest reference]({{ '/docs/extensions/manifest/' | relative_url }}) — complete `extension.yaml` schema
3. [Programmatic SDK]({{ '/docs/extensions/programmatic/' | relative_url }}) — Python, TypeScript, and Go packages

## Examples in the repository

| Example | Demonstrates |
|---------|--------------|
| [corp-pack](https://github.com/plmbr/nui/tree/main/dev/extension-examples/corp-pack/) | Harnesses, MCP tools, skills, rules, mentions |
| [hitl-demo](https://github.com/plmbr/nui/tree/main/dev/extension-examples/hitl-demo/) | HITL channels and REST bridge |
| [storage-demo](https://github.com/plmbr/nui/tree/main/dev/extension-examples/storage-demo/) | Custom persistence |
| [docker-deployer](https://github.com/plmbr/nui/tree/main/dev/extension-examples/docker-deployer/) | Agent deployer |
| [programmatic-echo](https://github.com/plmbr/nui/tree/main/dev/extension-examples/programmatic-echo/) | Python `NuiExtension` package |
