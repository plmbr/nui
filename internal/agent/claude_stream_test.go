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

func TestClaudeStreamParserSkipsAssistantTextReplayAfterToolUse(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 8)

	parser.handleLine([]byte(`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tool-1","name":"AskUserQuestion"}}}`), events)
	parser.handleLine([]byte(`{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`), events)
	parser.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"old response from a prior turn"}]}}`), events)
	close(events)

	for ev := range events {
		if ev.Type == EventText {
			t.Fatalf("unexpected text replay: %q", ev.Content)
		}
	}
}

func TestClaudeStreamParserAssistantTextBeforeTools(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 4)

	parser.handleLine([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]}}`), events)
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
}

func TestClaudeStreamParserCompletesTurnOnMessageStopEndTurn(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 4)

	lines := [][]byte{
		[]byte(`{"type":"stream_event","session_id":"sess-1","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}}}`),
		[]byte(`{"type":"stream_event","session_id":"sess-1","event":{"type":"message_stop"}}`),
	}
	for _, line := range lines {
		parser.handleLine(line, events)
	}
	close(events)

	var done Event
	for ev := range events {
		if ev.Type == EventDone {
			done = ev
		}
	}
	if done.Type != EventDone {
		t.Fatalf("expected EventDone, got %+v", done)
	}
	if done.SessionID != "sess-1" {
		t.Fatalf("session id = %q", done.SessionID)
	}
	if !parser.completedTurn() {
		t.Fatal("expected completed turn")
	}
}

func TestClaudeStreamParserDoesNotCompleteWithPendingToolWork(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 8)

	parser.handleLine([]byte(`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tool-1","name":"AskUserQuestion"}}}`), events)
	parser.handleLine([]byte(`{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`), events)
	parser.handleLine([]byte(`{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}}}`), events)
	parser.handleLine([]byte(`{"type":"stream_event","event":{"type":"message_stop"}}`), events)
	close(events)

	for ev := range events {
		if ev.Type == EventDone {
			t.Fatal("expected no EventDone while tool is pending")
		}
	}
	if parser.completedTurn() {
		t.Fatal("turn should not complete while tool result is pending")
	}
}

func TestClaudeStreamParserCompletesTurnOnAssistantSnapshotAfterToolResult(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 8)

	lines := [][]byte{
		[]byte(`{"type":"stream_event","session_id":"sess-1","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tool-1","name":"AskUserQuestion"}}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`),
		[]byte(`{"type":"user","parent_tool_use_id":"tool-1","tool_use_result":{"answers":["animal"]}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Here's a joke."}}}`),
		[]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Here's a joke."}]}}`),
		[]byte(`{"type":"stream_event","session_id":"sess-1","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}}}`),
		[]byte(`{"type":"stream_event","session_id":"sess-1","event":{"type":"message_stop"}}`),
	}
	for _, line := range lines {
		parser.handleLine(line, events)
	}
	close(events)

	var done Event
	for ev := range events {
		if ev.Type == EventDone {
			done = ev
		}
	}
	if done.Type != EventDone {
		t.Fatal("expected EventDone after post-tool assistant snapshot")
	}
}

func TestClaudeStreamParserDoesNotCompleteBeforeFollowUpToolUse(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 16)

	lines := [][]byte{
		[]byte(`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tool-1","name":"run_query"}}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`),
		[]byte(`{"type":"user","parent_tool_use_id":"tool-1","tool_use_result":{"rows":[]}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Let me try a different table:"}}}`),
		[]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Let me try a different table:"}]}}`),
	}
	for _, line := range lines {
		parser.handleLine(line, events)
	}
	if parser.completedTurn() {
		t.Fatal("turn should not complete before follow-up tool_use")
	}

	parser.handleLine([]byte(`{"type":"stream_event","event":{"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"tool-2","name":"discover_tables"}}}`), events)
	parser.handleLine([]byte(`{"type":"stream_event","event":{"type":"content_block_stop","index":2}}`), events)
	close(events)

	var toolStarts int
	for ev := range events {
		if ev.Type == EventToolCallStart && ev.ToolCallID == "tool-2" {
			toolStarts++
		}
		if ev.Type == EventDone {
			t.Fatal("should not emit EventDone before turn actually ends")
		}
	}
	if toolStarts != 1 {
		t.Fatalf("expected follow-up tool start, got %d", toolStarts)
	}
}

func TestClaudeStreamParserSubagentMessageStopDoesNotCompleteParentTurn(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 16)

	lines := [][]byte{
		[]byte(`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"task-1","name":"Agent"}}}`),
		[]byte(`{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`),
		[]byte(`{"type":"stream_event","parent_tool_use_id":"task-1","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Searching files"}}}`),
		[]byte(`{"type":"stream_event","parent_tool_use_id":"task-1","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}}}`),
		[]byte(`{"type":"stream_event","parent_tool_use_id":"task-1","event":{"type":"message_stop"}}`),
	}
	for _, line := range lines {
		parser.handleLine(line, events)
	}
	close(events)

	var text string
	var done bool
	for ev := range events {
		if ev.Type == EventText && ev.ParentToolCallID == "task-1" {
			text += ev.Content
		}
		if ev.Type == EventDone {
			done = true
		}
	}
	if text != "Searching files" {
		t.Fatalf("subagent text = %q", text)
	}
	if done {
		t.Fatal("subagent message_stop should not complete parent turn")
	}
	if parser.completedTurn() {
		t.Fatal("parent turn should not be marked complete")
	}
}

func TestClaudeStreamParserSubagentToolEventsScopedToParent(t *testing.T) {
	parser := newClaudeStreamParser()
	events := make(chan Event, 16)

	lines := [][]byte{
		[]byte(`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"task-1","name":"Agent"}}}`),
		[]byte(`{"type":"stream_event","parent_tool_use_id":"task-1","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"inner-1","name":"Read"}}}`),
		[]byte(`{"type":"stream_event","parent_tool_use_id":"task-1","event":{"type":"content_block_stop","index":0}}`),
	}
	for _, line := range lines {
		parser.handleLine(line, events)
	}
	close(events)

	var innerStart bool
	for ev := range events {
		if ev.Type == EventToolCallStart && ev.ToolCallID == "inner-1" && ev.ParentToolCallID == "task-1" {
			innerStart = true
		}
	}
	if !innerStart {
		t.Fatal("expected scoped subagent tool start event")
	}
	if parser.hasPendingToolWork() != true {
		t.Fatal("parent task should still be pending")
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
