// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
)

type codexStreamParser struct {
	seenToolStarts   map[string]struct{}
	seenToolEnds     map[string]struct{}
	seenToolResults  map[string]struct{}
}

func newCodexStreamParser() *codexStreamParser {
	return &codexStreamParser{
		seenToolStarts:  make(map[string]struct{}),
		seenToolEnds:    make(map[string]struct{}),
		seenToolResults: make(map[string]struct{}),
	}
}

func (p *codexStreamParser) handleLine(line []byte, events chan<- Event) (sessionID string, done bool) {
	if len(line) == 0 {
		return "", false
	}

	var envelope struct {
		Type     string `json:"type"`
		ThreadID string `json:"thread_id"`
		Message  string `json:"message"`
		Error    struct {
			Message string `json:"message"`
		} `json:"error"`
		Item json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		fmt.Printf("[codex stdout] %s\n", line)
		return "", false
	}

	switch envelope.Type {
	case "thread.started":
		return envelope.ThreadID, false
	case "item.started", "item.updated", "item.completed":
		p.handleItem(envelope.Type, envelope.Item, events)
	case "turn.completed":
		return "", true
	case "turn.failed":
		msg := envelope.Error.Message
		if msg == "" {
			msg = "turn failed"
		}
		events <- Event{Type: EventError, Error: msg}
	case "error":
		msg := envelope.Message
		if msg == "" {
			msg = envelope.Error.Message
		}
		if msg != "" {
			events <- Event{Type: EventError, Error: msg}
		}
	}
	return "", false
}

func (p *codexStreamParser) handleItem(eventType string, raw json.RawMessage, events chan<- Event) {
	if len(raw) == 0 || string(raw) == "null" {
		return
	}

	var item struct {
		ID        string          `json:"id"`
		Type      string          `json:"type"`
		Text      string          `json:"text"`
		Server    string          `json:"server"`
		Tool      string          `json:"tool"`
		Arguments json.RawMessage `json:"arguments"`
		Result    json.RawMessage `json:"result"`
		Error     struct {
			Message string `json:"message"`
		} `json:"error"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return
	}

	switch item.Type {
	case "agent_message":
		if eventType == "item.completed" && item.Text != "" {
			events <- Event{Type: EventText, Content: item.Text}
		}
	case "mcp_tool_call":
		p.handleMCPToolCall(eventType, item, events)
	}
}

func (p *codexStreamParser) handleMCPToolCall(eventType string, item struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Server    string          `json:"server"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
	Result    json.RawMessage `json:"result"`
	Error     struct {
		Message string `json:"message"`
	} `json:"error"`
	Status string `json:"status"`
}, events chan<- Event) {
	toolID := item.ID
	if toolID == "" {
		return
	}

	toolName := item.Tool
	if item.Server != "" {
		toolName = item.Server + "/" + item.Tool
	}

	if eventType == "item.started" {
		if _, ok := p.seenToolStarts[toolID]; ok {
			return
		}
		p.seenToolStarts[toolID] = struct{}{}
		events <- Event{
			Type:       EventToolCallStart,
			ToolCallID: toolID,
			ToolName:   toolName,
		}
		return
	}

	if eventType != "item.completed" {
		return
	}

	if _, ok := p.seenToolStarts[toolID]; !ok {
		p.seenToolStarts[toolID] = struct{}{}
		events <- Event{
			Type:       EventToolCallStart,
			ToolCallID: toolID,
			ToolName:   toolName,
		}
	}

	argsJSON := string(item.Arguments)
	if argsJSON == "" || argsJSON == "null" {
		argsJSON = "{}"
	}
	events <- Event{
		Type:       EventToolCallArgs,
		ToolCallID: toolID,
		ToolArgs:   argsJSON,
	}

	if _, ok := p.seenToolEnds[toolID]; !ok {
		p.seenToolEnds[toolID] = struct{}{}
		events <- Event{
			Type:       EventToolCallEnd,
			ToolCallID: toolID,
			ToolName:   toolName,
			ToolArgs:   argsJSON,
		}
	}

	if _, ok := p.seenToolResults[toolID]; ok {
		return
	}
	p.seenToolResults[toolID] = struct{}{}

	if item.Status == "failed" || item.Error.Message != "" {
		msg := item.Error.Message
		if msg == "" {
			msg = "tool failed"
		}
		events <- Event{
			Type:       EventToolCallResult,
			ToolCallID: toolID,
			Content:    msg,
		}
		return
	}

	if len(item.Result) > 0 && string(item.Result) != "null" {
		events <- Event{
			Type:       EventToolCallResult,
			ToolCallID: toolID,
			Content:    string(item.Result),
		}
		emitImageEvents(item.Result, "", events)
	}
}
