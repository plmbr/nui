// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
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

func TestExtractTextToolCalls_ignoresProseMentioningVisualization(t *testing.T) {
	content := `I can help with coding and charts. When you ask for a chart I call show_visualization with HTML like <div>example</div>.`
	cleaned, calls := extractTextToolCalls(content, []string{"show_visualization"})
	if cleaned != content || len(calls) != 0 {
		t.Fatalf("cleaned=%q calls=%+v", cleaned, calls)
	}
}

func TestExtractTextToolCalls_brokenVisualizationJSON(t *testing.T) {
	content := `{"name": "show_visualization", "parameters": {"html": "<canvas id="chart" width="400" height="200"></canvas><script src="https://cdn.jsdelivr.net/npm/chart.js"></script>"}}`
	cleaned, calls := extractTextToolCalls(content, []string{"show_visualization"})
	if cleaned != content || len(calls) != 0 {
		t.Fatalf("cleaned=%q calls=%+v", cleaned, calls)
	}
}

func TestExtractTextToolCalls_relaxedAskUserJSON(t *testing.T) {
	content := `{"name": "ask_user", "parameters": {"title": "Math", "questions": [{"question": "What is 2+2?", "options": [{"label": "4"}]}]}}`
	cleaned, calls := extractTextToolCalls(content, []string{"ask_user"})
	if cleaned != "" {
		t.Fatalf("cleaned = %q, want empty", cleaned)
	}
	if len(calls) != 1 || calls[0].Function.Name != "ask_user" {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestLooksLikeTextToolJSON(t *testing.T) {
	if !looksLikeTextToolJSON(`{"name":"ask_user","parameters":{}}`) {
		t.Fatal("expected tool json")
	}
	if looksLikeTextToolJSON(`{"foo": "bar"}`) {
		t.Fatal("expected non-tool json to be ignored")
	}
}

func TestExtractTextToolCalls_embeddedAskUserJSON(t *testing.T) {
	content := `Sure! {"name": "ask_user", "parameters": {"message": "What is 2+2?"}}`
	cleaned, calls := extractTextToolCalls(content, []string{"ask_user"})
	if cleaned != "Sure!" {
		t.Fatalf("cleaned = %q, want %q", cleaned, "Sure!")
	}
	if len(calls) != 1 || calls[0].Function.Name != "ask_user" {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestShouldBufferTextToolStream_startsOnBrace(t *testing.T) {
	if !shouldBufferTextToolStream("{") {
		t.Fatal("expected buffering on opening brace")
	}
	if shouldBufferTextToolStream("hello") {
		t.Fatal("expected plain text not to buffer")
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
