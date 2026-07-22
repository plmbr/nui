// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"

	"nui/internal/llm"
)

func TestAccumulateToolCallArgs_incremental(t *testing.T) {
	current := ""
	for _, chunk := range []string{`{"message": `, `"hello`, ` world"}`} {
		current = accumulateToolCallArgs(current, chunk)
	}
	if current != `{"message": "hello world"}` {
		t.Fatalf("incremental = %q", current)
	}
}

func TestAccumulateToolCallArgs_cumulative(t *testing.T) {
	current := ""
	chunks := []string{
		`{"message": "hel`,
		`{"message": "hello`,
		`{"message": "hello world"}`,
	}
	for _, chunk := range chunks {
		current = accumulateToolCallArgs(current, chunk)
	}
	if current != `{"message": "hello world"}` {
		t.Fatalf("cumulative = %q", current)
	}
}

func TestAccumulateToolCallArgs_snapshotReplace(t *testing.T) {
	prev := `{"html":"<canvas id=","title":""}`
	next := `{"html":"<canvas id=\"c\"></canvas><script>new Chart()</script>","title":"Sales"}`
	got := accumulateToolCallArgs(prev, next)
	if got != next {
		t.Fatalf("snapshot replace = %q", got)
	}
}

func TestToolArgsStreamUpdate_cumulative(t *testing.T) {
	prev := `{"html":"`
	next := `{"html":"<c`
	delta, changed := toolArgsStreamUpdate(prev, next)
	if !changed || delta != `<c` {
		t.Fatalf("cumulative delta = %q changed=%v", delta, changed)
	}
}

func TestToolArgsStreamUpdate_snapshot(t *testing.T) {
	prev := `{"html":"<canvas id=","title":""}`
	next := `{"html":"<canvas id=\"c\"></canvas><script>new Chart()</script>","title":"Sales"}`
	delta, changed := toolArgsStreamUpdate(prev, next)
	if !changed || delta != next {
		t.Fatalf("snapshot delta = %q changed=%v", delta, changed)
	}
	_, changed = toolArgsStreamUpdate(next, next)
	if changed {
		t.Fatal("expected unchanged snapshot to emit no delta")
	}
}

func TestFilterExecutableToolCalls_skipsPartialViz(t *testing.T) {
	calls := []llm.ToolCall{
		{
			ID: "1",
			Function: llm.FunctionCall{
				Name:      "show_visualization",
				Arguments: `{"html":"<canvas id=","title":""}`,
			},
		},
		{
			ID: "2",
			Function: llm.FunctionCall{
				Name:      "show_visualization",
				Arguments: `{"html":"<script src=\"https://cdn.jsdelivr.net/npm/chart.js\"></script><canvas id=\"c\"></canvas><script>new Chart(document.getElementById('c'), {type:'bar'});","title":"Sales"}`,
			},
		},
		{
			ID: "3",
			Function: llm.FunctionCall{
				Name:      "ask_user",
				Arguments: `{"question":"Pick one"}`,
			},
		},
	}
	filtered := filterExecutableToolCalls(calls)
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}
	if filtered[0].ID != "2" || filtered[1].ID != "3" {
		t.Fatalf("unexpected filtered ids: %#v", filtered)
	}
}
