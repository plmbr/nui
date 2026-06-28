// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"
)

func TestClaudeStreamParserResultErrorUsesResultField(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 4)

	line := []byte(`{"type":"result","is_error":true,"result":"Not logged in · Please run /login","session_id":"sess-1"}`)
	parser.handleLine(line, events)
	close(events)

	var got []Event
	for ev := range events {
		got = append(got, ev)
	}
	if len(got) != 1 {
		t.Fatalf("events = %d, want 1", len(got))
	}
	if got[0].Type != EventError {
		t.Fatalf("type = %v, want EventError", got[0].Type)
	}
	if got[0].Error != "Not logged in · Please run /login" {
		t.Fatalf("error = %q", got[0].Error)
	}
}

func TestClaudeStreamParserTextSepAfterToolCall(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 16)

	lines := [][]byte{
		[]byte(`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"I'll read the file."}}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"tool-1","name":"Read"}}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_stop","index":1}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_start","index":2,"content_block":{"type":"text"}}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_delta","index":2,"delta":{"type":"text_delta","text":"The file contains code."}}}`),
	}
	for _, line := range lines {
		parser.handleLine(line, events)
	}
	close(events)

	var text strings.Builder
	for ev := range events {
		if ev.Type == EventText {
			text.WriteString(ev.Content)
		}
	}
	got := text.String()
	want := "I'll read the file.\n\nThe file contains code."
	if got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
}
