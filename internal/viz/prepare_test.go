// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package viz

import (
	"strings"
	"testing"
)

func TestPrepareHTML_closesUnclosedScript(t *testing.T) {
	raw := `<script src="https://cdn.jsdelivr.net/npm/chart.js@2.9.4/dist/Chart.min.js"></script>` +
		`<canvas id="myChart" width="400" height="200"></canvas>` +
		`<script>var ctx = document.getElementById('myChart').getContext('2d');new Chart(ctx, {type: 'bar'});`
	prepared := PrepareHTML(raw)
	if !ScriptsBalanced(prepared) {
		t.Fatalf("expected balanced scripts after prepare, got %q", prepared)
	}
	if !VisualizationHTMLReady(prepared) {
		t.Fatal("expected prepared chart html to be ready")
	}
	if chartJSScriptSrcRE.MatchString(prepared) {
		t.Fatal("expected chart.js cdn to be rewritten")
	}
	if !containsAll(prepared, "<!DOCTYPE html>", "<body>", "/vendor/chart.min.js", "</script>") {
		t.Fatalf("prepared html missing expected parts: %q", prepared)
	}
}

func TestPrepareHTML_injectsChartJSWhenMissing(t *testing.T) {
	raw := `<canvas id="chart" width="400" height="200"></canvas><script>new Chart(document.getElementById('chart').getContext('2d'),{type:'bar',data:{labels:['A'],datasets:[{data:[1]}]}});</script>`
	prepared := PrepareHTML(raw)
	if !strings.Contains(prepared, "/vendor/chart.min.js") {
		t.Fatalf("expected chart.js injection, got %q", prepared)
	}
}

func TestPrepareHTML_rebuildsBrokenOllamaChart(t *testing.T) {
	raw := `<script src="https://cdn.jsdelivr.net/npm/chart.js@2.9.4/dist/Chart.min.js"></script>` +
		`<div id="line-chart"></div>` +
		`<script>var ctx = document.getElementById("line-chart").getContext('2d');new Chart(ctx, {type: 'line', data: {labels: ['January', 'February', 'March'], datasets: [{'label': 'Series 1', 'data': [12, 19, 3], 'backgroundColor': 'rgba(255,99,132,1)', 'borderColor': 'rgba(255,99,132,1),0,2)}]}})`
	prepared := PrepareHTML(raw)
	if !contains(prepared, `<canvas id="nui-chart"`) {
		t.Fatalf("expected rebuilt canvas chart, got %q", prepared)
	}
	if !contains(prepared, "January") || !contains(prepared, "[12,19,3]") {
		t.Fatalf("expected extracted labels/data in rebuilt chart, got %q", prepared)
	}
	if !contains(prepared, "plugins:{legend:{display:false}}") {
		t.Fatalf("expected v4 plugins legend options in rebuilt chart, got %q", prepared)
	}
}

func TestPrepareHTML_injectsChartErrorHandlerOnceBeforeInlineChart(t *testing.T) {
	raw := `<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.4/dist/chart.umd.min.js"></script>` +
		`<canvas id="c"></canvas>` +
		`<script>new Chart(document.getElementById('c'),{type:'bar'});</script>`
	once := PrepareHTML(raw)
	twice := PrepareHTML(once)
	if strings.Count(twice, chartErrorHandlerMarker) != 1 {
		t.Fatalf("expected one chart error handler, got %q", twice)
	}
	chartIdx := strings.Index(strings.ToLower(twice), "new chart")
	handlerIdx := strings.Index(twice, chartErrorHandlerMarker)
	if handlerIdx < 0 || chartIdx < 0 || handlerIdx > chartIdx {
		t.Fatalf("expected error handler before inline chart script, got %q", twice)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !contains(s, part) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
