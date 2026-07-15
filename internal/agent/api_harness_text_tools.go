// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"

	"loop/internal/llm"
	"github.com/google/uuid"
	"loop/internal/viz"
)

func toolNamesFromLLM(tools []llm.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Function.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func toolNameAllowed(name string, available []string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, candidate := range available {
		if candidate == name || viz.BareToolName(candidate) == viz.BareToolName(name) {
			return true
		}
	}
	return false
}

// extractTextToolCalls recovers tool calls that weaker models print as JSON text instead of
// using native tool/function calling (common with Ollama models).
func extractTextToolCalls(content string, availableTools []string) (cleaned string, calls []llm.ToolCall) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || len(availableTools) == 0 {
		return content, nil
	}

	for _, candidate := range textToolCandidates(trimmed) {
		if call, ok := parseTextToolCallJSON(candidate, availableTools); ok {
			return stripToolCallText(content, candidate), []llm.ToolCall{call}
		}
	}

	if call, ok := extractVisualizationTextToolCall(trimmed, availableTools); ok {
		return stripToolCallText(content, trimmed), []llm.ToolCall{call}
	}

	return content, nil
}

func textToolCandidates(content string) []string {
	out := []string{content}
	if fenced := stripMarkdownCodeFence(content); fenced != content {
		out = append(out, fenced)
	}
	return out
}

func stripMarkdownCodeFence(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "```") {
		return content
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 {
		return content
	}
	body := strings.Join(lines[1:len(lines)-1], "\n")
	return strings.TrimSpace(body)
}

func stripToolCallText(original, matched string) string {
	cleaned := strings.Replace(original, matched, "", 1)
	return strings.TrimSpace(cleaned)
}

func parseTextToolCallJSON(content string, availableTools []string) (llm.ToolCall, bool) {
	var envelope struct {
		Name       string         `json:"name"`
		Parameters map[string]any `json:"parameters"`
		Arguments  map[string]any `json:"arguments"`
		Input      map[string]any `json:"input"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err != nil {
		return llm.ToolCall{}, false
	}
	name := strings.TrimSpace(envelope.Name)
	if !toolNameAllowed(name, availableTools) {
		return llm.ToolCall{}, false
	}
	args := envelope.Parameters
	if len(args) == 0 {
		args = envelope.Arguments
	}
	if len(args) == 0 {
		args = envelope.Input
	}
	if len(args) == 0 {
		return llm.ToolCall{}, false
	}
	return newTextToolCall(name, args), true
}

func extractVisualizationTextToolCall(content string, availableTools []string) (llm.ToolCall, bool) {
	if !toolNameAllowed(viz.ToolName, availableTools) {
		return llm.ToolCall{}, false
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, viz.ToolName) {
		return llm.ToolCall{}, false
	}
	html := htmlFromToolJSON(content)
	if html == "" {
		html = extractHTMLFragment(content)
	}
	html = strings.TrimSpace(html)
	if html == "" {
		return llm.ToolCall{}, false
	}
	html = viz.PrepareHTML(html)
	return newTextToolCall(viz.ToolName, map[string]any{"html": html}), true
}

func htmlFromToolJSON(content string) string {
	var envelope struct {
		Parameters map[string]any `json:"parameters"`
		Arguments  map[string]any `json:"arguments"`
		Input      map[string]any `json:"input"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err != nil {
		return ""
	}
	for _, args := range []map[string]any{envelope.Parameters, envelope.Arguments, envelope.Input} {
		if html, _, ok := viz.ParseInput(args); ok {
			return html
		}
	}
	return ""
}

func extractHTMLFragment(content string) string {
	lower := strings.ToLower(content)
	for _, marker := range []string{"<canvas", "<!doctype", "<html", "<svg", "<div"} {
		idx := strings.Index(lower, marker)
		if idx < 0 {
			continue
		}
		fragment := content[idx:]
		for _, closing := range []string{"</script>", "</html>", "</svg>"} {
			if end := strings.LastIndex(strings.ToLower(fragment), closing); end >= 0 {
				return fragment[:end+len(closing)]
			}
		}
		if end := strings.LastIndex(fragment, ">"); end >= 0 {
			return fragment[:end+1]
		}
	}
	return ""
}

func newTextToolCall(name string, args map[string]any) llm.ToolCall {
	payload, _ := json.Marshal(args)
	return llm.ToolCall{
		ID:   "text_tool_" + uuid.NewString(),
		Type: "function",
		Function: llm.FunctionCall{
			Name:      name,
			Arguments: string(payload),
		},
	}
}

func shouldBufferTextToolStream(delta string) bool {
	trimmed := strings.TrimSpace(delta)
	return strings.HasPrefix(trimmed, "{") &&
		(strings.Contains(trimmed, `"name"`) || strings.Contains(trimmed, `"parameters"`) || strings.Contains(trimmed, `"arguments"`))
}
