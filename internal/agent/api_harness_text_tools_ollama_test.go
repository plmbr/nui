// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractTextToolCalls_ollamaEscapedVisualization(t *testing.T) {
	content := `{"name": "show_visualization", "parameters": {"html": "<canvas id=\\\"chart\\\" width=\\\"400\\\" height=\\\"200\\\"></canvas><script>new Chart(document.getElementById(\\\"chart\\\")_.getContext(\\\"2d\\\")_, { type: \\\"bar\\\", data: { labels: \\[\\], datasets: [{ label: \\\"Series 1\\\", data: \\[10, 20, 30\\], backgroundColor: \\\"rgba(54,162,235,0.5)\\\" }] }, options: { responsive: false, legend: { display: false } } });</script>"}}`
	cleaned, calls := extractTextToolCalls(content, []string{"show_visualization"})
	if cleaned != "" {
		t.Fatalf("cleaned = %q, want empty", cleaned)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %+v", calls)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
		t.Fatal(err)
	}
	filtered := filterExecutableToolCalls(calls)
	if len(filtered) != 1 {
		t.Fatalf("filtered = %+v", filtered)
	}
	if err := json.Unmarshal([]byte(filtered[0].Function.Arguments), &args); err != nil {
		t.Fatal(err)
	}
	html, _ := args["html"].(string)
	if strings.Contains(html, ")_.") || strings.Contains(html, `\"`) {
		t.Fatalf("html not repaired: %q", html)
	}
	if !strings.Contains(html, "/vendor/chart.min.js") {
		t.Fatalf("html missing chart.js: %q", html)
	}
}
