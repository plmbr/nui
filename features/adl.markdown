---
layout: page
title: ADL agents
subtitle: Define custom agents in YAML with a harness, system prompt, and optional sandbox.
permalink: /features/adl/
---

The Agent Definition Language (ADL) lets you define custom agents as YAML files in `~/.nui/agents/`. Pick a harness, set a `systemPrompt`, and optionally choose a sandbox mode. Multi-step workflows are supported too when you need them.

## Install a custom agent

```bash
nui agent add ./my-agent.yaml
```

Custom agents appear under **Installed agents** in the New Session dialog.

## Example

```yaml
adl: "1.0"
id: review-agent
name: Review Agent
description: Review a codebase and produce a report.
harness:
  type: claude-code
  sandbox: none
systemPrompt: |
  You are a code reviewer. Read the working directory, list issues by
  severity, and suggest concrete fixes. Be concise.
```

## Eval test cases

ADL agents can include an `evals:` list for automated testing. Run cases with `nui agent eval run -a <id>` against a running server.

```yaml
evals:
  - name: smoke
    input: List three code review best practices.
    expect:
      type: contains
      value: review
  - name: multi-turn
    messages:
      - role: user
        content: Remember the project uses Go.
      - role: assistant
        content: Got it — Go project noted.
      - role: user
        content: What language is this project?
    expect:
      type: contains
      value: Go
```

Grader types: `contains`, `exact`, `regex`, `llm` (natural-language criteria), and `none`. See the [CLI reference]({{ '/cli/#agent-evaluation-schema' | relative_url }}) for the full schema.

## Harness types

ADL agents can use any harness type:

| Type | Description |
|---|---|
| `claude-code`, `pi`, `codex`, `opencode` | Built-in CLI harnesses |
| `api` | In-process LLM API (Anthropic, OpenAI, Gemini, etc.) |
| `docker` | HTTP/SSE agent in a managed container |
| `devcontainer` | nui-managed dev container |
| `remote` | Pre-running HTTP/SSE agent at `host:port` |
| `ext:<extension>/<harness-id>` | Extension-contributed harness |

## Further reading

- [CLI reference — Agent evaluation]({{ '/cli/#agent-evaluation-schema' | relative_url }}) — eval schema and flags
- [ADL design doc](https://github.com/plmbr/nui/blob/main/dev/adl/design.md) — full schema reference
- [ADL examples](https://github.com/plmbr/nui/tree/main/dev/adl/examples/) — sample definitions
- [Harness examples](https://github.com/plmbr/nui/tree/main/dev/harness-examples/) — custom harness implementations
