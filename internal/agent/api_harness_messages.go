// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"

	"nui/internal/llm"
)

// normalizeToolCallArguments ensures tool arguments are a JSON object string.
// Anthropic requires tool_use.input to be an object; empty or invalid args become "{}".
func normalizeToolCallArguments(args string) string {
	args = strings.TrimSpace(args)
	if args == "" {
		return "{}"
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(args), &obj); err != nil || obj == nil {
		return "{}"
	}
	return args
}

func normalizeAPIToolCalls(calls []llm.ToolCall) []llm.ToolCall {
	if len(calls) == 0 {
		return calls
	}
	out := make([]llm.ToolCall, len(calls))
	copy(out, calls)
	for i := range out {
		out[i].Function.Arguments = normalizeToolCallArguments(out[i].Function.Arguments)
	}
	return out
}

func normalizeAPIMessages(messages []llm.Message) []llm.Message {
	if len(messages) == 0 {
		return messages
	}
	out := make([]llm.Message, len(messages))
	copy(out, messages)
	for i := range out {
		if out[i].Role != llm.RoleAssistant || len(out[i].ToolCalls) == 0 {
			continue
		}
		out[i].ToolCalls = normalizeAPIToolCalls(out[i].ToolCalls)
	}
	return out
}
