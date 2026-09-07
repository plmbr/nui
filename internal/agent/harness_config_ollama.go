// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "strings"

const ollamaToolsSystemPromptAppendix = `
## Tool calling (Ollama)

Use the native tool/function calling API for tool invocations. Do not print tool-call JSON in assistant text.

Prefer answering directly when no tool is needed. Use tools when they help fulfill the user's request.

Call **show_visualization** on **nui-viz** only for charts, graphs, plots, tables, or dashboards the user asked for. Pass complete self-contained HTML in **html** with valid JSON escaping and closed script tags.
`

const ollamaHitlSystemPromptAppendix = `
## Human in the loop (Ollama)

Use **ask_user** on **nui-hitl** when you need the user to choose among options or provide input the tools cannot supply. Prefer a normal text reply when you can answer without blocking on a prompt card.
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
