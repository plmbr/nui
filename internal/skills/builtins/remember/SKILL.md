---
name: remember
description: Save durable facts to Loop persistent memory (~/.loop/memory/) when the user asks
---

# Remember

Use this skill when the user invokes `/remember` or explicitly asks you to remember something (e.g. "remember this", "save to memory").

Do **not** save to memory proactively in manual mode — wait for an explicit user request.

## Memory locations

- **User memory** (`~/.loop/memory/user.md`) — preferences and facts that apply across **all** agents (timezone, coding style, default branch, communication preferences).
- **Agent memory** (`~/.loop/memory/agents/<agent-id>.md`) — facts specific to **this** agent's role (review checklist, project conventions, choices made in this workflow).

## Scope rules

**Default to agent scope.** When the user asks to remember something — including a choice, decision, or preference for the current task — save to **agent** memory unless they clearly mean all agents.

Use **user** scope only when the user explicitly asks for cross-agent or personal memory, e.g.:
- "always", "for all agents", "everywhere"
- "remember this about me" / personal preference that should follow them across agents

Do not move agent-specific choices into user memory just because they sound like a preference.

## Saving memory

### API harness (loop-agent MCP)

Call **`update_memory`** on the **`loop-agent`** MCP server:

- `scope`: `"user"` or `"agent"`
- `content`: concise markdown (bullet points or short paragraphs)
- `mode`: `"append"` to add to existing memory, `"replace"` to overwrite the whole file

### CLI harness (Write/Edit tools)

Write directly to the canonical path:

- User: `~/.loop/memory/user.md`
- Agent: `~/.loop/memory/agents/<agent-id>.md`

When appending, read the existing file first and preserve prior content.

## Quality bar

- Store durable facts, not transient task state or conversation summaries.
- Keep entries concise; prefer bullets over long prose.
- Do not store secrets, API keys, or credentials.
- If memory would exceed roughly 8KB, consolidate or prune older low-value entries before appending.
- When unsure, default to **agent** scope; ask only if it is genuinely ambiguous whether the fact applies to all agents.
