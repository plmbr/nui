// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"
)

func TestOpenCodeStreamParserText(t *testing.T) {
	parser := newOpenCodeStreamParser()
	events := make(chan Event, 4)
	line := []byte(`{"type":"text","part":{"type":"text","text":"Hello opencode"}}`)
	parser.handleLine(line, events)
	close(events)

	var text string
	for ev := range events {
		if ev.Type == EventText {
			text += ev.Content
		}
	}
	if text != "Hello opencode" {
		t.Fatalf("text = %q", text)
	}
}

func TestOpenCodeStreamParserToolUse(t *testing.T) {
	parser := newOpenCodeStreamParser()
	events := make(chan Event, 8)
	line := []byte(`{"type":"tool_use","part":{"callID":"call-1","tool":"read","state":{"status":"completed","input":{"path":"README.md"},"output":"# Title"}}}`)
	parser.handleLine(line, events)
	close(events)

	var start, args, end bool
	for ev := range events {
		switch ev.Type {
		case EventToolCallStart:
			start = true
			if ev.ToolCallID != "call-1" || ev.ToolName != "read" {
				t.Fatalf("start = %+v", ev)
			}
		case EventToolCallArgs:
			args = true
		case EventToolCallEnd:
			end = true
		}
	}
	if !start || !args || !end {
		t.Fatalf("start=%v args=%v end=%v", start, args, end)
	}
}
