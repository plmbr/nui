---
name: ask-user
description: Ask the human structured questions via nui HITL instead of plain chat text
---

# Ask User (nui HITL)

When you need **any** input, clarification, preference, or approval from the human, use the **`nui-hitl`** MCP tool **`ask_user`**. Do **not** ask those questions only in assistant text — the human answers through the nui UI prompt card.

## When to use `ask_user`

- The user asks you to ask clarifying questions before continuing
- You need a choice between options (cuisine, scope, constraints, etc.)
- You would otherwise write "Which do you prefer?" or list questions in the chat
- You need structured answers before taking a consequential action

## How to call it

Use the MCP tool **`ask_user`** on server **`nui-hitl`** with a `questions` array. Each question object can include:

- `question` — the prompt text (required)
- `header` — short label shown in the UI (optional)
- `options` — array of `{ "label": "...", "description": "..." }` for multiple choice (optional)

Example:

```json
{
  "title": "Recipe preferences",
  "questions": [
    {
      "header": "Region",
      "question": "Which South American country or region should the recipe focus on?",
      "options": [
        { "label": "Peru", "description": "Andean / coastal Peruvian" },
        { "label": "Argentina", "description": "Grilled meats, empanadas" },
        { "label": "Brazil", "description": "Feijoada, moqueca, etc." }
      ]
    },
    {
      "header": "Diet",
      "question": "Any dietary restrictions?",
      "options": [
        { "label": "None" },
        { "label": "Vegetarian" },
        { "label": "Gluten-free" }
      ]
    }
  ]
}
```

Wait for the tool result (answers from the human), then continue with the task using those answers.

## Tool approval

For yes/no gates before risky actions, use **`request_approval`** on **`nui-hitl`** instead of `ask_user`.

## Do not

- List clarification questions in plain assistant text when `ask_user` is available
- Proceed with assumptions when the user explicitly asked you to clarify first
