// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"

	anyllm "github.com/mozilla-ai/any-llm-go"
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

func normalizeAPIToolCalls(calls []anyllm.ToolCall) []anyllm.ToolCall {
	if len(calls) == 0 {
		return calls
	}
	out := make([]anyllm.ToolCall, len(calls))
	copy(out, calls)
	for i := range out {
		out[i].Function.Arguments = normalizeToolCallArguments(out[i].Function.Arguments)
	}
	return out
}

func normalizeAPIMessages(messages []anyllm.Message) []anyllm.Message {
	if len(messages) == 0 {
		return messages
	}
	out := make([]anyllm.Message, len(messages))
	copy(out, messages)
	for i := range out {
		if out[i].Role != anyllm.RoleAssistant || len(out[i].ToolCalls) == 0 {
			continue
		}
		out[i].ToolCalls = normalizeAPIToolCalls(out[i].ToolCalls)
	}
	return out
}
