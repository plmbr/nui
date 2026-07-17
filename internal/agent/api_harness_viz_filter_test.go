// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"

	"nui/internal/llm"
)

func TestShouldAnswerInPlainText(t *testing.T) {
	cases := map[string]bool{
		"what can u fo":                    true,
		"what can you do":                  true,
		"What is the capital of France":    true,
		"who is the president":             true,
		"show me a bar chart of sales":     false,
		"which do you prefer, pie or bar":  false,
	}
	for msg, want := range cases {
		if got := shouldAnswerInPlainText(msg); got != want {
			t.Errorf("shouldAnswerInPlainText(%q) = %v, want %v", msg, got, want)
		}
	}
}

func TestFilterSpuriousVisualization_ollamaFactualQuestion(t *testing.T) {
	calls := []llm.ToolCall{
		{
			Function: llm.FunctionCall{
				Name: "show_visualization",
				Arguments: `{"html":"<!DOCTYPE html><html><body><p>France's capital is Paris.</p></body></html>"}`,
			},
		},
	}
	filtered, removed := filterSpuriousVisualization(calls, "What is the capital of France", "ollama")
	if len(filtered) != 0 {
		t.Fatalf("filtered = %#v", filtered)
	}
	if len(removed) != 1 {
		t.Fatalf("removed = %#v", removed)
	}
	text := salvageVisualizationText(removed)
	if text != "France's capital is Paris." {
		t.Fatalf("salvage = %q", text)
	}
}

func TestFilterSpuriousVisualization_keepsExplicitChartRequest(t *testing.T) {
	calls := []llm.ToolCall{
		{Function: llm.FunctionCall{Name: "show_visualization", Arguments: `{"html":"<canvas></canvas>"}`}},
	}
	filtered, removed := filterSpuriousVisualization(calls, "show me a bar chart", "ollama")
	if len(filtered) != 1 || len(removed) != 0 {
		t.Fatalf("filtered=%#v removed=%#v", filtered, removed)
	}
}
