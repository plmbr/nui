// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
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
