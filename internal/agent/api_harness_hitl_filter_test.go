// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"

	"nui/internal/llm"
)

func TestIsInformationalUserMessage(t *testing.T) {
	cases := map[string]bool{
		"what can u do":     true,
		"what can u fo":     true,
		"What can you do?":  true,
		"who are you":       true,
		"what is 2+2":       true,
		"what is the capital of France": true,
		"hello":             true,
		"hi":                true,
		"build me a chart":  false,
		"which color do you prefer for the logo?": false,
	}
	for msg, want := range cases {
		if got := isInformationalUserMessage(msg); got != want {
			t.Errorf("isInformationalUserMessage(%q) = %v, want %v", msg, got, want)
		}
	}
}

func TestFilterSpuriousAskUser_ollamaInformational(t *testing.T) {
	calls := []llm.ToolCall{
		{Function: llm.FunctionCall{Name: "ask_user", Arguments: `{"message":"Pick one"}`}},
		{Function: llm.FunctionCall{Name: "show_visualization", Arguments: `{"html":"<p>x</p>"}`}},
	}
	filtered, removed := filterSpuriousAskUser(calls, "what can you do", "ollama")
	if len(filtered) != 1 || filtered[0].Function.Name != "show_visualization" {
		t.Fatalf("filtered = %#v", filtered)
	}
	if len(removed) != 1 || removed[0].Function.Name != "ask_user" {
		t.Fatalf("removed = %#v", removed)
	}
}

func TestFilterSpuriousAskUser_nonOllamaKeepsAskUser(t *testing.T) {
	calls := []llm.ToolCall{
		{Function: llm.FunctionCall{Name: "ask_user", Arguments: `{"message":"Pick one"}`}},
	}
	filtered, removed := filterSpuriousAskUser(calls, "what can you do", "openai")
	if len(filtered) != 1 {
		t.Fatalf("filtered = %#v", filtered)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %#v", removed)
	}
}

func TestSalvageAskUserText(t *testing.T) {
	removed := []llm.ToolCall{
		{Function: llm.FunctionCall{
			Name:      "ask_user",
			Arguments: `{"message":"I can help with many tasks. Which would you like?"}`,
		}},
	}
	got := salvageAskUserText(removed)
	if got != "I can help with many tasks. Which would you like?" {
		t.Fatalf("salvage = %q", got)
	}
}
