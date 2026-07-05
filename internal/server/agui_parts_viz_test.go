// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"

	"loop/internal/agent"
)

func TestAssistantPartAccumulator_visualization(t *testing.T) {
	acc := newAssistantPartAccumulator()
	acc.applyEvent(agent.Event{
		Type:       agent.EventToolCallStart,
		ToolCallID: "tc1",
		ToolName:   "mcp__loop-viz__show_visualization",
	}, nil)
	acc.applyEvent(agent.Event{
		Type:       agent.EventToolCallArgs,
		ToolCallID: "tc1",
		ToolArgs:   `{"html":"<canvas></canvas>","title":"Chart"}`,
	}, nil)
	acc.applyEvent(agent.Event{
		Type:       agent.EventToolCallEnd,
		ToolCallID: "tc1",
		ToolName:   "mcp__loop-viz__show_visualization",
	}, nil)

	if len(acc.parts) != 1 {
		t.Fatalf("parts = %d, want 1", len(acc.parts))
	}
	part := acc.parts[0]
	if part.VisualizationHTML != "<canvas></canvas>" {
		t.Fatalf("VisualizationHTML = %q", part.VisualizationHTML)
	}
	if part.VisualizationTitle != "Chart" {
		t.Fatalf("VisualizationTitle = %q", part.VisualizationTitle)
	}
}
