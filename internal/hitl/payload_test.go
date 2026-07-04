// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import (
	"testing"
)

func TestNormalizePayloadStringOptions(t *testing.T) {
	payload := NormalizePayload(map[string]any{
		"title": "Choose a color",
		"questions": []any{
			map[string]any{
				"question": "Which color?",
				"options":  []any{"Red", "Blue", "Green"},
			},
		},
	})
	questions, ok := payload["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("questions = %#v", payload["questions"])
	}
	q, ok := questions[0].(map[string]any)
	if !ok {
		t.Fatalf("question = %#v", questions[0])
	}
	opts, ok := q["options"].([]any)
	if !ok || len(opts) != 3 {
		t.Fatalf("options = %#v", q["options"])
	}
	first, ok := opts[0].(map[string]any)
	if !ok || first["label"] != "Red" {
		t.Fatalf("first option = %#v", opts[0])
	}
}

func TestNormalizePayloadAskUserQuestionHookShape(t *testing.T) {
	payload := NormalizePayload(map[string]any{
		"tool_name": "AskUserQuestion",
		"tool_input": map[string]any{
			"questions": []any{
				map[string]any{
					"question": "Where?",
					"options":  []any{"Here", "There"},
				},
			},
		},
	})
	questions, ok := payload["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("questions = %#v", payload["questions"])
	}
}
