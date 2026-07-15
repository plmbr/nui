// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"
	"unicode"

	"loop/internal/hitl"
	"loop/internal/llm"
)

// shouldAnswerInPlainText reports whether the user expects a direct text reply
// instead of ask_user prompts or show_visualization.
func shouldAnswerInPlainText(msg string) bool {
	s := strings.ToLower(strings.TrimSpace(msg))
	if s == "" {
		return false
	}
	switch s {
	case "hi", "hello", "hey", "hmm", "thanks", "thank you", "yo":
		return true
	}
	if looksLikeCapabilityQuestion(s) || looksLikeFactualQuestion(s) || looksLikeSimpleMathQuestion(s) {
		return true
	}
	return false
}

func looksLikeCapabilityQuestion(s string) bool {
	if strings.Contains(s, "what can") && (strings.Contains(s, " do") || strings.Contains(s, " help") || strings.Contains(s, "u ")) {
		return true
	}
	phrases := []string{
		"what do you do",
		"who are you",
		"what are you",
		"how can you help",
		"what are your capabilities",
		"what can you help",
	}
	for _, phrase := range phrases {
		if strings.Contains(s, phrase) {
			return true
		}
	}
	return false
}

func looksLikeFactualQuestion(s string) bool {
	prefixes := []string{
		"what is ", "what's ", "who is ", "who's ", "where is ", "where's ",
		"when is ", "when was ", "how many ", "how much ", "tell me ",
		"capital of ", "define ", "explain ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// isInformationalUserMessage is an alias kept for callers; prefer shouldAnswerInPlainText.
func isInformationalUserMessage(msg string) bool {
	return shouldAnswerInPlainText(msg)
}

func userRequestedExplicitChoices(msg string) bool {
	s := strings.ToLower(strings.TrimSpace(msg))
	phrases := []string{
		"which do you prefer",
		"which would you prefer",
		"choose between",
		"pick one",
		"pick between",
		"ask me clarifying",
		"ask clarifying",
		"multiple choice",
		"select one",
		"option a",
		" a or b",
	}
	for _, phrase := range phrases {
		if strings.Contains(s, phrase) {
			return true
		}
	}
	return false
}

func looksLikeSimpleMathQuestion(s string) bool {
	hasDigit := false
	for _, r := range s {
		if unicode.IsDigit(r) {
			hasDigit = true
			break
		}
	}
	if !hasDigit {
		return false
	}
	for _, op := range []string{"+", "-", "*", "/", " plus ", " minus ", " times ", " divided "} {
		if strings.Contains(s, op) {
			return true
		}
	}
	if strings.HasPrefix(s, "what is ") || strings.HasPrefix(s, "what's ") {
		return true
	}
	return false
}

func filterSpuriousAskUser(calls []llm.ToolCall, userMessage, provider string) (filtered, removed []llm.ToolCall) {
	if strings.TrimSpace(provider) != "ollama" {
		return calls, nil
	}
	if userRequestedExplicitChoices(userMessage) {
		return calls, nil
	}
	if len(calls) == 0 {
		return calls, nil
	}
	filtered = make([]llm.ToolCall, 0, len(calls))
	for _, tc := range calls {
		if hitl.IsQuestionTool(tc.Function.Name) {
			removed = append(removed, tc)
			continue
		}
		filtered = append(filtered, tc)
	}
	return filtered, removed
}

func salvageAskUserText(removed []llm.ToolCall) string {
	for _, tc := range removed {
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			continue
		}
		if text := hitl.SalvageAskUserMessage(args); text != "" {
			return text
		}
	}
	return ""
}
