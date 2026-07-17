// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"

	"nui/internal/llm"
	"nui/internal/viz"
)

func toolArgsStreamUpdate(previous, next string) (delta string, changed bool) {
	next = strings.TrimSpace(next)
	if next == "" {
		return "", false
	}
	if previous == next {
		return "", false
	}
	if json.Valid([]byte(next)) {
		var obj map[string]any
		if err := json.Unmarshal([]byte(next), &obj); err == nil && len(obj) > 0 {
			return next, true
		}
	}
	if strings.HasPrefix(next, previous) {
		suffix := next[len(previous):]
		return suffix, suffix != ""
	}
	return next, true
}

func filterExecutableToolCalls(calls []llm.ToolCall) []llm.ToolCall {
	if len(calls) == 0 {
		return calls
	}
	out := make([]llm.ToolCall, 0, len(calls))
	for _, tc := range calls {
		if !viz.IsVisualizationTool(tc.Function.Name) {
			out = append(out, tc)
			continue
		}
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			continue
		}
		html, _, ok := viz.ParseInput(args)
		if !ok {
			continue
		}
		html = viz.PrepareHTML(html)
		if !viz.VisualizationHTMLReady(html) {
			continue
		}
		tc.Function.Arguments = mustSetHTMLArg(tc.Function.Arguments, html)
		out = append(out, tc)
	}
	return out
}

func toolCallVisualizationReady(tc llm.ToolCall) bool {
	if !viz.IsVisualizationTool(tc.Function.Name) {
		return true
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return false
	}
	html, _, ok := viz.ParseInput(args)
	if !ok {
		return false
	}
	html = viz.PrepareHTML(html)
	return viz.VisualizationHTMLReady(html)
}

func mustSetHTMLArg(argsJSON, html string) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return argsJSON
	}
	args["html"] = html
	out, err := json.Marshal(args)
	if err != nil {
		return argsJSON
	}
	return string(out)
}
