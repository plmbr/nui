// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"encoding/json"
	"testing"
)

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
			ID:    "call_abc",
			Type:  "function",
			Index: 0,
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
	if calls[0]["id"] != "call_abc" {
		t.Fatalf("id = %v", calls[0]["id"])
	}
	fn, ok := calls[0]["function"].(map[string]any)
	if !ok || fn["name"] != "show_visualization" {
		t.Fatalf("function = %v", calls[0]["function"])
	}
}

func TestOllamaToolCalls_preservesIDAndIndex(t *testing.T) {
	rawJSON := `[
		{"id":"call_aaa","function":{"index":0,"name":"list_skills","arguments":{}}},
		{"id":"call_bbb","function":{"index":1,"name":"bash","arguments":{"command":"echo hi"}}}
	]`
	var raw []ollamaToolCall
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		t.Fatal(err)
	}
	got := ollamaToolCalls(raw)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].ID != "call_aaa" || got[0].Index != 0 || got[0].Function.Name != "list_skills" {
		t.Fatalf("first = %+v", got[0])
	}
	if got[1].ID != "call_bbb" || got[1].Index != 1 || got[1].Function.Arguments != `{"command":"echo hi"}` {
		t.Fatalf("second = %+v", got[1])
	}
}

func TestOllamaArgumentsJSON(t *testing.T) {
	if got := ollamaArgumentsJSON(nil); got != "{}" {
		t.Fatalf("nil = %q", got)
	}
	if got := ollamaArgumentsJSON(json.RawMessage(`{"a":1}`)); got != `{"a":1}` {
		t.Fatalf("object = %q", got)
	}
	if got := ollamaArgumentsJSON(json.RawMessage(`"{\"a\":1}"`)); got != `{"a":1}` {
		t.Fatalf("stringified = %q", got)
	}
}

func TestOllamaStreamState_finishReasonToolCalls(t *testing.T) {
	state := &ollamaStreamState{id: "x", created: 1}
	var raw ollamaChatResponse
	if err := json.Unmarshal([]byte(`{
		"message":{"role":"assistant","content":"","tool_calls":[
			{"id":"call_1","function":{"index":0,"name":"bash","arguments":{"command":"ls"}}}
		]}
	}`), &raw); err != nil {
		t.Fatal(err)
	}
	chunk1 := state.handle(&raw)
	if len(chunk1.Choices[0].Delta.ToolCalls) != 1 {
		t.Fatalf("chunk1 tools = %+v", chunk1.Choices[0].Delta.ToolCalls)
	}
	if chunk1.Choices[0].Delta.ToolCalls[0].ID != "call_1" {
		t.Fatalf("id = %q", chunk1.Choices[0].Delta.ToolCalls[0].ID)
	}
	chunk2 := state.handle(&ollamaChatResponse{Done: true, DoneReason: "stop"})
	if chunk2.Choices[0].FinishReason != FinishReasonToolCalls {
		t.Fatalf("finish = %q, want tool_calls", chunk2.Choices[0].FinishReason)
	}
}
