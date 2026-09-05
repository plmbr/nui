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
	// Concatenated objects: take the first valid action.
	action, ok = parseChairAction(
		`{"action":"delegate","agent":"a","prompt":"p"}{"action":"finish","answer":"line1\nline2"}`,
	)
	if !ok || action.Action != "delegate" || action.Agent != "a" {
		t.Fatalf("concat: %+v ok=%v", action, ok)
	}
	action, ok = parseChairAction(`{"action":"finish","answer":"line1\nline2"}`)
	if !ok || action.Answer != "line1\nline2" {
		t.Fatalf("multiline answer: %+v ok=%v", action, ok)
	}
}

func TestCollectingEventsMuteText(t *testing.T) {
	upstream := make(chan Event, 8)
	collecting := &collectingEvents{upstream: upstream, muteText: true}
	events := collecting.start()
	events <- Event{Type: EventText, Content: `{"action":"finish","answer":"hi"}`}
	events <- Event{Type: EventError, Error: "boom"}
	collecting.finish()
	close(upstream)

	var texts, errors int
	for ev := range upstream {
		switch ev.Type {
		case EventText:
			texts++
		case EventError:
			errors++
		}
	}
	if texts != 0 {
		t.Fatalf("muted text forwarded %d times", texts)
	}
	if errors != 1 {
		t.Fatalf("errors = %d, want 1", errors)
	}
	if collecting.text == "" {
		t.Fatal("expected text collected while muted")
	}
}

func TestSubAgentsChairInstructions(t *testing.T) {
	s := subAgentsChairInstructions([]resolvedCouncilMember{{id: "a", label: "A"}})
	if !strings.Contains(s, "run_sub_agent") || !strings.Contains(s, "a") {
		t.Fatalf("instructions = %q", s)
	}
	if !strings.Contains(s, "exactly ONE") || !strings.Contains(s, "never fan out") {
		t.Fatalf("expected one-at-a-time guidance, got %q", s)
	}
	turn := subAgentsTurnInstruction(1, 20)
	if !strings.Contains(turn, "ONE agent") {
		t.Fatalf("turn instruction = %q", turn)
	}
}
