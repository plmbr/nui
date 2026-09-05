---
name: create-agent
description: Create a nui ADL agent definition from the current conversation and save it to ~/.nui/agents/
---

# Create Agent

Use this skill when the user invokes `/create-agent` or asks you to turn the conversation into a reusable nui agent.

## Default behavior

When no other instructions are given, convert the conversation into a new ADL agent definition:

1. Infer a short **id** (kebab-case, e.g. `code-review-helper`) and human-readable **name**.
2. Write a **description** summarizing what the agent does.
3. Distill the conversation into a **systemPrompt** that captures the user's intent, constraints, and examples discussed.
4. If the conversation implies a recurring first message, set **defaultPrompt**; otherwise omit it.
5. Choose **harness.type** from context (default `claude-code` when unclear).
6. Omit **aiAssets**, **orchestration**, and **env** unless the conversation clearly requires them.

Save the result as `~/.nui/agents/<id>.yaml`.

## When the user adds instructions

Follow any instructions in the same message after `/create-agent` or in follow-up messages. Examples:

- change harness, model, or working directory behavior
- add skills, MCP servers, or orchestration (workflow steps, subAgents, or council)
- rename the agent or adjust the system prompt
- update an existing agent file instead of creating a new one

User instructions override the default conversion behavior when they conflict.

## ADL template

Use ADL 1.0. Minimum viable agent:

```yaml
adl: "1.0"

id: example-agent
name: Example Agent
description: One-line summary of what this agent does.
version: 0.1.0

harness:
  type: claude-code
  model: claude-sonnet-4-6

systemPrompt: |
  Instructions distilled from the conversation.
```

Optional fields when relevant: `defaultPrompt`, `promptMode`, `workingDirInput`, `env`, `aiAssets`, `orchestration`.

For multi-agent agents use `orchestration` (never top-level `steps`, `subAgents`, or `council`):

```yaml
orchestration:
  type: workflow          # or subAgents | council
  steps:                  # when type is workflow
    - name: review
      systemPrompt: ...
```

## Saving

- Target directory: `~/.nui/agents/`
- Filename: `<id>.yaml` (must match the `id` field)
- Create the directory if needed.
- If a file with the same id already exists, ask before overwriting unless the user explicitly requested an update.

### API harness (no Write tool)

When you do not have filesystem Write/Edit tools, call **`save_agent`** on the **`nui-agent`** MCP server with the full YAML in the **`content`** field. Set **`overwrite`** to true only when updating an existing agent the user asked to replace.

### CLI harness

Write the YAML file directly to `~/.nui/agents/<id>.yaml`.
- After saving, tell the user the file path and that the agent will appear under **Installed agents** in nui (they may need to refresh agent types).

## Quality bar

- Keep YAML valid and readable.
- Prefer concise system prompts over copying the entire chat verbatim.
- Preserve concrete requirements, tool usage, output format, and examples from the conversation.
- Do not invent API keys, secrets, or private paths.
