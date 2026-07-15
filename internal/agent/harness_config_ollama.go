// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "strings"

const ollamaToolsSystemPromptAppendix = `
## Tool calling (Ollama)

Use the native tool/function calling API for every tool invocation. Never print JSON such as {"name": "...", "parameters": {...}} in assistant text — not even for ask_user.

Answer greetings, factual questions, math, and capability questions directly in plain text. Never call **ask_user** to quiz the user, offer demos, or ask what they want next.

**show_visualization** is only when the user explicitly asks for a chart, graph, plot, table, or dashboard. Never wrap a plain text answer in HTML or call **show_visualization** for definitions, geography, or Q&A.

When a chart is explicitly requested, call **show_visualization** on **loop-viz** with complete self-contained HTML in the **html** field. Use valid JSON with properly escaped quotes inside HTML attribute values. Every script tag must be properly closed.
`

const ollamaHitlSystemPromptAppendix = `
## Human in the loop (Ollama)

Do **not** use **ask_user** on **loop-hitl** unless the user explicitly asked you to choose between options (for example: "which do you prefer, A or B?").

Never use ask_user for greetings, capability questions, demos, or follow-up quizzes. Describe capabilities in plain text instead of prompting the user to pick a demo.
`

func appendOllamaToolsSystemPrompt(systemPrompt string) string {
	return appendSystemPromptBlock(systemPrompt, ollamaToolsSystemPromptAppendix)
}

func appendOllamaHitlSystemPrompt(systemPrompt string) string {
	return appendSystemPromptBlock(systemPrompt, ollamaHitlSystemPromptAppendix)
}

func appendSystemPromptBlock(systemPrompt, block string) string {
	block = strings.TrimSpace(block)
	if block == "" {
		return systemPrompt
	}
	base := strings.TrimSpace(systemPrompt)
	if base == "" {
		return block
	}
	return base + "\n\n" + block
}
