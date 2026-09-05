// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"
)

func TestParseChairAction(t *testing.T) {
	action, ok := parseChairAction(`{"action":"delegate","agent":"coder","prompt":"fix the bug"}`)
	if !ok || action.Action != "delegate" || action.Agent != "coder" {
		t.Fatalf("got %+v ok=%v", action, ok)
	}
	action, ok = parseChairAction(`{"action":"finish","answer":"done"}`)
	if !ok || action.Action != "finish" || action.Answer != "done" {
		t.Fatalf("finish: %+v ok=%v", action, ok)
	}
	action, ok = parseChairAction("Decision:\n{\"action\":\"delegate\",\"agent\":\"x\",\"prompt\":\"y\"}\nThanks")
	if !ok || action.Agent != "x" {
		t.Fatalf("embedded: %+v ok=%v", action, ok)
	}
	if _, ok := parseChairAction("just a free-form answer with no json"); ok {
		t.Fatal("expected no action")
	}
}

func TestSubAgentsChairInstructions(t *testing.T) {
	s := subAgentsChairInstructions([]resolvedCouncilMember{{id: "a", label: "A"}})
	if !strings.Contains(s, "run_sub_agent") || !strings.Contains(s, "a") {
		t.Fatalf("instructions = %q", s)
	}
}
