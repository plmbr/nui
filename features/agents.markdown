---
layout: page
title: Built-in agents
subtitle: Master launcher, CLI subprocesses, and in-process API agents — all selectable from the New Session panel.
permalink: /features/agents/
---

nui ships with built-in agents for the home launcher, common coding-agent CLIs, and API providers. Each session resolves to an ADL definition; single-step agents dispatch directly to the harness.

## Master agent

| Agent | ADL id | Role |
|---|---|---|
| nui | `nui` | Home launcher / router. Uses `nui-orchestrator` MCP tools (`list_agents`, `launch_session`) to open specialist sessions, and can create agents via the create-agent skill. Legacy id: `nui-orchestrator`. |

The home screen calls `POST /api/orchestrate`. The master's harness comes from Settings → **Default harness** (defaults to Anthropic API).

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

| ADL id | Name | Provider |
|---|---|---|
| `anthropic` | Claude API | Claude models via the Anthropic API |
| `openai` | OpenAI | GPT models via the OpenAI API |
| `gemini` | Gemini | Google Gemini via the Gemini API |
| `openrouter` | OpenRouter | Multi-model routing via OpenRouter |
| `ollama` | Ollama | Local models via Ollama |

See the [harness design doc](https://github.com/plmbr/nui/blob/main/dev/harness-design.md) for API harness configuration and environment variables.

## Sandbox variants

Built-in CLI agents support three sandbox modes:

- **`none`** — runs unsandboxed on the host
- **`bubblewrap`** — Linux-only namespace isolation with bind mounts for agent config dirs
- **`docker`** — runs in a managed Docker container (`nui-claude-code:latest`, `nui-pi:latest`, etc.)

Set `harness.sandbox` in the ADL definition or pick the variant when creating a session.
