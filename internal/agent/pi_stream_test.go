// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"
)

func TestPiStreamParserTextMessage(t *testing.T) {
	parser := newPiStreamParser()
	events := make(chan Event, 4)
	line := []byte(`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Hi from pi"}}`)
	parser.handleLine(line, events)
	close(events)

	var text string
	for ev := range events {
		if ev.Type == EventText {
			text += ev.Content
		}
	}
	if text != "Hi from pi" {
		t.Fatalf("text = %q", text)
	}
}

func TestPiStreamParserToolExecutionStart(t *testing.T) {
	parser := newPiStreamParser()
	events := make(chan Event, 4)
	line := []byte(`{"type":"tool_execution_start","toolCallId":"tc-1","toolName":"Bash"}`)
	parser.handleLine(line, events)
	close(events)

	var got Event
	for ev := range events {
		got = ev
	}
	if got.Type != EventToolCallStart || got.ToolCallID != "tc-1" || got.ToolName != "Bash" {
		t.Fatalf("event = %+v", got)
	}
}

func TestPiStreamParserDuplicateToolStartIgnored(t *testing.T) {
	parser := newPiStreamParser()
	events := make(chan Event, 8)
	line := []byte(`{"type":"tool_execution_start","toolCallId":"tc-1","toolName":"Bash"}`)
	parser.handleLine(line, events)
	parser.handleLine(line, events)
	close(events)

	count := 0
	for ev := range events {
		if ev.Type == EventToolCallStart {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("tool starts = %d", count)
	}
}
