// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package viz

import (
	"strings"
	"testing"
)

const ollamaEscapedSample = `<canvas id=\"chart\" width=\"400\" height=\"200\"></canvas><script>new Chart(document.getElementById(\"chart\")_.getContext(\"2d\")_, { type: \"bar\", data: { labels: \[\], datasets: [{ label: \"Series 1\", data: \[10, 20, 30\], backgroundColor: \"rgba(54,162,235,0.5)\" }] }, options: { responsive: false, legend: { display: false } } });</script>`

const ollamaMixedQuoteSample = `<canvas id=\"chart\" width=\"400\" height=\"200\"></canvas><script>new Chart(document.getElementById('chart').getContext('2d'), {
  type: 'bar',
  data: {
    labels: ['A', 'B', 'C'],
    datasets: [{ label: 'Series 1', data: [10, 20, 30], backgroundColor: 'rgba(54,162,235,0.5)' }]
  },
  options: { responsive: false, legend: { display: false } }
});</script>`

func TestPrepareHTML_repairsOllamaEscapedHTML(t *testing.T) {
	prepared := PrepareHTML(ollamaEscapedSample)
	if strings.Contains(prepared, `\\"`) || strings.Contains(prepared, ")_.") {
		t.Fatalf("expected escaped corruption to be repaired, got %q", prepared)
	}
	if !strings.Contains(prepared, `id="chart"`) {
		t.Fatalf("expected valid chart canvas id, got %q", prepared)
	}
	if !strings.Contains(prepared, ").getContext(") {
		t.Fatalf("expected repaired getContext call, got %q", prepared)
	}
	if !strings.Contains(prepared, "/vendor/chart.min.js") {
		t.Fatalf("expected bundled chart.js, got %q", prepared)
	}
	if !VisualizationHTMLReady(prepared) {
		t.Fatal("expected repaired ollama chart to be ready")
	}
}

func TestPrepareHTML_repairsOllamaMixedQuoteHTML(t *testing.T) {
	prepared := PrepareHTML(ollamaMixedQuoteSample)
	if strings.Contains(prepared, `\"`) {
		t.Fatalf("expected escaped quotes to be repaired, got %q", prepared)
	}
	if !strings.Contains(prepared, "/vendor/chart.min.js") {
		t.Fatalf("expected bundled chart.js, got %q", prepared)
	}
	if !VisualizationHTMLReady(prepared) {
		t.Fatal("expected mixed-quote ollama chart to be ready")
	}
}
