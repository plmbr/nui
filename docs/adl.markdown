---
layout: docs
title: ADL — Agent Definition Language
subtitle: Define custom agents, workflows, MCP assets, and HITL in YAML.
permalink: /docs/adl/
---

ADL (Agent Definition Language) is nui's YAML schema for describing agents. Definitions live in `~/.nui/agents/*.yaml` or are contributed by extensions. The UI form editor and headless CLI both use the same schema.

## Minimal agent

```yaml
id: my-agent
name: My Agent
description: A simple Claude Code agent

harness:
  type: claude-code
  model: claude-sonnet-4-6

systemPrompt: |
  You are a helpful coding assistant.
```

## Harness types

| `harness.type` | Description |
|----------------|-------------|
| `claude-code` | Anthropic Claude Code CLI |
| `pi` | Pi agent (`pi --mode rpc`) |
| `codex` | OpenAI Codex CLI |
| `opencode` | OpenCode CLI |
| `api` | In-process LLM (Anthropic, OpenAI, Gemini, OpenRouter, Ollama) |
| `docker` | Custom HTTP/SSE harness in a Docker container |
| `remote` | Remote HTTP/SSE harness (no lifecycle management) |
| `devcontainer` | Builtin CLI inside a nui-provisioned dev container |
| `ext:<extension>/<harness-id>` | Extension harness (stdio, TCP, or HTTP) |

### API harness example

```yaml
id: api-claude
name: API Claude
harness:
  type: api
  provider: anthropic
  model: claude-sonnet-4-20250514
```

Set `ANTHROPIC_API_KEY` (and optionally `ANTHROPIC_BASE_URL`) in the environment.

### Extension harness example

```yaml
id: echo-bot
name: Echo Bot
harness:
  type: ext:corp-pack/echo
```

## aiAssets — MCP, skills, rules, mentions

Reference extension contributions with `ref:`:

```yaml
aiAssets:
  mcpServers:
    - name: corp-tools
      ref: ext:corp-pack/corp-tools
  skills:
    - name: deploy-checklist
      ref: ext:corp-pack/deploy-checklist
  rules:
    - name: corp-guidelines
      ref: ext:corp-pack/corp-guidelines
  mentionProviders:
    - ref: ext:corp-pack/corp-refs
```

Inline MCP servers (same schema as extensions):

```yaml
aiAssets:
  mcpServers:
    - name: local-tools
      command: ["npx", "-y", "my-mcp-server"]
      env:
        API_KEY: ${localEnv:MY_API_KEY}
```

## Multi-step workflows

```yaml
id: review-and-fix
name: Review and Fix
steps:
  - id: review
    harness:
      type: claude-code
    systemPrompt: Review the code and list issues.
    outputs:
      - name: issues
        description: List of issues found

  - id: fix
    dependsOn: [review]
    harness:
      type: claude-code
    inputs:
      - from: review.issues
    systemPrompt: Fix the issues from the review.
```

## HITL (human-in-the-loop)

```yaml
hitl:
  mode: interactive
  channels:
    - nui-ui
    - ext:hitl-demo/demo-slack
```

When `mode: interactive`, builtin harnesses receive an injected `nui-hitl` MCP server. Extension harnesses can call `ask_user()` via the Python SDK or the REST API. See [HITL]({{ '/docs/extensions/hitl/' | relative_url }}).

## Sandbox

```yaml
harness:
  type: claude-code
  sandbox: none          # default — runs on host
  # sandbox: bubblewrap  # Linux only
  # sandbox: docker      # nui-managed container
```

## Evals

```yaml
evals:
  - id: smoke
    input: "Say hello"
    graders:
      - type: contains
        value: hello
```

Run from CLI: `nui agent eval my-agent` or `POST /api/agents/:id/evals/run`.

## Agent IDs from extensions

Extension-contributed agents are namespaced: `ext:<extension>/<agent-id>`. They appear in `GET /api/agent-types` alongside builtins and user agents.

## Further reading

- [ADL design doc](https://github.com/plmbr/nui/blob/main/dev/adl/design.md) — full schema in the repository
- [ADL examples](https://github.com/plmbr/nui/tree/main/dev/adl/examples/) — workflow, API, docker, and more
- [Harness protocols]({{ '/docs/harness-protocols/' | relative_url }}) — wire formats for custom harnesses
- [Extension API]({{ '/docs/extensions/' | relative_url }}) — contributing agents via extensions
