// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import "testing"

func TestOllamaMessages_toolResultIncludesToolName(t *testing.T) {
	out := ollamaMessages([]Message{{
		Role:     RoleTool,
		ToolName: "ask_user",
		Content:  `{"answers":{"q1":"4"}}`,
	}})
	if len(out) != 1 {
		t.Fatalf("len = %d", len(out))
	}
	if out[0]["role"] != "tool" {
		t.Fatalf("role = %v", out[0]["role"])
	}
	if out[0]["tool_name"] != "ask_user" {
		t.Fatalf("tool_name = %v", out[0]["tool_name"])
	}
}

func TestOllamaMessages_assistantToolCallsIncludeType(t *testing.T) {
	out := ollamaMessages([]Message{{
		Role:    RoleAssistant,
		Content: "",
		ToolCalls: []ToolCall{{
			Type: "function",
			Function: FunctionCall{
				Name:      "show_visualization",
				Arguments: `{"html":"<p>x</p>"}`,
			},
		}},
	}})
	if len(out) != 1 {
		t.Fatalf("len = %d", len(out))
	}
	calls, ok := out[0]["tool_calls"].([]map[string]any)
	if !ok || len(calls) != 1 {
		t.Fatalf("tool_calls = %T %+v", out[0]["tool_calls"], out[0]["tool_calls"])
	}
	if calls[0]["type"] != "function" {
		t.Fatalf("type = %v", calls[0]["type"])
	}
	fn, ok := calls[0]["function"].(map[string]any)
	if !ok || fn["name"] != "show_visualization" {
		t.Fatalf("function = %v", calls[0]["function"])
	}
}
