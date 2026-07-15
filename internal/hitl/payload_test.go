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

func TestNormalizePayloadPromptField(t *testing.T) {
	payload := NormalizePayload(map[string]any{
		"title": "Chart type",
		"questions": []any{
			map[string]any{
				"id":      "chart_type",
				"prompt":  "What kind of chart would you like?",
				"options": []any{map[string]any{"id": "bar", "label": "Bar chart"}},
			},
		},
	})
	questions, ok := payload["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("questions = %#v", payload["questions"])
	}
	q, ok := questions[0].(map[string]any)
	if !ok || q["question"] != "What kind of chart would you like?" {
		t.Fatalf("question = %#v", questions[0])
	}
	opts, ok := q["options"].([]any)
	if !ok || len(opts) != 1 {
		t.Fatalf("options = %#v", q["options"])
	}
	first, ok := opts[0].(map[string]any)
	if !ok || first["label"] != "Bar chart" {
		t.Fatalf("first option = %#v", opts[0])
	}
}

func TestNormalizePayloadSynthesizeFromMessage(t *testing.T) {
	payload := NormalizePayload(map[string]any{
		"message": "What kind of chart would you like?",
	})
	questions, ok := payload["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("questions = %#v", payload["questions"])
	}
	q, ok := questions[0].(map[string]any)
	if !ok || q["question"] != "What kind of chart would you like?" {
		t.Fatalf("question = %#v", questions[0])
	}
}

func TestNormalizePayloadTopLevelQuestion(t *testing.T) {
	payload := NormalizePayload(map[string]any{
		"question": "Pick a color",
	})
	questions, ok := payload["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("questions = %#v", payload["questions"])
	}
}

func TestNormalizePayloadHeaderOnlyQuestion(t *testing.T) {
	payload := NormalizePayload(map[string]any{
		"questions": []any{
			map[string]any{
				"header": "Choose an action",
				"options": []any{
					map[string]any{"label": "Answer a question"},
				},
			},
		},
	})
	questions, ok := payload["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("questions = %#v", payload["questions"])
	}
	q, ok := questions[0].(map[string]any)
	if !ok || q["question"] != "Choose an action" {
		t.Fatalf("question = %#v", questions[0])
	}
}

func TestNormalizePayloadQuestionsJSONInMessage(t *testing.T) {
	payload := NormalizePayload(map[string]any{
		"message": `I can help. Which would you like? [{"header":"Choose an action","options":[{"label":"A"}]}]`,
	})
	if payload["message"] != "I can help. Which would you like?" {
		t.Fatalf("message = %#v", payload["message"])
	}
	questions, ok := payload["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("questions = %#v", payload["questions"])
	}
}

func TestSalvageAskUserMessage(t *testing.T) {
	got := SalvageAskUserMessage(map[string]any{
		"message": "I can help with tasks.",
	})
	if got != "I can help with tasks." {
		t.Fatalf("salvage = %q", got)
	}
}
