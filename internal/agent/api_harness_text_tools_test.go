// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractTextToolCalls_validJSON(t *testing.T) {
	content := `{"name": "ask_user", "parameters": {"message": "Pick a color"}}`
	cleaned, calls := extractTextToolCalls(content, []string{"ask_user"})
	if cleaned != "" {
		t.Fatalf("cleaned = %q, want empty", cleaned)
	}
	if len(calls) != 1 || calls[0].Function.Name != "ask_user" {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestExtractTextToolCalls_ignoresUnknownTool(t *testing.T) {
	content := `{"name": "unknown_tool", "parameters": {"x": 1}}`
	cleaned, calls := extractTextToolCalls(content, []string{"ask_user"})
	if cleaned != content || len(calls) != 0 {
		t.Fatalf("cleaned=%q calls=%+v", cleaned, calls)
	}
}

func TestExtractTextToolCalls_brokenVisualizationJSON(t *testing.T) {
	content := `{"name": "show_visualization", "parameters": {"html": "<canvas id="chart" width="400" height="200"></canvas><script src="https://cdn.jsdelivr.net/npm/chart.js"></script>"}}`
	cleaned, calls := extractTextToolCalls(content, []string{"show_visualization"})
	if cleaned != "" {
		t.Fatalf("cleaned = %q, want empty", cleaned)
	}
	if len(calls) != 1 || calls[0].Function.Name != "show_visualization" {
		t.Fatalf("calls = %+v", calls)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
		t.Fatal(err)
	}
	html, _ := args["html"].(string)
	if !strings.Contains(html, "<canvas") || !strings.Contains(html, "</script>") {
		t.Fatalf("html = %q", html)
	}
}

func TestExtractHTMLFragment(t *testing.T) {
	html := extractHTMLFragment(`prefix {"html": "<canvas id="x"></canvas><script></script>"}`)
	if !strings.HasPrefix(html, "<canvas") || !strings.HasSuffix(html, "</script>") {
		t.Fatalf("html = %q", html)
	}
}

func TestStripMarkdownCodeFence(t *testing.T) {
	got := stripMarkdownCodeFence("```json\n{\"name\":\"ask_user\"}\n```")
	if got != `{"name":"ask_user"}` {
		t.Fatalf("got %q", got)
	}
}
