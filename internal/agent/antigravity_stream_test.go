// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"
)

func TestAntigravityStreamParserTextDelta(t *testing.T) {
	parser := newAntigravityStreamParser()
	events := make(chan Event, 4)
	line := []byte(`{"event":"step_update","step_update":{"conversation_id":"c1","step_index":2,"state":"ACTIVE","step_type":"agent_response","text_delta":"Hello"}}`)
	parser.handleLine(line, events)
	close(events)

	var text string
	for ev := range events {
		if ev.Type == EventText {
			text += ev.Content
		}
	}
	if text != "Hello" {
		t.Fatalf("text = %q", text)
	}
	if parser.conversationID != "c1" {
		t.Fatalf("conversationID = %q", parser.conversationID)
	}
}

func TestAntigravityStreamParserResultFallback(t *testing.T) {
	parser := newAntigravityStreamParser()
	events := make(chan Event, 4)
	line := []byte(`{"event":"result","result":{"conversation_id":"c2","status":"SUCCESS","response":"done\n"}}`)
	parser.handleLine(line, events)
	close(events)

	var text string
	for ev := range events {
		if ev.Type == EventText {
			text += ev.Content
		}
	}
	if text != "done\n" {
		t.Fatalf("text = %q", text)
	}
	if !parser.turnDone {
		t.Fatal("expected turnDone")
	}
	if parser.conversationID != "c2" {
		t.Fatalf("conversationID = %q", parser.conversationID)
	}
}

func TestAntigravityStreamParserToolStep(t *testing.T) {
	parser := newAntigravityStreamParser()
	events := make(chan Event, 8)
	line := []byte(`{"event":"step_update","step_update":{"conversation_id":"c3","step_index":4,"state":"DONE","step_type":"tool","tool_name":"run_command","tool_info":{"name":"run_command","parameters":{"CommandLine":"echo hi"},"output":"hi\n"}}}`)
	parser.handleLine(line, events)
	close(events)

	var start, end, result bool
	for ev := range events {
		switch ev.Type {
		case EventToolCallStart:
			start = ev.ToolCallID == "agy-4" && ev.ToolName == "run_command"
		case EventToolCallEnd:
			end = ev.ToolCallID == "agy-4"
		case EventToolCallResult:
			result = ev.Content == "hi\n"
		}
	}
	if !start || !end || !result {
		t.Fatalf("start=%v end=%v result=%v", start, end, result)
	}
}

func TestAntigravityStreamParserErrorStatus(t *testing.T) {
	parser := newAntigravityStreamParser()
	events := make(chan Event, 4)
	line := []byte(`{"event":"result","result":{"conversation_id":"c4","status":"ERROR","response":"","error":"authentication required"}}`)
	parser.handleLine(line, events)
	close(events)

	for range events {
		t.Fatal("error result should not emit events until enrichment")
	}
	if !parser.turnDone || parser.turnError != "authentication required" {
		t.Fatalf("turnDone=%v turnError=%q", parser.turnDone, parser.turnError)
	}
}

func TestEnrichAntigravityErrorUsesQuotaStderr(t *testing.T) {
	stderr := "noise\nError 429, Message: Quota exceeded for metric: foo, model: gemini-3.6-flash-medium\nmore"
	got := enrichAntigravityError("Agent execution terminated due to error.", stderr)
	if !strings.Contains(got, "Quota exceeded") {
		t.Fatalf("enriched = %q", got)
	}
}
