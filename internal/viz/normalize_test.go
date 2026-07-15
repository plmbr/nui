// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package viz

import (
	"strings"
	"testing"

	"loop/internal/model"
)

func TestHTMLMatches(t *testing.T) {
	a := "<!DOCTYPE html><html><body><canvas id=\"c\"></canvas></body></html>"
	b := "<!DOCTYPE html>\n<html><body><canvas id=\"c\"></canvas></body></html>"
	if !HTMLMatches(a, b) {
		t.Fatal("expected whitespace-normalized HTML to match")
	}
	if HTMLMatches(a, "<html><body><p>text</p></body></html>") {
		t.Fatal("expected different HTML to not match")
	}
}

func TestNormalizeParts_dedupesWriteWhenShowVisualizationPresent(t *testing.T) {
	html := `<canvas id="c"></canvas><script>new Chart(document.getElementById("c"))</script>`
	parts := []model.ChatMessagePart{
		{
			Type:     "tool",
			ToolName: "Write",
			ToolArgs: map[string]any{"content": html, "file_path": "/tmp/chart.html"},
			VisualizationHTML: html,
		},
		{
			Type:     "tool",
			ToolName: "mcp__loop-viz__show_visualization",
			ToolArgs: map[string]any{"html": html, "title": "Chart"},
			VisualizationHTML: html,
		},
	}
	out := NormalizeParts(parts)
	if out[0].VisualizationHTML != "" {
		t.Fatalf("Write part should drop duplicate viz html, got %q", out[0].VisualizationHTML)
	}
	if out[1].VisualizationHTML == "" {
		t.Fatal("show_visualization part should keep viz html")
	}
}

func TestNormalizeParts_enrichesFromToolArgs(t *testing.T) {
	html := `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="120"><rect width="120" height="120"/></svg>`
	out := NormalizeParts([]model.ChatMessagePart{{
		Type:     "tool",
		ToolName: "Write",
		ToolArgs: map[string]any{"content": html},
	}})
	if out[0].VisualizationHTML == "" {
		t.Fatal("expected VisualizationHTML to be set")
	}
	if !strings.Contains(out[0].VisualizationHTML, "<svg") {
		t.Fatalf("VisualizationHTML = %q", out[0].VisualizationHTML)
	}
}
