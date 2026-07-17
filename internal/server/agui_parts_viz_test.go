// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"strings"
	"testing"

	"nui/internal/agent"
)

func TestAssistantPartAccumulator_visualization(t *testing.T) {
	chartHTML := `<canvas id="c"></canvas><script>new Chart(document.getElementById("c"))</script>`
	toolArgsBytes, err := json.Marshal(map[string]string{"html": chartHTML, "title": "Chart"})
	if err != nil {
		t.Fatal(err)
	}
	toolArgs := string(toolArgsBytes)
	acc := newAssistantPartAccumulator()
	acc.applyEvent(agent.Event{
		Type:       agent.EventToolCallStart,
		ToolCallID: "tc1",
		ToolName:   "mcp__nui-viz__show_visualization",
	}, nil)
	acc.applyEvent(agent.Event{
		Type:       agent.EventToolCallArgs,
		ToolCallID: "tc1",
		ToolArgs:   toolArgs,
	}, nil)
	acc.applyEvent(agent.Event{
		Type:       agent.EventToolCallEnd,
		ToolCallID: "tc1",
		ToolName:   "mcp__nui-viz__show_visualization",
		ToolArgs:   toolArgs,
	}, nil)

	if len(acc.parts) != 1 {
		t.Fatalf("parts = %d, want 1", len(acc.parts))
	}
	part := acc.parts[0]
	if !strings.Contains(part.VisualizationHTML, chartHTML) {
		t.Fatalf("VisualizationHTML = %q", part.VisualizationHTML)
	}
	if part.VisualizationTitle != "Chart" {
		t.Fatalf("VisualizationTitle = %q", part.VisualizationTitle)
	}
}
