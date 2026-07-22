---
layout: page
title: Built-in agents
subtitle: CLI subprocesses and in-process API agents — all selectable from the New Session dialog.
permalink: /features/agents/
---

nui ships with built-in agents for the most common coding-agent CLIs and API providers. Each session resolves to an ADL definition; single-step agents dispatch directly to the harness.

## CLI agents

| Agent | Harness | CLI |
|---|---|---|
| Claude Code | `claude-code` | `claude -p … --output-format stream-json` |
| pi | `pi` | `pi --mode rpc` (JSON-RPC over stdin/stdout) |
| codex | `codex` | `codex exec … --json` |
| opencode | `opencode` | `opencode serve` + `opencode run --attach` |

Persistent sessions are maintained per harness — Claude Code, pi, codex, and opencode each keep their own session state across turns.

## API agents

API agents run in-process without a CLI binary:

| Agent | Provider |
|---|---|
| Anthropic | Claude models via the Anthropic API |
| OpenAI | GPT models via the OpenAI API |
| Gemini | Google Gemini via the Gemini API |
| OpenRouter | Multi-model routing via OpenRouter |
| Ollama | Local models via Ollama |

See the [harness design doc](https://github.com/plmbr/nui/blob/main/dev/harness-design.md) for API harness configuration and environment variables.

## Sandbox variants

Built-in CLI agents support three sandbox modes:

- **`none`** — runs unsandboxed on the host
- **`bubblewrap`** — Linux-only namespace isolation with bind mounts for agent config dirs
- **`docker`** — runs in a managed Docker container (`nui-claude-code:latest`, `nui-pi:latest`, etc.)

Set `harness.sandbox` in the ADL definition or pick the variant when creating a session.
