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

ADL agents can include an `evals:` list for automated testing. Define cases in the UI or YAML, then run them from the agent editor or the CLI.

### Define and run evals in the UI

1. Open **Customize** (sidebar gear icon) → **Agents** tab.
2. Select an installed agent or create a new one.
3. Scroll to the **Evals** section in the form editor.

Each eval case has:

| Field | Description |
|---|---|
| **Name** | Unique case id (used by `--case` on the CLI) |
| **On** | Enable or disable the case without deleting it |
| **Prompt** | User message sent to the agent |
| **Expected text** | Substring the response should contain (maps to a `contains` grader) |

Click **Add eval** to create another case. Expand **Advanced** for more options:

| Field | Description |
|---|---|
| **Grader** | `Contains`, `Exact match`, `Regex`, `LLM judge`, or `Manual (none)` |
| **Expected value / Criteria** | Required for `exact`, `regex`, and `llm` graders |
| **Description** | Optional note about what the case verifies |
| **Timeout** | Per-case timeout in seconds (default `120`) |
| **Tags** | Labels for organization (CLI filtering only for now) |
| **Working dir override** | Optional path for this case |

**Run from the UI:**

- Click **Run** on a single case to execute it immediately.
- Click **Run evals** in the editor toolbar to run all enabled cases. nui saves unsaved changes first, then shows pass/fail results in a dialog. You can optionally set a working directory for the run.

**Form vs YAML mode:** Use the **Form** / **YAML** toggle at the top of the editor. Single-turn evals are easiest in Form mode. **Conversation evals** (multi-turn `messages:`) are read-only in the form — switch to YAML mode to edit message turns. Assistant turns are not injected into the session; only user messages are sent at run time.

For CI and scripting, the same cases run via CLI: `nui agent eval run -a <id>`. See [Headless & scheduled runs]({{ '/features/headless/#agent-evaluation' | relative_url }}).

### YAML schema

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
