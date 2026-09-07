// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"encoding/json"
	"testing"
)

func TestGeminiContents_preservesThoughtSignature(t *testing.T) {
	contents, _ := geminiContents([]Message{
		{Role: RoleUser, Content: "run pwd"},
		{
			Role: RoleAssistant,
			ToolCalls: []ToolCall{{
				ID:               "call_gemini_0",
				ThoughtSignature: "sig-abc",
				Function: FunctionCall{
					Name:      "nui-bash__bash",
					Arguments: `{"command":"pwd"}`,
				},
			}},
		},
		{
			Role:       RoleTool,
			ToolCallID: "call_gemini_0",
			ToolName:   "nui-bash__bash",
			Content:    `{"cwd":"/tmp","exitCode":0}`,
		},
	})
	if len(contents) != 3 {
		t.Fatalf("contents len = %d", len(contents))
	}
	modelParts, ok := contents[1]["parts"].([]map[string]any)
	if !ok || len(modelParts) != 1 {
		t.Fatalf("model parts = %#v", contents[1]["parts"])
	}
	if modelParts[0]["thoughtSignature"] != "sig-abc" {
		t.Fatalf("thoughtSignature = %#v", modelParts[0]["thoughtSignature"])
	}
	fc, ok := modelParts[0]["functionCall"].(map[string]any)
	if !ok || fc["name"] != "nui-bash__bash" {
		t.Fatalf("functionCall = %#v", modelParts[0]["functionCall"])
	}

	frParts, ok := contents[2]["parts"].([]map[string]any)
	if !ok || len(frParts) != 1 {
		t.Fatalf("tool parts = %#v", contents[2]["parts"])
	}
	fr, ok := frParts[0]["functionResponse"].(map[string]any)
	if !ok || fr["name"] != "nui-bash__bash" {
		t.Fatalf("functionResponse = %#v", frParts[0]["functionResponse"])
	}
}

func TestGeminiContents_injectsSkipSignatureWhenMissing(t *testing.T) {
	contents, _ := geminiContents([]Message{
		{
			Role: RoleAssistant,
			ToolCalls: []ToolCall{{
				Function: FunctionCall{Name: "nui-fs__read", Arguments: `{"path":"a.txt"}`},
			}},
		},
	})
	parts := contents[0]["parts"].([]map[string]any)
	if parts[0]["thoughtSignature"] != geminiSkipThoughtSignature {
		t.Fatalf("thoughtSignature = %#v", parts[0]["thoughtSignature"])
	}
}

func TestGeminiToCompletion_capturesThoughtSignature(t *testing.T) {
	const payload = `{
		"candidates": [{
			"content": {
				"role": "model",
				"parts": [{
					"functionCall": {"name": "nui-bash__bash", "args": {"command": "pwd"}},
					"thoughtSignature": "sig-1"
				}]
			},
			"finishReason": "STOP"
		}]
	}`
	var raw geminiGenerateResponse
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		t.Fatal(err)
	}
	comp := geminiToCompletion(&raw, "gemini-3.5-flash")
	if len(comp.Choices) != 1 || len(comp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("completion = %#v", comp)
	}
	tc := comp.Choices[0].Message.ToolCalls[0]
	if tc.ThoughtSignature != "sig-1" {
		t.Fatalf("sig = %q", tc.ThoughtSignature)
	}
	if tc.Function.Name != "nui-bash__bash" {
		t.Fatalf("name = %q", tc.Function.Name)
	}
	if comp.Choices[0].FinishReason != FinishReasonToolCalls {
		t.Fatalf("finish = %q", comp.Choices[0].FinishReason)
	}
}

func TestGeminiToolResponse(t *testing.T) {
	got := geminiToolResponse(`{"exitCode":0}`)
	v, ok := got["exitCode"].(float64)
	if !ok || v != 0 {
		t.Fatalf("got = %#v", got)
	}
	got = geminiToolResponse("plain text")
	if got["result"] != "plain text" {
		t.Fatalf("plain = %#v", got)
	}
}

func TestSanitizeGeminiSchema_stripsAdditionalProperties(t *testing.T) {
	schema := map[string]any{
		"type":                 "object",
		"$schema":              "http://json-schema.org/draft-07/schema#",
		"additionalProperties": false,
		"properties": map[string]any{
			"command": map[string]any{
				"type":                 "string",
				"description":          "Shell command",
				"additionalProperties": false,
			},
			"value": map[string]any{
				"anyOf": []any{
					map[string]any{
						"type":                 "string",
						"additionalProperties": false,
					},
					map[string]any{"type": "null"},
				},
				"additionalProperties": false,
			},
		},
		"required": []any{"command"},
	}
	got := sanitizeGeminiSchema(schema)
	if _, ok := got["additionalProperties"]; ok {
		t.Fatalf("additionalProperties survived: %#v", got)
	}
	if _, ok := got["$schema"]; ok {
		t.Fatal("$schema survived")
	}
	props := got["properties"].(map[string]any)
	cmd := props["command"].(map[string]any)
	if _, ok := cmd["additionalProperties"]; ok {
		t.Fatalf("nested additionalProperties survived: %#v", cmd)
	}
	value := props["value"].(map[string]any)
	anyOf := value["anyOf"].([]any)
	first := anyOf[0].(map[string]any)
	if _, ok := first["additionalProperties"]; ok {
		t.Fatalf("anyOf additionalProperties survived: %#v", first)
	}
}

func TestSanitizeGeminiSchema_nullTypeUnion(t *testing.T) {
	got := sanitizeGeminiSchema(map[string]any{
		"type": []any{"string", "null"},
	})
	if got["type"] != "string" {
		t.Fatalf("type = %#v", got["type"])
	}
	if got["nullable"] != true {
		t.Fatalf("nullable = %#v", got["nullable"])
	}
}

func TestGeminiBuildRequest_sanitizesToolParameters(t *testing.T) {
	p := &geminiProvider{apiKey: "k", baseURL: geminiDefaultBaseURL}
	req := p.buildRequest(CompletionParams{
		Model: "gemini-3.5-flash",
		Messages: []Message{
			{Role: RoleUser, Content: "hi"},
		},
		Tools: []Tool{{
			Type: "function",
			Function: Function{
				Name:        "nui-bash__bash",
				Description: "Run shell",
				Parameters: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"command": map[string]any{"type": "string"},
					},
				},
			},
		}},
	})
	tools := req["tools"].([]map[string]any)
	decls := tools[0]["functionDeclarations"].([]map[string]any)
	params := decls[0]["parameters"].(map[string]any)
	if _, ok := params["additionalProperties"]; ok {
		t.Fatalf("parameters = %#v", params)
	}
}
