// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"

	"nui/internal/llm"
	"nui/internal/viz"
)

func userRequestedVisualization(msg string) bool {
	s := strings.ToLower(strings.TrimSpace(msg))
	keywords := []string{
		"chart", "graph", "plot", "dashboard", "visualiz", "diagram",
		"bar chart", "pie chart", "line chart", "histogram",
		"scatter plot", "show me a table", "data table",
	}
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}

func filterSpuriousVisualization(calls []llm.ToolCall, userMessage, provider string) (filtered, removed []llm.ToolCall) {
	if strings.TrimSpace(provider) != "ollama" || userRequestedVisualization(userMessage) {
		return calls, nil
	}
	if len(calls) == 0 {
		return calls, nil
	}
	filtered = make([]llm.ToolCall, 0, len(calls))
	for _, tc := range calls {
		if viz.IsVisualizationTool(tc.Function.Name) {
			removed = append(removed, tc)
			continue
		}
		filtered = append(filtered, tc)
	}
	return filtered, removed
}

func salvageVisualizationText(removed []llm.ToolCall) string {
	for _, tc := range removed {
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			continue
		}
		html, _, ok := viz.ParseInput(args)
		if !ok {
			continue
		}
		if text := strings.TrimSpace(viz.PlainTextFromHTML(html)); text != "" {
			return text
		}
	}
	return ""
}
