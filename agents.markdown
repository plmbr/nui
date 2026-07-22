---
layout: page
title: Agents
subtitle: Built-in CLI and API agents — pick one per session, or install custom ADL definitions.
permalink: /agents/
---

## CLI agents

These require the corresponding binary on your `PATH`:

| Agent (ADL id) | CLI | Description |
|---|---|---|
| `claude-code` | `claude` | Anthropic's Claude Code CLI |
| `pi` | `pi` | pi agent CLI |
| `codex` | `codex` | OpenAI Codex CLI |
| `opencode` | `opencode` | OpenCode CLI |

## API agents

In-process LLM calls — no CLI required. Select under **Built-in → API** in the New Session panel:

| Name | API key environment variable |
|---|---|
| Anthropic | `ANTHROPIC_API_KEY` (or `ANTHROPIC_AUTH_TOKEN`) |
| OpenAI | `OPENAI_API_KEY` |
| Gemini | `GEMINI_API_KEY` or `GOOGLE_API_KEY` |
| OpenRouter | `OPENROUTER_API_KEY` |
| Ollama | none (local; optional `OLLAMA_HOST`) |

## Custom agents

Install your own agent definitions (ADL YAML) to `~/.nui/agents/`:

```bash
nui agent add ./my-agent.yaml
```

Custom agents appear under **Installed agents** in the New Session panel. They can run in Docker, dev containers, remote servers, or sandboxes.

See [ADL agents](/features/adl/) for the schema and [harness examples](https://github.com/plmbr/nui/tree/main/dev/harness-examples) for templates.

## Harness types

| Harness | Lifecycle |
|---|---|
| `claude-code`, `pi`, `codex`, `opencode` | Go-managed subprocesses on the host or in Docker |
| `docker` | nui runs a container, health-checks, and tears it down on delete |
| `remote` | Connect to a pre-running HTTP/SSE agent at `host:port` |
| `ext:<extension>/<harness-id>` | Extension-contributed stdio, TCP, or HTTP harnesses |

Sandbox options per harness: `none`, `bubblewrap` (Linux only), or `docker`.
