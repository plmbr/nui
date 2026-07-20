// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"
)

func TestCodexStreamParserThreadStarted(t *testing.T) {
	parser := newCodexStreamParser()
	events := make(chan Event, 4)
	sid, done := parser.handleLine([]byte(`{"type":"thread.started","thread_id":"thread-1"}`), events)
	close(events)
	if sid != "thread-1" || done {
		t.Fatalf("sid=%q done=%v", sid, done)
	}
}

func TestCodexStreamParserAgentMessage(t *testing.T) {
	parser := newCodexStreamParser()
	events := make(chan Event, 4)
	line := []byte(`{"type":"item.completed","item":{"type":"agent_message","text":"Hello from codex"}}`)
	parser.handleLine(line, events)
	close(events)

	var text string
	for ev := range events {
		if ev.Type == EventText {
			text += ev.Content
		}
	}
	if text != "Hello from codex" {
		t.Fatalf("text = %q", text)
	}
}

func TestCodexStreamParserTurnFailed(t *testing.T) {
	parser := newCodexStreamParser()
	events := make(chan Event, 4)
	parser.handleLine([]byte(`{"type":"turn.failed","error":{"message":"auth failed"}}`), events)
	close(events)

	var got Event
	for ev := range events {
		got = ev
	}
	if got.Type != EventError || got.Error != "auth failed" {
		t.Fatalf("event = %+v", got)
	}
}

func TestCodexStreamParserTurnCompleted(t *testing.T) {
	parser := newCodexStreamParser()
	events := make(chan Event, 4)
	_, done := parser.handleLine([]byte(`{"type":"turn.completed"}`), events)
	close(events)
	if !done {
		t.Fatal("expected done")
	}
}
