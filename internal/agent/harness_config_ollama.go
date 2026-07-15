// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "strings"

const ollamaToolsSystemPromptAppendix = `
## Tool calling (Ollama)

Use the native tool/function calling API for every tool invocation. Do not print JSON such as {"name": "...", "parameters": {...}} in your assistant text.

For charts and visualizations, call **show_visualization** on **loop-viz** with complete self-contained HTML in the **html** field. Use valid JSON with properly escaped quotes inside HTML attribute values. Every script tag must be properly closed.

Chart.js template (copy this structure exactly — use a **canvas** element, never a div):
<script src="https://cdn.jsdelivr.net/npm/chart.js@2.9.4/dist/Chart.min.js"></script>
<canvas id="chart" width="400" height="200"></canvas>
<script>
new Chart(document.getElementById('chart').getContext('2d'), {
  type: 'bar',
  data: {
    labels: ['A', 'B', 'C'],
    datasets: [{ label: 'Series 1', data: [10, 20, 30], backgroundColor: 'rgba(54,162,235,0.5)' }]
  },
  options: { responsive: false, legend: { display: false } }
});
</script>
`

func appendOllamaToolsSystemPrompt(systemPrompt string) string {
	block := strings.TrimSpace(ollamaToolsSystemPromptAppendix)
	if block == "" {
		return systemPrompt
	}
	base := strings.TrimSpace(systemPrompt)
	if base == "" {
		return block
	}
	return base + "\n\n" + block
}
