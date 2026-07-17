// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"

	"nui/internal/llm"
	"github.com/google/uuid"
	"nui/internal/viz"
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

	if call, ok := parseTextToolCallRelaxed(trimmed, availableTools); ok {
		if obj, ok := findEmbeddedJSONObject(trimmed); ok {
			return stripToolCallText(content, obj), []llm.ToolCall{call}
		}
		return stripToolCallText(content, trimmed), []llm.ToolCall{call}
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
	return textToolCallFromEnvelope(envelope.Name, envelope.Parameters, envelope.Arguments, envelope.Input, availableTools)
}

func parseTextToolCallRelaxed(content string, availableTools []string) (llm.ToolCall, bool) {
	name, ok := extractJSONStringField(content, "name")
	if !ok {
		return llm.ToolCall{}, false
	}
	for _, key := range []string{"parameters", "arguments", "input"} {
		if args, ok := extractJSONObjectField(content, key); ok && len(args) > 0 {
			return textToolCallFromEnvelope(name, args, nil, nil, availableTools)
		}
	}
	return llm.ToolCall{}, false
}

func textToolCallFromEnvelope(name string, parameters, arguments, input map[string]any, availableTools []string) (llm.ToolCall, bool) {
	name = strings.TrimSpace(name)
	if !toolNameAllowed(name, availableTools) {
		return llm.ToolCall{}, false
	}
	args := parameters
	if len(args) == 0 {
		args = arguments
	}
	if len(args) == 0 {
		args = input
	}
	if len(args) == 0 {
		return llm.ToolCall{}, false
	}
	resolved := resolveTextToolName(name, availableTools)
	return newTextToolCall(resolved, args), true
}

func resolveTextToolName(name string, availableTools []string) string {
	name = strings.TrimSpace(name)
	for _, candidate := range availableTools {
		if candidate == name {
			return candidate
		}
		if viz.BareToolName(candidate) == viz.BareToolName(name) {
			return candidate
		}
	}
	return name
}

func extractJSONStringField(content, field string) (string, bool) {
	pattern := `"` + field + `"`
	idx := strings.Index(content, pattern)
	if idx < 0 {
		return "", false
	}
	rest := strings.TrimSpace(content[idx+len(pattern):])
	if !strings.HasPrefix(rest, ":") {
		return "", false
	}
	rest = strings.TrimSpace(rest[1:])
	if len(rest) < 2 || rest[0] != '"' {
		return "", false
	}
	var b strings.Builder
	escape := false
	for i := 1; i < len(rest); i++ {
		c := rest[i]
		if escape {
			b.WriteByte(c)
			escape = false
			continue
		}
		if c == '\\' {
			escape = true
			continue
		}
		if c == '"' {
			return b.String(), true
		}
		b.WriteByte(c)
	}
	return "", false
}

func extractJSONObjectField(content, field string) (map[string]any, bool) {
	pattern := `"` + field + `"`
	idx := strings.Index(content, pattern)
	if idx < 0 {
		return nil, false
	}
	rest := content[idx+len(pattern):]
	brace := strings.Index(rest, "{")
	if brace < 0 {
		return nil, false
	}
	obj, ok := extractBalancedJSONObject(rest[brace:])
	if !ok {
		return nil, false
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(obj), &out); err != nil || len(out) == 0 {
		return nil, false
	}
	return out, true
}

func findEmbeddedJSONObject(content string) (string, bool) {
	idx := strings.Index(content, "{")
	if idx < 0 {
		return "", false
	}
	return extractBalancedJSONObject(content[idx:])
}

func findEmbeddedJSONObjectOrEmpty(content string) string {
	if obj, ok := findEmbeddedJSONObject(content); ok {
		return obj
	}
	return ""
}

func extractBalancedJSONObject(s string) (string, bool) {
	if s == "" || s[0] != '{' {
		return "", false
	}
	depth := 0
	inString := false
	escape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[:i+1], true
			}
		}
	}
	return "", false
}

func extractVisualizationTextToolCall(content string, availableTools []string) (llm.ToolCall, bool) {
	if !toolNameAllowed(viz.ToolName, availableTools) {
		return llm.ToolCall{}, false
	}
	trimmed := strings.TrimSpace(content)
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(trimmed, "{") || !strings.Contains(lower, `"`+viz.ToolName+`"`) {
		return llm.ToolCall{}, false
	}
	html := htmlFromToolJSON(trimmed)
	if html == "" {
		html = extractHTMLFragment(trimmed)
	}
	html = strings.TrimSpace(html)
	if html == "" {
		return llm.ToolCall{}, false
	}
	html = viz.PrepareHTML(html)
	if !viz.VisualizationHTMLReady(html) {
		return llm.ToolCall{}, false
	}
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
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, "{") && looksLikeTextToolJSON(trimmed) {
		return true
	}
	if strings.HasPrefix(trimmed, "{") {
		return true
	}
	if strings.HasPrefix(trimmed, "```") && strings.Contains(trimmed, "{") {
		return true
	}
	return false
}

func looksLikeTextToolJSON(s string) bool {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "{") {
		return false
	}
	lower := strings.ToLower(trimmed)
	return strings.Contains(lower, `"name"`) &&
		(strings.Contains(lower, `"parameters"`) || strings.Contains(lower, `"arguments"`) || strings.Contains(lower, `"input"`))
}
